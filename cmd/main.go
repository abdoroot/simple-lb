package main

import (
	"log/slog"
	"os"
	"os/signal"

	lb "github.com/abdoroot/g-load-balance/internal/loadbalance"
)

func main() {
	lbs := lb.NewLbServer(lb.Config{})
	go func() {
		//load balancer server
		if err := lbs.Start(); err != nil {
			slog.Info("error starting load balance server", err)
			return
		}
	}()

	cfg := lb.Config{Addr: ":8001"}
	client := lb.NewClientServer(cfg)
	go func() {
		//load balancer client
		if err := client.Start(); err != nil {
			slog.Info("error starting client server", err)
		}
	}()

	err := lbs.AddServer("http://localhost:8001")
	if err != nil {
		slog.Info("error adding client server to load balance server", "err", err)
	}

	quitChan := make(chan os.Signal, 8)
	signal.Notify(quitChan, os.Interrupt, os.Kill)
	select {
	case <-quitChan:
		slog.Info("shutting down")
	}
}
