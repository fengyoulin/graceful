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

type commandResponse struct {
	Ok  bool   `json:"ok,omitempty"`
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

		// process commands
		var response commandResponse
		switch request.RequestURI {
		case "/shutdown":
			CommandCh <- CtrlCommand{Command: CommandShutdown}
		case "/restart":
			cmd := CtrlCommand{Command: CommandRestart, ErrCh: make(chan error)}
			CommandCh <- cmd
			if e := <-cmd.ErrCh; e != nil {
				response.Err = e.Error()
			}
		default:
			response.Err = "unknown command"
		}

		// write result back
		if len(response.Err) == 0 {
			response.Ok = true
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
