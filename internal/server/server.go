package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Omotolani98/foostash/internal/sshsrv"
)

// Server runs the HTTP API and, optionally, the SSH TUI transport.
type Server struct {
	httpServer *http.Server
	ssh        *sshsrv.Server
}

// Options controls server construction.
type Options struct {
	HTTPAddr       string
	SSHAddr        string // empty disables SSH
	SSHHostKeyPath string
}

func New(opts Options, d *Deps) (*Server, error) {
	s := &Server{
		httpServer: &http.Server{
			Addr:              opts.HTTPAddr,
			Handler:           NewRouter(d),
			ReadHeaderTimeout: 10 * time.Second,
		},
	}

	if opts.SSHAddr != "" {
		ssh, err := sshsrv.New(sshsrv.Config{
			Addr:        opts.SSHAddr,
			HostKeyPath: opts.SSHHostKeyPath,
		}, sshsrv.Deps{
			Auth: d.Auth,
			TUI: sshsrv.TUIDeps{
				Projects: d.Projects,
				Users:    d.Users,
				Audit:    d.Audit,
				Secrets:  d.Secrets,
				Vault:    d.Vault,
				Invites:  d.Invites,
			},
		})
		if err != nil {
			return nil, err
		}
		s.ssh = ssh
	}

	return s, nil
}

// Run starts both listeners and blocks until ctx is canceled or one fails.
// On shutdown both listeners are stopped concurrently with a shared 10s deadline.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		slog.Info("http server listening", "addr", s.httpServer.Addr)
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	if s.ssh != nil {
		go func() {
			errCh <- s.ssh.Run(ctx)
		}()
	}

	select {
	case <-ctx.Done():
		return s.shutdown()
	case err := <-errCh:
		// One subsystem died; tear down the other.
		_ = s.shutdown()
		return err
	}
}

func (s *Server) shutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var httpErr, sshErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		httpErr = s.httpServer.Shutdown(shutdownCtx)
	}()

	if s.ssh != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sshErr = s.ssh.Shutdown(shutdownCtx)
		}()
	}
	wg.Wait()

	if httpErr != nil {
		return httpErr
	}
	return sshErr
}
