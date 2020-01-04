package graceful

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
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

// AddServer adds a server's info to the servers
func AddServer(network, address string, server Server) error {
	lock.Lock()
	defer lock.Unlock()

	// must not add a server after servers started
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

// RunServers runs all servers added in the global "servers"
func RunServers(startWait, shutdownWait time.Duration) (err error) {
	lock.Lock()
	defer lock.Unlock()
	started = true

	// get listeners from inherited files or create them
	if isGraceful {
		if len(inheritedFiles) < len(servers) {
			return fmt.Errorf("inherited files not enough, need %d but got %d", len(servers), len(inheritedFiles))
		}
		for idx := range servers {
			servers[idx].listener, err = net.FileListener(inheritedFiles[idx])
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

	// run servers each in a goroutine
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
	CommandCh = make(chan CtrlCommand)

	// run the signal handler goroutine
	sch := make(chan os.Signal)
	signal.Notify(sch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGKILL)
	go func() {
		for {
			sig := <-sch
			switch sig {
			case syscall.SIGINT, syscall.SIGKILL:
				CommandCh <- CtrlCommand{Command: CommandShutdown}
			case syscall.SIGHUP:
				CommandCh <- CtrlCommand{Command: CommandRestart}
			}
		}
	}()

	// process control commands in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			cmd := <-CommandCh
			switch cmd.Command {
			case CommandShutdown:
				shutdownServers(shutdownWait)
				return
			case CommandRestart:
				err := startProcess(startWait)
				if err != nil {
					if cmd.ErrCh != nil {
						cmd.ErrCh <- err
					}
				} else {
					go func() {
						CommandCh <- CtrlCommand{Command: CommandShutdown}
					}()
				}
			}
		}
	}()

	// wait until all finished
	wg.Wait()
	return
}

func shutdownServers(wait time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	nch := make(chan struct{})

	// call servers's shutdown, each in a goroutine
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

	// wait until all finished or timeout
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

func startProcess(wait time.Duration) (err error) {
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
	err = startAndWait(files, wait)
	return
}
