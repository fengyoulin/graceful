package graceful

import (
	"errors"
	"fmt"
	"net"
	"sync"
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
func RunServers(startDelay, shutdownWait time.Duration) (err error) {
	if err = Init(nil); err != nil {
		return
	}
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
		runAsWorker(shutdownWait)
	} else {
		for idx := range servers {
			info := &servers[idx]
			info.listener, err = net.Listen(info.network, info.address)
			if err != nil {
				return
			}
		}
		if err = runAsMaster(startDelay); err != nil {
			return
		}
	}
	return
}
