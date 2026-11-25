package csms_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v201"
	"github.com/foecum/go-ocpp-sdk/csms/v201"
	"github.com/foecum/go-ocpp-sdk/v201/availability"
)

type benchHandler struct {
	csms.UnimplementedAvailabilityHandler
}

func (benchHandler) Heartbeat(context.Context, *csms.ChargePoint, *availability.HeartbeatRequest) (*availability.HeartbeatResponse, error) {
	return &availability.HeartbeatResponse{CurrentTime: ocpp.Now()}, nil
}

func heapInUse() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// BenchmarkStations connects N simulated stations to one CSMS over loopback
// WebSockets and makes every station send heartbeats concurrently. ns/op is the
// time per heartbeat round trip across all stations; the extra metrics are the
// heap per connected station (both ends are in this process) and the time to
// connect them all.
func BenchmarkStations(b *testing.B) {
	for _, n := range []int{1, 100, 1000} {
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			srv, err := csms.New(csms.Config{Availability: benchHandler{}})
			if err != nil {
				b.Fatal(err)
			}
			ts := httptest.NewServer(srv)
			defer ts.Close()
			defer func() { _ = srv.Shutdown(context.Background()) }()
			url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ocpp"

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			before := heapInUse()
			start := time.Now()
			stations := make([]*chargepoint.ChargePoint, n)
			for i := range stations {
				cp, err := chargepoint.New(chargepoint.Config{URL: url, ID: fmt.Sprintf("CS%04d", i)})
				if err != nil {
					b.Fatal(err)
				}
				stations[i] = cp
				go func() { _ = cp.Run(ctx) }()
			}
			wctx, wcancel := context.WithTimeout(ctx, 60*time.Second)
			for _, cp := range stations {
				if err := cp.WaitConnected(wctx); err != nil {
					b.Fatal(err)
				}
			}
			wcancel()
			connect := time.Since(start)
			perStation := float64(heapInUse()-before) / float64(n)
			defer func() {
				for _, cp := range stations {
					_ = cp.Close()
				}
			}()

			b.ResetTimer()
			var wg sync.WaitGroup
			per := max(b.N/n, 1)
			for _, cp := range stations {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range per {
						if _, err := cp.Heartbeat(ctx, availability.HeartbeatRequest{}); err != nil {
							b.Error(err)
							return
						}
					}
				}()
			}
			wg.Wait()
			b.StopTimer()
			b.ReportMetric(perStation, "heap-B/station")
			b.ReportMetric(float64(connect.Milliseconds()), "connect-ms")
		})
	}
}
