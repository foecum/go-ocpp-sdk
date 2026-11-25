package ocppj

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

func TestParseValid(t *testing.T) {
	tests := []struct {
		name  string
		frame string
		want  Message
	}{
		{"call", `[2,"19223201","BootNotification",{"reason":"PowerUp"}]`,
			&Call{UniqueID: "19223201", Action: "BootNotification", Payload: []byte(`{"reason":"PowerUp"}`)}},
		{"call null payload", `[2,"1","Heartbeat",null]`, &Call{UniqueID: "1", Action: "Heartbeat", Payload: []byte(`{}`)}},
		{"call whitespace", " [ 2 , \"1\" , \"Heartbeat\" , {} ] ", &Call{UniqueID: "1", Action: "Heartbeat", Payload: []byte(`{}`)}},
		{"call result", `[3,"1",{"status":"Accepted"}]`, &CallResult{UniqueID: "1", Payload: []byte(`{"status":"Accepted"}`)}},
		{"call error", `[4,"1","NotSupported","nope",{}]`,
			&CallErrorFrame{UniqueID: "1", Code: ocpp.NotSupported, Description: "nope", Details: []byte(`{}`)}},
		{"max id", `[3,"` + strings.Repeat("a", 36) + `",{}]`, &CallResult{UniqueID: strings.Repeat("a", 36), Payload: []byte(`{}`)}},
	}
	for _, tt := range tests {
		for _, v := range []ocpp.Version{ocpp.V16, ocpp.V201} {
			got, err := Parse(v, []byte(tt.frame))
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if got.Type() != tt.want.Type() || got.ID() != tt.want.ID() {
				t.Errorf("%s: got %#v", tt.name, got)
			}
			switch w := tt.want.(type) {
			case *Call:
				g := got.(*Call)
				if g.Action != w.Action || string(g.Payload) != string(w.Payload) {
					t.Errorf("%s: got %#v", tt.name, g)
				}
			case *CallResult:
				if string(got.(*CallResult).Payload) != string(w.Payload) {
					t.Errorf("%s: got %#v", tt.name, got)
				}
			case *CallErrorFrame:
				g := got.(*CallErrorFrame)
				if g.Code != w.Code || g.Description != w.Description || string(g.Details) != string(w.Details) {
					t.Errorf("%s: got %#v", tt.name, g)
				}
			}
		}
	}
}

