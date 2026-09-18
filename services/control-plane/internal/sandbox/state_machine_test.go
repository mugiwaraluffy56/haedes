package sandbox

import "testing"

func TestTransitionAllowsOnlyLifecycleEdges(t *testing.T) {
	tests := []struct {
		name string
		from State
		to   State
		ok   bool
	}{
		{"requested to provisioning", StateRequested, StateProvisioning, true},
		{"provisioning to starting", StateProvisioning, StateStarting, true},
		{"starting to running", StateStarting, StateRunning, true},
		{"running to snapshotting", StateRunning, StateSnapshotting, true},
		{"running to stopping", StateRunning, StateStopping, true},
		{"stopping to stopped", StateStopping, StateStopped, true},
		{"stopped to destroyed", StateStopped, StateDestroyed, true},
		{"destroyed cannot restart", StateDestroyed, StateRunning, false},
		{"stopped cannot run", StateStopped, StateRunning, false},
		{"requested cannot run", StateRequested, StateRunning, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sandbox := Sandbox{State: test.from}
			err := Transition(&sandbox, test.to, "test")
			if test.ok && err != nil {
				t.Fatalf("Transition() error = %v", err)
			}
			if !test.ok {
				if _, ok := err.(InvalidStateTransition); !ok {
					t.Fatalf("Transition() error = %T, want InvalidStateTransition", err)
				}
				return
			}
			if sandbox.State != test.to {
				t.Fatalf("state = %q, want %q", sandbox.State, test.to)
			}
		})
	}
}

func TestTransitionRetainsFailureReason(t *testing.T) {
	sandbox := Sandbox{State: StateStarting}
	if err := Transition(&sandbox, StateFailed, "runtime unavailable"); err != nil {
		t.Fatal(err)
	}
	if sandbox.FailureReason != "runtime unavailable" {
		t.Fatalf("failure reason = %q", sandbox.FailureReason)
	}
}
