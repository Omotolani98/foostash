// Package sshsrv serves the Foostash TUI over SSH via Wish. Transport only:
// authentication resolves an AuthContext, which is stashed for the TUI to
// read; no service logic lives here.
package sshsrv

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/Omotolani98/foostash/internal/service"
	"charm.land/wish/v2"
	"charm.land/wish/v2/bubbletea"
	"github.com/charmbracelet/ssh"
)

// Config controls SSH server startup.
type Config struct {
	Addr        string // e.g. ":2222"; empty disables the SSH server
	HostKeyPath string
	IdleTimeout time.Duration
}

// Deps is the dependency bundle for the SSH server.
type Deps struct {
	Auth *service.Auth
	TUI  TUIDeps
}

// Server wraps a wish SSH server.
type Server struct {
	cfg    Config
	srv    *ssh.Server
	addr   string
}

// New constructs an SSH server. Must be called after EnsureHostKey.
func New(cfg Config, deps Deps) (*Server, error) {
	if cfg.Addr == "" {
		return nil, errors.New("ssh addr is empty")
	}
	if cfg.HostKeyPath == "" {
		return nil, errors.New("ssh host key path is empty")
	}
	if err := EnsureHostKey(cfg.HostKeyPath); err != nil {
		return nil, err
	}

	idle := cfg.IdleTimeout
	if idle == 0 {
		idle = 5 * time.Minute
	}

	s, err := wish.NewServer(
		wish.WithAddress(cfg.Addr),
		wish.WithHostKeyPath(cfg.HostKeyPath),
		wish.WithIdleTimeout(idle),
		wish.WithPublicKeyAuth(publicKeyHandler(deps.Auth)),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler(deps.TUI)),
			sessionLogger(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build ssh server: %w", err)
	}

	return &Server{cfg: cfg, srv: s, addr: cfg.Addr}, nil
}

// Run listens until ctx is canceled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("ssh server listening", "addr", s.addr)
		err := s.srv.ListenAndServe()
		if err != nil && !errors.Is(err, ssh.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

// Shutdown gracefully closes the SSH listener, waiting up to 10s for sessions.
func (s *Server) Shutdown(ctx context.Context) error {
	shutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(shutCtx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		return err
	}
	return nil
}

// sessionLogger logs session start/end with the resolved user.
func sessionLogger() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			actx := AuthFromContext(sess.Context())
			email := "(anonymous)"
			if actx != nil {
				email = actx.Email
			}
			start := time.Now()
			next(sess)
			slog.Info("ssh session end",
				"user", email,
				"remote", sess.RemoteAddr().String(),
				"duration", time.Since(start).String(),
			)
		}
	}
}
