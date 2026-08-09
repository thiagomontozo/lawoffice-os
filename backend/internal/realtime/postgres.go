package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const notificationChannel = "lawoffice_realtime"

type envelope struct {
	FirmID string `json:"firmId"`
	Event  Event  `json:"event"`
}

func StartPostgres(ctx context.Context, pool *pgxpool.Pool, hub *Hub, logger *slog.Logger) error {
	connection, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	if _, err = connection.Exec(ctx, "LISTEN "+notificationChannel); err != nil {
		connection.Release()
		return err
	}
	hub.SetPublisher(func(firmID string, event Event) error {
		payload, marshalErr := json.Marshal(envelope{FirmID: firmID, Event: event})
		if marshalErr != nil {
			return marshalErr
		}
		publishContext, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		_, publishErr := pool.Exec(publishContext, `SELECT pg_notify($1,$2)`, notificationChannel, string(payload))
		return publishErr
	})
	go func() {
		defer connection.Release()
		for {
			notification, waitErr := connection.Conn().WaitForNotification(ctx)
			if waitErr != nil {
				if ctx.Err() == nil {
					logger.Error("realtime listener stopped", "error", waitErr)
				}
				return
			}
			var message envelope
			if jsonErr := json.Unmarshal([]byte(notification.Payload), &message); jsonErr != nil || message.FirmID == "" || message.Event.Type == "" {
				logger.Warn("invalid realtime database notification")
				continue
			}
			hub.Deliver(message.FirmID, message.Event)
		}
	}()
	return nil
}
