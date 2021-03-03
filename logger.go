package graceful

import (
	"log"
	"os"
)

// Logger interface
type Logger interface {
	Printf(template string, values ...interface{})
}

var lg Logger

func defaultLogger() Logger {
	return log.New(os.Stderr, "[graceful] ", log.LstdFlags)
}
