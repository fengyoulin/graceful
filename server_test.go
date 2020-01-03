package graceful

import (
	"net/http"
	"testing"
	"time"
)

func Test(t *testing.T) {
	err := AddServer("tcp", "127.0.0.1:9999", NewControlServer())
	if err != nil {
		t.Error(err)
	}
	err = AddServer("tcp", ":9001", NewTestServer())
	if err != nil {
		t.Error(err)
	}
	err = RunServers(time.Second, time.Second)
	if err != nil {
		t.Error(err)
	}
}

func NewTestServer() Server {
	m := http.NewServeMux()
	m.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("Test Server.\n"))
	})
	return &http.Server{
		Handler: m,
	}
}
