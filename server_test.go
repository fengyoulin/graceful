package graceful

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func Test(t *testing.T) {
	err := AddServer("tcp", ":9001", NewTestServer())
	if err != nil {
		t.Error(err)
	}
	err = AddServer("unix", "/tmp/test.sock", NewTestServer())
	if err != nil {
		t.Error(err)
	}
	err = RunServers(time.Second, time.Second)
	if err != nil {
		t.Error(err)
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

func NewTestServer() Server {
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
