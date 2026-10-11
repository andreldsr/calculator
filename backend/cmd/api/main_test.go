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

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name:    "real healthz is healthy",
			handler: newMux().ServeHTTP,
		},
		{
			name:    "500 is unhealthy",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantErr: true,
		},
		{
			name:    "404 is unhealthy",
			handler: http.NotFound,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			err := checkHealth(context.Background(), srv.URL+"/healthz")
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkHealth() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("unreachable server is unhealthy", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL + "/healthz"
		srv.Close()

		if err := checkHealth(context.Background(), url); err == nil {
			t.Fatal("checkHealth() = nil, want error for a closed server")
		}
	})

	t.Run("slow server times out", func(t *testing.T) {
		release := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			<-release
		}))
		defer srv.Close()
		defer close(release)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if err := checkHealth(ctx, srv.URL+"/healthz"); err == nil {
			t.Fatal("checkHealth() = nil, want error when the context times out")
		}
	})
}

func TestHealthcheckURL(t *testing.T) {
	tests := []struct {
		port string
		want string
	}{
		{port: "", want: "http://127.0.0.1:8080/healthz"},
		{port: "9090", want: "http://127.0.0.1:9090/healthz"},
	}

	for _, tt := range tests {
		t.Run("PORT="+tt.port, func(t *testing.T) {
			t.Setenv("PORT", tt.port)
			if got := healthcheckURL(); got != tt.want {
				t.Errorf("healthcheckURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
