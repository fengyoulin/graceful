package graceful

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"
)

func runAsWorker(shutdownWait time.Duration) {
	var wg sync.WaitGroup

	// run servers each in a goroutine
	for idx := range servers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			info := &servers[i]
			if err := info.server.Serve(info.listener); err != nil {
				lg.Printf("server [%d] serve error: %v", i, err)
			}
		}(idx)
	}

	// run the signal handler goroutine
	sch := make(chan os.Signal)
	signal.Notify(sch, sigWorkerQuit)

	// process control commands in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sch
		shutdownServers(shutdownWait)
		return
	}()

	// wait until all finished
	wg.Wait()
}

func shutdownServers(wait time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	nch := make(chan struct{})

	// call server's shutdown, each in a goroutine
	for idx := range servers {
		go func(i int) {
			info := &servers[i]
			if err := info.server.Shutdown(ctx); err != nil {
				lg.Printf("server [%d] shutdown error: %v", i, err)
			}
			nch <- struct{}{}
		}(idx)
	}

	// wait until all finished or timeout
	var cnt int
	for cnt < len(servers) {
		select {
		case <-ctx.Done():
			return
		case <-nch:
			cnt++
		}
	}
}
