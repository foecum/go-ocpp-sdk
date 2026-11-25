package ocppj

import (
	"bytes"
	"context"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

var frameSeeds = []string{
	`[2,"1","Ping",{"message":"hi"}]`,
	`[2,"1","Ping",null]`,
	`[3,"1",{"reply":"ok"}]`,
	`[4,"1","NotImplemented","",{}]`,
	`[4,"1","FormationViolation","bad",{"a":1}]`,
	`[2,"1","Ping"]`,
	`[9,"1","Ping",{}]`,
	`[6,"1","Stream",{"id":1}]`,
	`[6,"1","Stream"]`,
	`[5,"1","FormatViolation","bad",{}]`,
	`[5,"1","FormatViolation"]`,
	`[2,1,"Ping",{}]`,
	`["2","1","Ping",{}]`,
	`[]`, `{}`, `null`, `[2`, ``, `[2,"` + string(bytes.Repeat([]byte("x"), 40)) + `","Ping",{}]`,
	"[2,\"1\",\"Ping\",{\"message\":\"\xff\xfe\"}]",
}

// FuzzParse checks that arbitrary bytes never make Parse panic, and that a frame
// that parses can be marshalled and parsed again to the same message.
func FuzzParse(f *testing.F) {
	for _, s := range frameSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		for _, v := range []ocpp.Version{ocpp.V16, ocpp.V201, ocpp.V21} {
			m, err := Parse(v, []byte(s))
			if err != nil {
				continue
			}
			out, err := Marshal(m)
			if err != nil {
				continue // for example an ID longer than allowed was accepted on input
			}
			m2, err := Parse(v, out)
			if err != nil {
				t.Fatalf("%v: frame %q re-marshalled to %q, which does not parse: %v", v, s, out, err)
			}
			if m.Type() != m2.Type() || m.ID() != m2.ID() {
				t.Fatalf("%v: %q changed identity through Marshal: %q", v, s, out)
			}
		}
	})
}

// FuzzRegistryDecode checks that arbitrary payloads never panic the decoder.
func FuzzRegistryDecode(f *testing.F) {
	for _, s := range []string{`{"message":"hi"}`, `{"message":"hi","count":3}`, `{}`, `null`, `[]`, `{"message":1}`, `{"reply":"x"}`, `"x"`, ``} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		r := newRegistry(t, ocpp.V201)
		if req, err := r.DecodeRequest("Ping", []byte(s)); err == nil && req.Validate() != nil {
			t.Fatalf("decoded request %q does not validate", s)
		}
		if resp, err := r.DecodeResponse("Ping", []byte(s)); err == nil && resp.Validate() != nil {
			t.Fatalf("decoded response %q does not validate", s)
		}
	})
}

// FuzzEndpointHandleFrame feeds arbitrary frames to a live endpoint. It must
// not panic, and every frame it sends back must be a valid frame.
func FuzzEndpointHandleFrame(f *testing.F) {
	for _, s := range frameSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		sent := make(chan []byte, 16)
		e, err := NewEndpoint(Config{
			Registry:    newRegistry21(t),
			Timeout:     50 * time.Millisecond,
			SendHandler: func(context.Context, ocpp.Request) error { return nil },
			Send: func(_ context.Context, frame []byte) error {
				select {
				case sent <- frame:
				default:
				}
				return nil
			},
			Handler: func(context.Context, ocpp.Request) (ocpp.Response, error) {
				return pingResp{Reply: "ok"}, nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer e.Close()
		e.HandleFrame([]byte(s))
		select {
		case frame := <-sent:
			if _, err := Parse(ocpp.V21, frame); err != nil {
				t.Fatalf("input %q made the endpoint send an invalid frame %q: %v", s, frame, err)
			}
		case <-time.After(5 * time.Millisecond):
		}
	})
}
