package examples_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// output collects a process's lines and lets a test wait for one.
type output struct {
	mu    sync.Mutex
	lines []string
}

func (o *output) add(line string) { o.mu.Lock(); o.lines = append(o.lines, line); o.mu.Unlock() }

func (o *output) all() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return strings.Join(o.lines, "\n")
}

func (o *output) waitFor(t *testing.T, re *regexp.Regexp) []string {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if m := re.FindStringSubmatch(o.all()); m != nil {
			return m
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %v in:\n%s", re, o.all())
	return nil
}

func start(t *testing.T, ctx context.Context, bin string, args ...string) (*exec.Cmd, *output) {
	t.Helper()
	cmd := exec.CommandContext(ctx, bin, args...)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	out := &output{}
	go func() {
		sc := bufio.NewScanner(pipe)
		for sc.Scan() {
			out.add(sc.Text())
		}
		_, _ = io.Copy(io.Discard, pipe)
	}()
	return cmd, out
}

// TestExamplesTalkToEachOther builds both example programs, runs the Central
// System and a charging station against each other, and checks that a whole
// charging session completes: one started by the Central System, which then
// stops it remotely.
func TestExamplesTalkToEachOther(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs two programs")
	}
	dir := t.TempDir()
	build := func(pkg string) string {
		bin := filepath.Join(dir, filepath.Base(pkg))
		if b, err := exec.Command("go", "build", "-o", bin, pkg).CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", pkg, err, b)
		}
		return bin
	}
	csmsBin, stationBin := build("./csms"), build("./chargingstation")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	csms, csmsOut := start(t, ctx, csmsBin, "-addr", "127.0.0.1:0", "-interval", "1", "-password", "s3cret-password", "-remote-start")
	addr := csmsOut.waitFor(t, regexp.MustCompile(`listening on (127\.0\.0\.1:\d+)`))[1]

	// The idle pause is long enough for the Central System's remote start to
	// arrive first, so the one session is the remote one.
	station, stationOut := start(t, ctx, stationBin,
		"-url", fmt.Sprintf("ws://%s/ocpp", addr), "-id", "CS001", "-password", "s3cret-password",
		"-sessions", "1", "-session-length", "30s", "-meter-every", "300ms", "-idle", "10s")
	if err := station.Wait(); err != nil {
		t.Fatalf("station exited with %v\nstation:\n%s\ncsms:\n%s", err, stationOut.all(), csmsOut.all())
	}

	for _, want := range []string{`msg="boot answered"`, `msg="remote start accepted"`, `msg="session finished"`, `reason=Remote`, `msg="default tariff set"`, `msg="event stream values sent"`} {
		if !strings.Contains(stationOut.all(), want) {
			t.Errorf("station output lacks %s:\n%s", want, stationOut.all())
		}
	}
	// The Central System saw the whole flow, and called the station back.
	for _, want := range []string{"charging station connected", "boot notification", `msg=variable `, "variable set", "request start transaction",
		"event=Started", "event=Updated", "meter value", "request stop transaction", "event=Ended",
		"default tariff set", "event stream opened", "event stream values"} {
		csmsOut.waitFor(t, regexp.MustCompile(regexp.QuoteMeta(want)))
	}

	// Ctrl-C stops the Central System cleanly.
	if err := csms.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	if err := csms.Wait(); err != nil {
		t.Fatalf("csms exited with %v\n%s", err, csmsOut.all())
	}
}
