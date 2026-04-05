package service

import (
	"context"
	"log"
	"time"

	"github.com/Omotolani98/foostash/internal/store"
)

type AuditService struct {
	store *store.AuditStore
	queue chan store.AuditEntry
}

func NewAuditService(s *store.AuditStore) *AuditService {
	a := &AuditService{
		store: s,
		queue: make(chan store.AuditEntry, 256),
	}
	go a.run()
	return a
}

func (a *AuditService) run() {
	for e := range a.queue {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := a.store.Insert(ctx, e); err != nil {
			log.Printf("audit: insert failed: %v", err)
		}
		cancel()
	}
}

func (a *AuditService) LogAsync(e store.AuditEntry) {
	select {
	case a.queue <- e:
	default:
		log.Printf("audit: queue full, dropping entry action=%s", e.Action)
	}
}
