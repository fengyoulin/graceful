package graceful

import (
	"context"
	"net"
)

// Server is a common interface
type Server interface {
	/*
	 * Serve run the server with a listener
	 */
	Serve(l net.Listener) error
	/*
	 * Shutdown stop the server
	 */
	Shutdown(ctx context.Context) error
}
