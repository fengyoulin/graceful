# graceful #

Graceful restart and zero downtime deploy for golang servers. Support multiple servers each listening on a distinct address.

**Example:**
```go
package main

import (
	"context"
	"github.com/fengyoulin/graceful"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {
	err := graceful.AddServer("tcp", ":9001", NewTestServer())
	if err != nil {
		log.Fatalln(err)
	}
	err = graceful.AddServer("unix", "/tmp/test.sock", NewTestServer())
	if err != nil {
		log.Fatalln(err)
	}
	err = graceful.RunServers(time.Second, time.Second)
	if err != nil {
		log.Fatalln(err)
	}
}

type testServer struct {
	name   string
	server *http.Server
}

func (s *testServer) Name() string {
	return s.name
}

func (s *testServer) Serve(l net.Listener) error {
	err := s.server.Serve(l)
	if err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *testServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func NewTestServer() graceful.Server {
	m := http.NewServeMux()
	m.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("Test Server.\n"))
	})
	s := &testServer{
		name: "test-server",
		server: &http.Server{
			Handler: m,
		},
	}
	return s
}
```
**Signals:**
- Use `SIGINT` and `SIGKILL` to terminate the process.
- Use `SIGHUP` to perform a graceful restart.
