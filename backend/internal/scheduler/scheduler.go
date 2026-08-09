package scheduler

import (
	"context"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"log/slog"
	"time"
)

type Scheduler struct {
	store  *repository.Store
	logger *slog.Logger
}

func New(s *repository.Store, l *slog.Logger) *Scheduler { return &Scheduler{s, l} }
func (s *Scheduler) Run(ctx context.Context) {
	s.run(ctx)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.run(ctx)
		}
	}
}
func (s *Scheduler) run(ctx context.Context) {
	c, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	n, acquired, e := s.store.PublishDueNotificationsLocked(c, time.Now().Add(7*24*time.Hour))
	if e != nil {
		if ctx.Err() == nil {
			s.logger.Error("scheduler failed", "error", e)
		}
		return
	}
	if !acquired {
		s.logger.Debug("scheduler cycle handled by another instance")
		return
	}
	if n > 0 {
		s.logger.Info("notifications created", "count", n)
	}
}
