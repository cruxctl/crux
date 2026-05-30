package cruxapi

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSession_JSONRoundtrip(t *testing.T) {
	s := Session{
		ID:        "sess_1",
		AgentID:   "claude-code",
		ProjectID: "p1",
		UserID:    "u1",
		MachineID: "m1",
		Status:    SessionStatusRunning,
		StartedAt: time.Now().UTC(),
		Cost:      Cost{TokensIn: 1, TokensOut: 2, USD: 0.001},
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var s2 Session
	if err := json.Unmarshal(b, &s2); err != nil {
		t.Fatal(err)
	}
	if s2.AgentID != "claude-code" || s2.Status != SessionStatusRunning {
		t.Errorf("roundtrip lost data: %+v", s2)
	}
}

func TestAOSEvent_RequiredFields(t *testing.T) {
	e := AOSEvent{
		Schema:    "crux.aos.event.v1",
		EventID:   "evt_1",
		Timestamp: time.Now().UTC(),
		EventType: "session.created",
		Actor:     Actor{AgentID: "claude-code"},
	}
	if e.Schema != "crux.aos.event.v1" {
		t.Errorf("schema = %q", e.Schema)
	}
}

func TestPTYSpec_JSONRoundtrip(t *testing.T) {
	s := PTYSpec{
		ID:      "p_1",
		Command: "claude",
		Args:    []string{"--version"},
		Rows:    40,
		Cols:    120,
		Purpose: PTYPurposeProbe,
		Capture: CaptureRawAndANSI,
		Script: []PTYStep{
			{Send: "/help\n"},
			{Expect: "Commands"},
			{Snapshot: true},
		},
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var s2 PTYSpec
	if err := json.Unmarshal(b, &s2); err != nil {
		t.Fatal(err)
	}
	if s2.Purpose != PTYPurposeProbe || len(s2.Script) != 3 {
		t.Errorf("roundtrip lost data: %+v", s2)
	}
}

func TestCodingAgentSpec_Minimal(t *testing.T) {
	spec := CodingAgentSpec{
		APIVersion: "crux.dev/v1alpha1",
		Kind:       "CodingAgentSpec",
		Metadata:   AgentSpecMetadata{ID: "x", Name: "X", Provider: "p"},
	}
	if spec.Kind != "CodingAgentSpec" {
		t.Errorf("kind = %q", spec.Kind)
	}
}
