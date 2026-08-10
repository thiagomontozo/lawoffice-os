package scheduler

import (
	"context"
	"fmt"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/jobs"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/mailer"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"log/slog"
	"time"
)

type Scheduler struct {
	store  *repository.Store
	jobs   *jobs.Queue
	logger *slog.Logger
}

func New(s *repository.Store, queue *jobs.Queue, l *slog.Logger) *Scheduler {
	return &Scheduler{store: s, jobs: queue, logger: l}
}
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
	s.queueNotificationEmails(c)
}

func (s *Scheduler) queueNotificationEmails(ctx context.Context) {
	if s.jobs == nil || !s.jobs.Enabled() {
		return
	}
	items, err := s.store.PendingEmailNotifications(ctx, 100)
	if err != nil {
		s.logger.Error("pending notification email lookup failed", "error", err)
		return
	}
	for _, item := range items {
		message := mailer.Message{To: item.Email, Subject: item.Title, Text: fmt.Sprintf("Olá, %s.\n\n%s\n\nAcesse o LawOffice OS para consultar os detalhes com segurança.", item.Name, item.Message)}
		accepted, queueErr := s.jobs.EnsureEmail(ctx, item.FirmID, "notification:"+item.ID, message)
		if queueErr != nil {
			s.logger.Error("notification email enqueue failed", "notification_id", item.ID, "error", queueErr)
			continue
		}
		if accepted {
			if markErr := s.store.MarkNotificationEmailQueued(ctx, item.FirmID, item.ID); markErr != nil {
				s.logger.Error("notification email state update failed", "notification_id", item.ID, "error", markErr)
			}
		}
	}
}
