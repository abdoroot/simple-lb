package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	lb "github.com/abdoroot/g-load-balance/internal/loadbalance"
)

func main() {
	lbisStarted := make(chan struct{})
	lbs := lb.NewLbServer(lb.Config{IsStarted: lbisStarted})
	go func() {
		//load balancer server
		if err := lbs.Start(); err != nil {
			slog.Info("error starting load balance server", err)
			return
		}
	}()

	//wait load balance to start
	<-lbisStarted

	go func() {
		//check client health every 5 second
		lbs.PeriodicHealthCheck(time.Second * 5)
	}()

	//todo:sepreate the client from the server
	isStartedChan1 := make(chan struct{})
	isStartedChan2 := make(chan struct{})
	isStartedChan3 := make(chan struct{})

	cfg1 := lb.Config{Addr: ":8081", IsStarted: isStartedChan1}
	cfg2 := lb.Config{Addr: ":8082", IsStarted: isStartedChan2}
	cfg3 := lb.Config{Addr: ":8083", IsStarted: isStartedChan3}
	client1 := lb.NewClientServer(cfg1)
	client2 := lb.NewClientServer(cfg2)
	client3 := lb.NewClientServer(cfg3)
	go func() {
		//load balancer client 1
		if err := client1.Start(); err != nil {
			slog.Info("error starting client server", "addr", cfg1.Addr, "err", err)
		}
	}()

	go func() {
		//load balancer client 2
		if err := client2.Start(); err != nil {
			slog.Info("error starting client server", "addr", cfg2.Addr, "err", err)
		}
	}()

	go func() {
		//load balancer client 3
		if err := client3.Start(); err != nil {
			slog.Info("error starting client server", "addr", cfg3.Addr, "err", err)
		}
	}()

	//wait until all client servers is running
	<-isStartedChan1
	<-isStartedChan2
	<-isStartedChan3

	if err := lbs.AddServer("http://localhost:8081"); err != nil {
		slog.Info("error adding client server to load balance server 8081", "err", err)
	}

	if err := lbs.AddServer("http://localhost:8082"); err != nil {
		slog.Info("error adding client server to load balance server :8082", "err", err)
	}

	if err := lbs.AddServer("http://localhost:8083"); err != nil {
		slog.Info("error adding client server to load balance server :8083", "err", err)
	}

	quitChan := make(chan os.Signal, 8)
	signal.Notify(quitChan, os.Interrupt, syscall.SIGABRT)
	select {
	case <-quitChan:
		slog.Info("shutting down")
	}
}
