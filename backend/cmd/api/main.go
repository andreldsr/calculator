// Command api starts the calculator HTTP server.
package main

import (
	"context"
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

const shutdownTimeout = 10 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, ":"+envOr("PORT", "8080"))
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
