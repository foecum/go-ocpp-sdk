package ocppj

import (
	"context"
	"encoding/json"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

const benchCall = `[2,"19223201","Ping",{"message":"hello","count":3}]`

func BenchmarkParse(b *testing.B) {
	data := []byte(benchCall)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Parse(ocpp.V201, data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal(b *testing.B) {
	m := &Call{UniqueID: "19223201", Action: "Ping", Payload: json.RawMessage(`{"message":"hello","count":3}`)}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Marshal(m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeRequest(b *testing.B) {
	r := newRegistry(b, ocpp.V201)
	payload := json.RawMessage(`{"message":"hello","count":3}`)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := r.DecodeRequest("Ping", payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEndpointRoundTrip measures a complete call between two endpoints
// joined back to back: validate, encode, queue, send, parse, decode, dispatch,
// answer, correlate. No network is involved.
func BenchmarkEndpointRoundTrip(b *testing.B) {
	reg := newRegistry(b, ocpp.V201)
	var client, server *Endpoint
	var err error
	server, err = NewEndpoint(Config{
		Registry: reg,
		Send:     func(_ context.Context, f []byte) error { client.HandleFrame(f); return nil },
		Handler: func(context.Context, ocpp.Request) (ocpp.Response, error) {
			return pingResp{Reply: "ok"}, nil
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	client, err = NewEndpoint(Config{
		Registry: reg,
		Send:     func(_ context.Context, f []byte) error { server.HandleFrame(f); return nil },
	})
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()
	defer server.Close()
	ctx := context.Background()
	req := pingReq{Message: "hello", Count: 3}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := client.Call(ctx, req); err != nil {
			b.Fatal(err)
		}
	}
}
