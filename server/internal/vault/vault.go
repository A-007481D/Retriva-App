package vault

import (
	"context"
	"log/slog"
	"time"

	"github.com/A-007481D/retriva/server/internal/media"
	"github.com/A-007481D/retriva/server/internal/storage"
)

// Service manages the lifecycle of media files, such as expiration and deletion.
type Service struct {
	repo    media.Repository
	storage storage.Storage
	logger  *slog.Logger
	stop    chan struct{}
}

// NewService creates a new vault service.
func NewService(repo media.Repository, st storage.Storage, logger *slog.Logger) *Service {
	return &Service{
		repo:    repo,
		storage: st,
		logger:  logger,
		stop:    make(chan struct{}),
	}
}

// Start begins a background goroutine to periodically clean up expired media.
func (s *Service) Start(interval time.Duration) {
	s.logger.Info("vault service started", slog.Duration("interval", interval))
	
	// Run cleanup once immediately
	s.cleanup(context.Background())

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.cleanup(context.Background())
			case <-s.stop:
				return
			}
		}
	}()
}

// Stop halts the background goroutine.
func (s *Service) Stop() {
	close(s.stop)
	s.logger.Info("vault service stopped")
}

func (s *Service) cleanup(ctx context.Context) {
	expired, err := s.repo.ListExpired(ctx)
	if err != nil {
		s.logger.Error("failed to list expired media", slog.Any("error", err))
		return
	}

	for _, m := range expired {
		if m.Status == media.StatusActive {
			s.logger.Info("expiring media", slog.String("media_id", m.ID))
			
			if m.StorageKey != "" {
				err := s.storage.Delete(ctx, m.StorageKey)
				// It's okay if it's already deleted
				if err != nil && err.Error() != storage.ErrNotFound.Error() {
					s.logger.Error("failed to delete file", slog.String("key", m.StorageKey), slog.Any("error", err))
					continue
				}
			}

			m.Status = media.StatusExpired
			m.StorageKey = ""
			
			if err := s.repo.Update(ctx, m); err != nil {
				s.logger.Error("failed to update media status to expired", slog.String("media_id", m.ID), slog.Any("error", err))
			}
		}
	}
}
