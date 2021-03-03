package graceful

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	sigWorkerQuit = syscall.SIGHUP
	sigMasterQuit = syscall.SIGINT
)

func runAsMaster(startDelay time.Duration) error {
	start := true
	// run the signal handler goroutine
	sch := make(chan os.Signal)
	signal.Notify(sch, sigWorkerQuit, sigMasterQuit)
	go func() {
		for start {
			sig := <-sch
			switch sig {
			case sigMasterQuit:
				start = false
				fallthrough
			case sigWorkerQuit:
				if err := workerProcess.Signal(sigWorkerQuit); err != nil {
					lg.Printf("signal worker error: %v", err)
				}
			}
		}
	}()
	for start {
		ts := time.Now()
		if err := startWorker(); err != nil {
			if time.Now().Sub(ts) < startDelay {
				return err
			}
			lg.Printf("start worker error: %v", err)
		}
	}
	return nil
}

func startWorker() (err error) {
	// convert net.Listener to *os.File
	files := make([]*os.File, len(servers))
	for idx := range servers {
		switch listener := servers[idx].listener.(type) {
		case *net.TCPListener:
			files[idx], err = listener.File()
		case *net.UnixListener:
			files[idx], err = listener.File()
		default:
			err = ErrUnsupported
		}
		if err != nil {
			return
		}
	}

	// start the new process with extra files
	err = startAndWait(files)
	return
}
