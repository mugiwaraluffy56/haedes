package sandbox

var legalTransitions = map[State]map[State]struct{}{
	StateRequested: {
		StateProvisioning: {},
		StateFailed:       {},
	},
	StateProvisioning: {
		StateStarting: {},
		StateFailed:   {},
	},
	StateStarting: {
		StateRunning: {},
		StateFailed:  {},
	},
	StateRunning: {
		StateSnapshotting: {},
		StateStopping:     {},
		StateFailed:       {},
	},
	StateSnapshotting: {
		StateRunning: {},
		StateFailed:  {},
	},
	StateStopping: {
		StateStopped:   {},
		StateDestroyed: {},
		StateFailed:    {},
	},
	StateStopped: {
		StateDestroyed: {},
	},
	StateFailed: {
		StateDestroyed: {},
	},
	StateDestroyed: {},
}

func CanTransition(from, to State) bool {
	_, ok := legalTransitions[from][to]
	return ok
}

func Transition(sandbox *Sandbox, next State, reason string) error {
	if !CanTransition(sandbox.State, next) {
		return InvalidStateTransition{Current: sandbox.State, Requested: next}
	}
	sandbox.State = next
	if next == StateFailed {
		sandbox.FailureReason = reason
	} else {
		sandbox.FailureReason = ""
	}
	return nil
}

func RequireRunning(sandbox Sandbox) error {
	if sandbox.State != StateRunning {
		return ErrNotRunning
	}
	return nil
}
