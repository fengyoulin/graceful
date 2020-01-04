package graceful

// CtrlCommand is a control command
type CtrlCommand struct {
	// Command currently can be one of CommandShutdown and CommandRestart
	Command int
	// ErrCh if not nil can be used to receive the error
	ErrCh chan error
}

const (
	// CommandShutdown makes the all servers and the service process shutdown
	CommandShutdown = iota
	// CommandRestart makes the service process perform a graceful restart
	CommandRestart
)

// CommandCh is the "command channel"
var CommandCh chan CtrlCommand
