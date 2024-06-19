package loadbalance

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

type client struct {
	addr, LbEndPoint string //load balance server end point
	mux              *http.ServeMux
	logger           *slog.Logger
}

func NewClientServer(cfg Config) *client {
	var (
		mux  = http.NewServeMux()
		addr = ":8080"
	)

	if cfg.Addr != "" {
		addr = cfg.Addr
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return &client{
		mux:    mux,
		addr:   addr,
		logger: logger,
	}
}

func (c *client) Start() error {
	c.mux.HandleFunc("/status", c.HandleGetClientStatus)
	slog.Info("client server running at", "addr", c.addr)
	return http.ListenAndServe(c.addr, c.mux)
}

func (c *client) HandleGetClientStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (c *client) ISLBServerRunning() bool {
	url := fmt.Sprintf("%v/status", c.LbEndPoint)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("error creating request", err)
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.logger.Error("error doing the request", err)
		return false
	}
	return resp.StatusCode == 200
}

func (c *client) HandleIncomingRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("Hello From Backend Server"))
}
