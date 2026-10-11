package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthz(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantBody   string
	}{
		{name: "GET returns ok", method: http.MethodGet, wantStatus: http.StatusOK, wantBody: `{"status":"ok"}`},
		{name: "POST not allowed", method: http.MethodPost, wantStatus: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newMux().ServeHTTP(rec, httptest.NewRequest(tt.method, "/healthz", nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

// TestCalculateRoute checks the mux wiring end to end; the handler's own
// behaviour is covered in internal/handler. The 405 case guards against
// adding a method to the route pattern, which would make the mux answer
// in plain text instead of the handler's JSON envelope.
func TestCalculateRoute(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantBody   string
		wantAllow  string
	}{
		{
			name:       "POST calculates",
			method:     http.MethodPost,
			body:       `{"operation":"divide","a":10,"b":2}`,
			wantStatus: http.StatusOK,
			wantBody:   `{"result":5}`,
		},
		{
			name:       "POST domain error",
			method:     http.MethodPost,
			body:       `{"operation":"divide","a":10,"b":0}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"division by zero is not allowed"}`,
		},
		{
			name:       "GET is a JSON 405",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `{"error":"method not allowed"}`,
			wantAllow:  http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/api/v1/calculate", strings.NewReader(tt.body))
			newMux().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Errorf("Allow = %q, want %q", got, tt.wantAllow)
			}
		})
	}
}

func TestRun(t *testing.T) {
	t.Run("returns nil after graceful shutdown on cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() { errCh <- run(ctx, "127.0.0.1:0") }()

		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("run() = %v, want nil", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("run() did not return after context cancel")
		}
	})

	t.Run("returns listen error instead of exiting", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("reserve port: %v", err)
		}
		defer ln.Close()

		errCh := make(chan error, 1)
		go func() { errCh <- run(context.Background(), ln.Addr().String()) }()

		select {
		case err := <-errCh:
			if err == nil {
				t.Fatal("run() = nil, want error for an address already in use")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("run() did not return for an address already in use")
		}
	})
}
