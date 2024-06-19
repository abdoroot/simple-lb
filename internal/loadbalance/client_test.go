package loadbalance

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientStatus(t *testing.T) {
	client := NewClientServer(Config{})
	svr := httptest.NewServer(http.HandlerFunc(client.HandleGetClientStatus))
	defer svr.Close()
	slog.Info("server url", "url", svr.URL)

	req, err := http.NewRequest("GET", svr.URL, nil)
	if err != nil {
		t.Error(err)
		return
	}
	//do the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 got %v", resp.StatusCode)
	}
}
