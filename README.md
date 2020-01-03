# graceful #

Graceful restart and zero downtime deploy for golang servers. Support multiple servers each listening on a distinct address.

**Example:**
```go
package main

import (
	"github.com/fengyoulin/graceful"
	"log"
	"net/http"
	"time"
)

func main() {
	err := graceful.AddServer("tcp", "127.0.0.1:9999", graceful.NewControlServer())
	if err != nil {
		log.Fatalln(err)
	}
	err = graceful.AddServer("tcp", ":9001", NewTestServer())
	if err != nil {
		log.Fatalln(err)
	}
	err = graceful.RunServers(time.Second, time.Second)
	if err != nil {
		log.Fatalln(err)
	}
}

func NewTestServer() graceful.Server {
	m := http.NewServeMux()
	m.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("Test Server.\n"))
	})
	return &http.Server{
		Handler: m,
	}
}
```
**Signals:**
- Use `SIGINT` and `SIGKILL` to terminate the process.
- Use `SIGHUP` to perform a graceful restart.
