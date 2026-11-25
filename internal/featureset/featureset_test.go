package featureset_test

import (
	"testing"

	"github.com/foecum/go-ocpp-sdk/internal/featureset"
	"github.com/foecum/go-ocpp-sdk/v16/core"
)

func TestBuild(t *testing.T) {
	fs := featureset.Build(
		featureset.Pair{Req: core.HeartbeatRequest{}, Resp: core.HeartbeatConfirmation{}},
		featureset.Pair{Req: core.ResetRequest{}, Resp: core.ResetConfirmation{}},
	)
	if len(fs) != 2 || fs[0].Action() != "Heartbeat" || fs[1].Action() != "Reset" {
		t.Errorf("%v", fs)
	}
	if len(featureset.Build()) != 0 {
		t.Error("no pairs, no features")
	}
}

func TestBuildPanicsOnMismatchedPair(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a request paired with another action's confirmation must panic")
		}
	}()
	featureset.Build(featureset.Pair{Req: core.HeartbeatRequest{}, Resp: core.ResetConfirmation{}})
}
