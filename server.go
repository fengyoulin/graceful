package graceful

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type serverInfo struct {
	network  string
	address  string
	server   Server
	listener net.Listener
}

type ctrlCmd struct {
	cmd   int
	errCh chan error
}

const (
	cmdShutdown = iota
	cmdRestart
)

var (
	// ErrStarted means the servers already started
	ErrStarted = errors.New("already started")
	// ErrConflict means the current server already added
	ErrConflict = errors.New("address conflict")
	// ErrUnsupported means the listener type is not supported
	ErrUnsupported = errors.New("unsupported listener type")
)

var (
	servers []serverInfo
	started bool
	lock    sync.Mutex
)

var cmdCh chan ctrlCmd

// AddServer add a server
func AddServer(network, address string, server Server) error {
	lock.Lock()
	defer lock.Unlock()
	/*
	 * must not add a server after servers started
	 */
	if started {
		return ErrStarted
	}
	var found bool
	for _, svr := range servers {
		if svr.network == network && svr.address == address {
			found = true
		}
	}
	if !found {
		servers = append(servers, serverInfo{
			network: network,
			address: address,
			server:  server,
		})
	} else {
		return ErrConflict
	}
	return nil
}

// RunServers run all servers added
func RunServers(startWait, shutdownWait time.Duration) (err error) {
	lock.Lock()
	defer lock.Unlock()
	started = true
	/*
	 * get listeners from inherited files or create them
	 */
	if IsGraceful() {
		files := GetInheritedFiles()
		if len(files) < len(servers) {
			log.Fatalln("too few inherited files")
		}
		for idx := range servers {
			servers[idx].listener, err = net.FileListener(files[idx])
			if err != nil {
				return
			}
		}
	} else {
		for idx := range servers {
			info := &servers[idx]
			info.listener, err = net.Listen(info.network, info.address)
			if err != nil {
				return
			}
		}
	}
	var wg sync.WaitGroup
	/*
	 * run servers each in a goroutine
	 */
	for idx := range servers {
		wg.Add(1)
		go func(i int) {
			info := &servers[i]
			err := info.server.Serve(info.listener)
			if err != nil {
				log.Printf("server %d serve error: %v", i, err)
			}
			wg.Done()
		}(idx)
	}
	cmdCh = make(chan ctrlCmd)
	/*
	 * run the signal handler
	 */
	runSignalHandler()
	/*
	 * process control commands in a separate goroutine
	 */
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			cmd := <-cmdCh
			switch cmd.cmd {
			case cmdShutdown:
				shutdownServers(shutdownWait)
				return
			case cmdRestart:
				err := startProcess(startWait, func() {
					cmdCh <- ctrlCmd{cmd: cmdShutdown}
				})
				/*
				 * send the result to the caller, or print a log on error
				 */
				select {
				case cmd.errCh <- err:
				default:
					if err != nil {
						log.Printf("start process error: %v", err)
					}
				}
			}
		}
	}()
	/*
	 * wait until all finished
	 */
	wg.Wait()
	return
}

func runSignalHandler() {
	/*
	 * prepare signal channel
	 */
	sch := make(chan os.Signal)
	signal.Notify(sch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGKILL)
	/*
	 * run the signal handler routine
	 */
	go func() {
		for {
			sig := <-sch
			switch sig {
			case syscall.SIGINT, syscall.SIGKILL:
				cmdCh <- ctrlCmd{cmd: cmdShutdown}
			case syscall.SIGHUP:
				cmdCh <- ctrlCmd{cmd: cmdRestart}
			}
		}
	}()
}

func shutdownServers(wait time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	nch := make(chan struct{})
	/*
	 * call servers's shutdown, each in a goroutine
	 */
	for idx := range servers {
		go func(i int) {
			info := &servers[i]
			err := info.server.Shutdown(ctx)
			if err != nil {
				log.Printf("server %d shutdown error %v", i, err)
			}
			nch <- struct{}{}
		}(idx)
	}
	var cnt int
	/*
	 * wait until all finished or timeout
	 */
	for cnt < len(servers) {
		select {
		case <-ctx.Done():
			return
		case <-nch:
			cnt++
		}
	}
	cancel()
}

func startProcess(wait time.Duration, fn func()) (err error) {
	files := make([]*os.File, len(servers))
	/*
	 * convert net.Listener to *os.File
	 */
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
	/*
	 * start the new process with extra files
	 */
	var cmd *exec.Cmd
	cmd, err = Start(files)
	if err != nil {
		return
	}
	/*
	 * confirm the new process alive after a moment
	 */
	ch := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		ch <- struct{}{}
	}()
	go func() {
		select {
		case <-ch:
			log.Printf("process %d exited too quick", cmd.ProcessState.Pid())
		case <-time.After(wait):
			if fn != nil {
				fn()
			}
		}
	}()
	return
}
