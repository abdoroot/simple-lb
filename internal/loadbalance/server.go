package loadbalance

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

type server struct {
	Serverlist map[string]bool
	addr       string
	mux        *http.ServeMux
	logger     *slog.Logger
}

type Config struct {
	Addr string
}

func NewLbServer(cfg Config) *server {
	var (
		mux  = http.NewServeMux()
		addr = ":8080"
	)

	if cfg.Addr != "" {
		addr = cfg.Addr
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return &server{
		Serverlist: make(map[string]bool),
		mux:        mux,
		addr:       addr,
		logger:     logger,
	}
}

func (s *server) Start() error {
	s.mux.HandleFunc("/", s.HandleIncomingRequest)
	s.mux.HandleFunc("/status", s.HandleGetServerStatus)
	slog.Info("load balance server running at", "addr", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *server) AddServer(srv string) error {
	url := fmt.Sprintf("%v/status", srv)
	s.logger.Info("checking client server is alive ", "url", url)
	req, err := http.Get(url)
	if err != nil {
		return err
	}

	if req.StatusCode != 200 {
		return fmt.Errorf("client server is down")
	}

	s.Serverlist[srv] = true
	slog.Info("add client server to load balancer server", "srv", srv)
	return nil
}

func (s *server) HandleGetServerStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (s *server) HandleIncomingRequest(w http.ResponseWriter, r *http.Request) {
	s.logger.Info(
		"Received request from",
		"remote_addr", r.RemoteAddr,
		"Method", r.Method+" "+r.URL.Path,
		"User-Agent", r.Method+" "+r.UserAgent(),
		"Accept", r.Method+" "+r.Header.Get("Accept"),
	)
}
