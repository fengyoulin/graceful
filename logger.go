package graceful

import (
	"log"
	"os"
)

// Logger interface
type Logger interface {
	Fatalf(template string, values ...interface{})
	Printf(template string, values ...interface{})
}

var lg Logger

func init() {
	lg = log.New(os.Stderr, "[graceful] ", log.LstdFlags|log.Lshortfile)
}

// SetLogger ...
func SetLogger(logger Logger) {
	lg = logger
}