func TestParseErrors(t *testing.T) {
	long := strings.Repeat("a", 37)
	tests := []struct {
		name      string
		v         ocpp.Version
		frame     string
		wantType  MessageType
		wantID    string
		wantCode  ocpp.ErrorCode
		wantReply bool
	}{
		{"malformed json 1.6", ocpp.V16, `[2,"1"`, 0, "-1", ocpp.FormationViolation, true},
		{"malformed json 2.0.1", ocpp.V201, `not json`, 0, "-1", ocpp.RpcFrameworkError, true},
		{"not an array", ocpp.V201, `{"a":1}`, 0, "-1", ocpp.RpcFrameworkError, true},
		{"empty array", ocpp.V16, `[]`, 0, "-1", ocpp.FormationViolation, true},
		{"only type", ocpp.V201, `[2]`, 0, "-1", ocpp.RpcFrameworkError, true},
		{"type not number", ocpp.V201, `["2","1","A",{}]`, 0, "-1", ocpp.RpcFrameworkError, true},
		{"type not integer", ocpp.V201, `[2.5,"1","A",{}]`, 0, "-1", ocpp.RpcFrameworkError, true},
		{"id not string", ocpp.V201, `[2,1,"A",{}]`, CallType, "-1", ocpp.RpcFrameworkError, true},
		{"id empty", ocpp.V16, `[2,"","A",{}]`, CallType, "-1", ocpp.FormationViolation, true},
		{"id too long", ocpp.V201, `[2,"` + long + `","A",{}]`, CallType, "-1", ocpp.RpcFrameworkError, true},
		{"call too short", ocpp.V201, `[2,"1","A"]`, CallType, "1", ocpp.ProtocolError, true},
		{"call too long", ocpp.V201, `[2,"1","A",{},{}]`, CallType, "1", ocpp.ProtocolError, true},
		{"call action number", ocpp.V16, `[2,"1",5,{}]`, CallType, "1", ocpp.ProtocolError, true},
		{"call action empty", ocpp.V16, `[2,"1","",{}]`, CallType, "1", ocpp.ProtocolError, true},
		{"call payload array", ocpp.V16, `[2,"1","A",[]]`, CallType, "1", ocpp.ProtocolError, true},
		{"call payload string", ocpp.V16, `[2,"1","A","x"]`, CallType, "1", ocpp.ProtocolError, true},
		{"result too short", ocpp.V201, `[3,"1"]`, CallResultType, "1", ocpp.ProtocolError, false},
		{"result payload bad", ocpp.V201, `[3,"1",5]`, CallResultType, "1", ocpp.ProtocolError, false},
		{"error too short", ocpp.V201, `[4,"1","X","d"]`, CallErrorType, "1", ocpp.ProtocolError, false},
		{"error code not string", ocpp.V201, `[4,"1",7,"d",{}]`, CallErrorType, "1", ocpp.ProtocolError, false},
		{"error code empty", ocpp.V201, `[4,"1","","d",{}]`, CallErrorType, "1", ocpp.ProtocolError, false},
		{"error description not string", ocpp.V201, `[4,"1","X",7,{}]`, CallErrorType, "1", ocpp.ProtocolError, false},
		{"error details bad", ocpp.V201, `[4,"1","X","d","x"]`, CallErrorType, "1", ocpp.ProtocolError, false},
		{"unknown type 2.0.1", ocpp.V201, `[9,"1","A",{}]`, 9, "1", ocpp.MessageTypeNotSupported, true},
		{"unknown type 2.1 ignored", ocpp.V21, `[7,"1"]`, 7, "1", "", false},
		{"unknown type 1.6 ignored", ocpp.V16, `[9,"1","A",{}]`, 9, "1", "", false},
	}
	for _, tt := range tests {
		m, err := Parse(tt.v, []byte(tt.frame))
		var pe *ParseError
		if m != nil || !errors.As(err, &pe) {
			t.Errorf("%s: want ParseError, got %v %v", tt.name, m, err)
			continue
		}
		if pe.Type != tt.wantType || pe.ID != tt.wantID || pe.Code != tt.wantCode || pe.ShouldReply() != tt.wantReply {
			t.Errorf("%s: got type=%d id=%q code=%q reply=%v", tt.name, pe.Type, pe.ID, pe.Code, pe.ShouldReply())
		}
		if pe.Err == nil || !strings.Contains(pe.Error(), "invalid frame") || errors.Unwrap(pe) != pe.Err {
			t.Errorf("%s: bad error plumbing", tt.name)
		}
	}
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
		want string
	}{
		{"call", &Call{UniqueID: "1", Action: "Heartbeat", Payload: []byte(`{"a":1}`)}, `[2,"1","Heartbeat",{"a":1}]`},
		{"call empty payload", &Call{UniqueID: "1", Action: "Heartbeat"}, `[2,"1","Heartbeat",{}]`},
		{"result", &CallResult{UniqueID: "1", Payload: []byte(`{"x":"y"}`)}, `[3,"1",{"x":"y"}]`},
		{"result empty", &CallResult{UniqueID: "1"}, `[3,"1",{}]`},
		{"error", &CallErrorFrame{UniqueID: "1", Code: ocpp.NotImplemented, Description: "d"}, `[4,"1","NotImplemented","d",{}]`},
		{"error details", &CallErrorFrame{UniqueID: "1", Code: ocpp.GenericError, Details: []byte(`{"k":1}`)}, `[4,"1","GenericError","",{"k":1}]`},
	}
	for _, tt := range tests {
		got, err := Marshal(tt.msg)
		if err != nil || string(got) != tt.want {
			t.Errorf("%s: got %s, %v", tt.name, got, err)
		}
		// Round trip.
		back, err := Parse(ocpp.V201, got)
		if err != nil || back.ID() != tt.msg.ID() || back.Type() != tt.msg.Type() {
			t.Errorf("%s: round trip %v %v", tt.name, back, err)
		}
	}
}

func TestMarshalErrors(t *testing.T) {
	long := strings.Repeat("a", 37)
	tests := map[string]Message{
		"call no id":     &Call{Action: "A"},
		"call long id":   &Call{UniqueID: long, Action: "A"},
		"call no action": &Call{UniqueID: "1"},
		"result no id":   &CallResult{},
		"error no id":    &CallErrorFrame{Code: ocpp.GenericError},
		"error no code":  &CallErrorFrame{UniqueID: "1"},
		"error long desc": &CallErrorFrame{UniqueID: "1", Code: ocpp.GenericError,
			Description: strings.Repeat("é", 256)},
		"bad payload": &Call{UniqueID: "1", Action: "A", Payload: []byte(`{bad`)},
		"value type":  Call{UniqueID: "1", Action: "A"},
	}
	for name, m := range tests {
		if _, err := Marshal(m); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if _, err := Marshal(&Call{Action: "A"}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("want ErrValidation, got %v", err)
	}
	// Exactly 255 multi-byte characters is allowed.
	if _, err := Marshal(&CallErrorFrame{UniqueID: "1", Code: ocpp.GenericError, Description: strings.Repeat("é", 255)}); err != nil {
		t.Errorf("255 runes: %v", err)
	}
}

func TestUUIDGenerator(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	g := NewUUIDGenerator()
	seen := map[string]bool{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				id := g.NewID()
				mu.Lock()
				if !re.MatchString(id) || len(id) > MaxIDLength || seen[id] {
					t.Errorf("bad or duplicate id %q", id)
				}
				seen[id] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	f := IDGeneratorFunc(func() string { return "fixed" })
	if f.NewID() != "fixed" {
		t.Error("IDGeneratorFunc")
	}
}
