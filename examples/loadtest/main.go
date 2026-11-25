// Command loadtest runs a CSMS and many simulated charging stations in one
// process and reports throughput and latency.
//
//	go run ./examples/loadtest -stations 1000 -duration 30s
//	go run ./examples/loadtest -stations 1000 -version 2.0.1
//
// Every station connects over a real loopback WebSocket, boots, sends heartbeats
// and runs short charging sessions. The CSMS answers with trivial handlers, so
// the numbers measure the SDK and the network stack, not your application.
//
// 1,000 stations need about 2,000 sockets; raise the limit first if needed
// (ulimit -n 10000). The CSMS and the stations share one process, so use
// the figures to compare runs and versions, not as an absolute capacity.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var opts options
	flag.IntVar(&opts.stations, "stations", 1000, "number of simulated charging stations")
	flag.DurationVar(&opts.duration, "duration", 30*time.Second, "how long to run after all stations are connected")
	flag.StringVar(&opts.version, "version", "1.6", "OCPP version: 1.6 or 2.0.1")
	flag.DurationVar(&opts.heartbeat, "heartbeat", time.Second, "interval between heartbeats of each station")
	flag.DurationVar(&opts.session, "session-every", 5*time.Second, "interval between charging sessions of each station (0 disables them)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, opts, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

type options struct {
	stations           int
	duration           time.Duration
	version            string
	heartbeat, session time.Duration
}

// A central is the version-specific CSMS under test.
type central interface {
	serve(ln net.Listener) error
	shutdown(ctx context.Context) error
}

// A simulated is one version-specific charging station.
type simulated interface {
	run(ctx context.Context)
	waitConnected(ctx context.Context) error
	boot(ctx context.Context) error
	heartbeat(ctx context.Context) error
	chargingSession(ctx context.Context) error
	close()
}

// factory builds the CSMS and the stations of one OCPP version.
type factory struct {
	newCentral func() (central, error)
	newStation func(url, id string) (simulated, error)
}

func factoryFor(version string) (factory, error) {
	switch version {
	case "1.6":
		return factory16(), nil
	case "2.0.1":
		return factory201(), nil
	}
	return factory{}, fmt.Errorf("unknown version %q: use 1.6 or 2.0.1", version)
}

// stats collects the latency of every call and the errors.
type stats struct {
	mu        sync.Mutex
	latencies []time.Duration
	errors    atomic.Int64
	firstErr  atomic.Value // error
}

func (s *stats) observe(d time.Duration, err error) {
	if err != nil {
		s.errors.Add(1)
		s.firstErr.CompareAndSwap(nil, err)
		return
	}
	s.mu.Lock()
	s.latencies = append(s.latencies, d)
	s.mu.Unlock()
}

func (s *stats) percentile(p float64) time.Duration {
	if len(s.latencies) == 0 {
		return 0
	}
	return s.latencies[min(int(p*float64(len(s.latencies))), len(s.latencies)-1)]
}

// call runs one request, records its latency and counts it.
func (s *stats) call(calls *atomic.Int64, f func() error) {
	start := time.Now()
	err := f()
	s.observe(time.Since(start), err)
	calls.Add(1)
}

// drive makes one station boot, then send heartbeats and run sessions until ctx
// ends. Stations start at different offsets so they do not all fire together.
func drive(ctx context.Context, s simulated, index int, opts options, st *stats, calls *atomic.Int64) {
	var cctx = context.WithoutCancel(ctx) // a call in flight at the deadline still completes
	st.call(calls, func() error { return s.boot(cctx) })

	sleep := func(d time.Duration) bool {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(d):
			return true
		}
	}
	if !sleep(time.Duration(index) * opts.heartbeat / time.Duration(opts.stations)) {
		return
	}
	nextSession := time.Now().Add(opts.session)
	for {
		st.call(calls, func() error { return s.heartbeat(cctx) })
		if opts.session > 0 && !time.Now().Before(nextSession) {
			st.call(calls, func() error { return s.chargingSession(cctx) })
			nextSession = time.Now().Add(opts.session)
		}
		if !sleep(opts.heartbeat) {
			return
		}
	}
}

func run(ctx context.Context, opts options, out io.Writer) error {
	if opts.stations < 1 {
		return fmt.Errorf("-stations must be at least 1")
	}
	f, err := factoryFor(opts.version)
	if err != nil {
		return err
	}
	srv, err := f.newCentral()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	go func() { _ = srv.serve(ln) }()
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.shutdown(shutdown)
	}()
	url := "ws://" + ln.Addr().String() + "/ocpp"

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fmt.Fprintf(out, "OCPP %s: connecting %d stations to %s\n", opts.version, opts.stations, ln.Addr())
	connectStart := time.Now()
	stations := make([]simulated, opts.stations)
	for i := range stations {
		st, err := f.newStation(url, fmt.Sprintf("LT%05d", i))
		if err != nil {
			return err
		}
		stations[i] = st
		go st.run(ctx)
	}
	defer func() {
		for _, st := range stations {
			st.close()
		}
	}()
	cctx, ccancel := context.WithTimeout(ctx, 2*time.Minute)
	for _, st := range stations {
		if err := st.waitConnected(cctx); err != nil {
			ccancel()
			return fmt.Errorf("stations did not all connect: %w", err)
		}
	}
	ccancel()
	fmt.Fprintf(out, "all connected in %s\n", time.Since(connectStart).Round(time.Millisecond))

	var st stats
	var calls atomic.Int64
	deadline, stopTimer := context.WithTimeout(ctx, opts.duration)
	defer stopTimer()
	var wg sync.WaitGroup
	start := time.Now()
	for i, s := range stations {
		wg.Add(1)
		go func() {
			defer wg.Done()
			drive(deadline, s, i, opts, &st, &calls)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	sort.Slice(st.latencies, func(i, j int) bool { return st.latencies[i] < st.latencies[j] })
	fmt.Fprintf(out, "\n%d operations (a heartbeat, a boot or a whole charging session) in %s: %.0f operations/s, %d errors\n", calls.Load(), elapsed.Round(time.Millisecond),
		float64(calls.Load())/elapsed.Seconds(), st.errors.Load())
	fmt.Fprintf(out, "latency per operation: p50 %s, p95 %s, p99 %s, max %s\n", st.percentile(0.50), st.percentile(0.95), st.percentile(0.99), st.percentile(1))
	if e, _ := st.firstErr.Load().(error); e != nil {
		fmt.Fprintf(out, "first error: %v\n", e)
		return fmt.Errorf("%d operations failed", st.errors.Load())
	}
	return nil
}
