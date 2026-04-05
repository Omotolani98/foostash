package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Omotolani98/foostash/internal/config"
	"github.com/Omotolani98/foostash/internal/crypto"
	"github.com/Omotolani98/foostash/internal/handler"
	"github.com/Omotolani98/foostash/internal/router"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		if err := runServe(); err != nil {
			log.Fatalf("serve: %v", err)
		}
	case "migrate":
		if err := runMigrate(os.Args[2:]); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	case "genkey":
		key, err := crypto.GenerateKey()
		if err != nil {
			log.Fatalf("genkey: %v", err)
		}
		fmt.Println(key)
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`foostash server

Usage:
  foostash serve           Run the HTTP API server
  foostash migrate up      Apply pending migrations
  foostash migrate down N  Revert N migrations
  foostash genkey          Generate a base64 AES-256 master key`)
}

func runServe() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()
	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	masterKey, err := cfg.ResolveMasterKey()
	if err != nil {
		return err
	}
	engine, err := crypto.NewEngine(masterKey)
	if err != nil {
		return err
	}

	orgsStore := store.NewOrganizationsStore(db)
	usersStore := store.NewUsersStore(db)
	projectsStore := store.NewProjectsStore(db)
	envsStore := store.NewEnvironmentsStore(db)
	secretsStore := store.NewSecretsStore(db)
	apiKeysStore := store.NewAPIKeysStore(db)
	auditStore := store.NewAuditStore(db)

	auditSvc := service.NewAuditService(auditStore)
	authSvc := service.NewAuthService(usersStore, orgsStore, apiKeysStore, []byte(cfg.JWTSecret))
	projectsSvc := service.NewProjectsService(projectsStore, envsStore, orgsStore)
	secretsSvc := service.NewSecretsService(secretsStore, envsStore, engine, auditSvc)

	deps := &router.Dependencies{
		Auth:     handler.NewAuthHandler(authSvc),
		Projects: handler.NewProjectsHandler(projectsSvc),
		Secrets:  handler.NewSecretsHandler(secretsSvc),
		Health:   handler.NewHealthHandler(db),
		AuthSvc:  authSvc,
	}

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           router.New(deps),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("foostash server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func runMigrate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: foostash migrate up|down [N]")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()
	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	switch args[0] {
	case "up":
		n, err := store.MigrateUp(ctx, db, cfg.MigrationsDir)
		if err != nil {
			return err
		}
		log.Printf("applied %d migration(s)", n)
	case "down":
		steps := 1
		if len(args) > 1 {
			v, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid step count: %w", err)
			}
			steps = v
		}
		n, err := store.MigrateDown(ctx, db, cfg.MigrationsDir, steps)
		if err != nil {
			return err
		}
		log.Printf("reverted %d migration(s)", n)
	default:
		return fmt.Errorf("unknown migrate subcommand: %s", args[0])
	}
	return nil
}
