package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Server wraps an http.Server and the dependencies it serves.
type Server struct {
	httpServer *http.Server
}

func New(addr string, d *Deps) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           NewRouter(d),
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Run blocks until ctx is canceled, then triggers a graceful shutdown.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", s.httpServer.Addr)
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
