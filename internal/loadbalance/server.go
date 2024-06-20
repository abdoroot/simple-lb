package loadbalance

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

type clientServer struct {
	url   string
	alive bool
}

type server struct {
	Serverlist   []clientServer
	nextSrvIndex int //last server used
	addr         string
	mux          *http.ServeMux
	logger       *slog.Logger
	mu           sync.Mutex
	isStarted    chan struct{}
}

type Config struct {
	Addr      string
	IsStarted chan struct{}
}

func NewLbServer(cfg Config) *server {
	var (
		mux       = http.NewServeMux()
		addr      = ":8080"
		isStarted chan struct{}
	)

	if cfg.Addr != "" {
		addr = cfg.Addr
	}

	if cfg.IsStarted != nil {
		isStarted = cfg.IsStarted
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return &server{
		Serverlist: make([]clientServer, 0),
		mux:        mux,
		addr:       addr,
		logger:     logger,
		isStarted:  isStarted,
	}
}

func (s *server) Start() error {
	s.mux.HandleFunc("/", s.HandleIncomingRequest)
	s.mux.HandleFunc("/status", s.HandleGetServerStatus)
	slog.Info("load balance server running at", "addr", s.addr)
	s.isStarted <- struct{}{}
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *server) AddServer(srv string) error {
	if !s.isServerAlive(srv) {
		return fmt.Errorf("client server is down")
	}

	s.mu.Lock()
	s.Serverlist = append(s.Serverlist, clientServer{url: srv, alive: true})
	s.mu.Unlock()
	s.logger.Info("added client server to load balancer", "srv", srv)
	return nil
}

func (s *server) isServerAlive(srv string) bool {
	url := fmt.Sprintf("%v/status", srv)
	s.logger.Info("checking client server is alive", "url", url)
	req, err := http.Get(url)
	if err != nil {
		return false
	}
	return req.StatusCode == 200
}

func (s *server) Next() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastIndex := len(s.Serverlist) - 1
	if lastIndex < 0 {
		return "", fmt.Errorf("no available servers")
	}

	nextIndex := s.nextSrvIndex
	c := s.Serverlist[nextIndex]
	s.nextSrvIndex = (nextIndex + 1) % len(s.Serverlist)
	return c.url, nil
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

	if len(s.Serverlist) == 0 {
		HandlerError(w, fmt.Errorf("no available servers to handle the request"))
		return
	}
	srv, err := s.Next()
	if err != nil {
		HandlerError(w, err)
		return
	}

	httpClient := http.Client{Timeout: 10 * time.Second}
	s.logger.Error("forwording the request to", "srv", srv)
	req, err := http.NewRequest(r.Method, srv+r.URL.Path, r.Body)
	if err != nil {
		s.logger.Error("err creating new request HandleIncomingRequest", "err", err)
		HandlerError(w, err)
		return
	}

	copyHeader(r.Header, req.Header)
	resp, err := httpClient.Do(req)
	if err != nil {
		s.logger.Error("err doing the request HandleIncomingRequest", "err", err)
		HandlerError(w, err)
		return
	}
	defer resp.Body.Close()

	//send it back to users
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("err reading response body", "err", err)
		HandlerError(w, err)
		return
	}
	defer resp.Body.Close()
	w.Write(buf)
}

func HandlerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

func (s *server) PeriodicHealthCheck(interval time.Duration) {
	//for client
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.logger.Info("client server periodic health check")
		if len(s.Serverlist) > 0 {
			for i, c := range s.Serverlist {
				if s.isServerAlive(c.url) {
					s.mu.Lock()
					s.Serverlist[i].alive = true
					s.mu.Unlock()

					s.logger.Info("client server is alive ", "url", c.url)
				} else {
					s.mu.Lock()
					s.Serverlist[i].alive = false
					s.mu.Unlock()
					s.logger.Info("client server is down ", "url", c.url)
				}
			}
		}
	}
}

func copyHeader(src http.Header, dst http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
