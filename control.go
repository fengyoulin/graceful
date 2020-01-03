package graceful

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
)

type controlServer struct {
	server *http.Server
}

type cmdResponse struct {
	Err string `json:"err,omitempty"`
}

func (s *controlServer) Serve(l net.Listener) error {
	err := s.server.Serve(l)
	if err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *controlServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// NewControlServer creates a control command server
func NewControlServer() Server {
	h := func(writer http.ResponseWriter, request *http.Request) {
		var err error
		defer func() {
			if err != nil {
				log.Printf("control server error: %v", err)
			}
		}()
		var response cmdResponse
		switch request.RequestURI {
		case "/stop":
			cmdCh <- ctrlCmd{cmd: cmdShutdown}
		case "/restart":
			cmd := ctrlCmd{cmd: cmdRestart, errCh: make(chan error)}
			cmdCh <- cmd
			if e := <-cmd.errCh; e != nil {
				response.Err = e.Error()
			}
		default:
			response.Err = "unknown command"
		}
		var data []byte
		data, err = json.Marshal(response)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		writer.Header().Add("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, err = writer.Write(data)
	}
	m := http.NewServeMux()
	m.HandleFunc("/", h)
	return &controlServer{
		server: &http.Server{
			Handler: m,
		},
	}
}
