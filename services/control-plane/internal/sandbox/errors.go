package sandbox

import "errors"

var (
	ErrNotFound        = errors.New("sandbox not found")
	ErrNotRunning      = errors.New("sandbox is not running")
	ErrRuntimeMissing  = errors.New("sandbox runtime endpoint is unavailable")
	ErrCommandNotFound = errors.New("command not found")
	ErrFileNotFound    = errors.New("file not found")
)

type InvalidStateTransition struct {
	Current   State
	Requested State
}

func (e InvalidStateTransition) Error() string {
	return "invalid sandbox state transition from " + string(e.Current) + " to " + string(e.Requested)
}

func (e InvalidStateTransition) Code() string {
	return "invalid_state_transition"
}
