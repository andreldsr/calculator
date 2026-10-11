// Command api starts the calculator HTTP server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andre/sezzle-calculator/backend/internal/calculator"
	"github.com/andre/sezzle-calculator/backend/internal/handler"
)

const (
	shutdownTimeout    = 10 * time.Second
	healthcheckTimeout = 3 * time.Second
)

func main() {
	healthcheck := flag.Bool("healthcheck", false,
		"check GET /healthz on the local server and exit 0 if healthy, 1 otherwise")
	flag.Parse()

	// The runtime image has no shell or curl, so the binary checks itself.
	if *healthcheck {
		ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeout)
		err := checkHealth(ctx, healthcheckURL())
		cancel()
		if err != nil {
			log.Printf("unhealthy: %v", err)
			os.Exit(1)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, ":"+port())
	stop()
	if err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}

// run serves on addr until ctx is cancelled, then shuts down gracefully.
// It returns an error instead of exiting so deferred cleanup always runs.
func run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           newMux(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Buffered so the goroutine can exit even after run has returned.
	serveErr := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", addr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		// ListenAndServe only returns before Shutdown on failure.
		return fmt.Errorf("listen on %s: %w", addr, err)
	case <-ctx.Done():
	}

	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}

// newMux wires all routes.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	// No method in the pattern: the handler answers 405 with a JSON body.
	mux.Handle("/api/v1/calculate", handler.New(calculator.Calculate))
	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// checkHealth reports whether GET url answers 200 before ctx is done.
func checkHealth(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return nil
}

func healthcheckURL() string {
	return "http://127.0.0.1:" + port() + "/healthz"
}

func port() string {
	return envOr("PORT", "8080")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
