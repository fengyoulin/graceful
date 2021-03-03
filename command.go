package graceful

// CtrlCommand is a control command
type CtrlCommand struct {
	// Command currently can be one of CommandShutdown and CommandRestart
	Command int
	// ErrorChannel if not nil can be used to receive the error
	ErrorChannel chan error
}

const (
	// CommandShutdown makes the all servers and the service process shutdown
	CommandShutdown = iota
	// CommandRestart makes the service process perform a graceful restart
	CommandRestart
)

// CommandChannel is the "command channel"
var CommandChannel chan CtrlCommand
