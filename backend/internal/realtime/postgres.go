package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
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
		publishContext, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		transaction, beginErr := pool.Begin(publishContext)
		if beginErr != nil {
			return beginErr
		}
		defer func() { _ = transaction.Rollback(publishContext) }()
		var id int64
		if insertErr := transaction.QueryRow(publishContext, `INSERT INTO realtime_events(firm_id,event_type,resource_type,resource_id) VALUES($1,$2,$3,$4) RETURNING id,published_at`, firmID, event.Type, event.ResourceType, event.ResourceID).Scan(&id, &event.PublishedAt); insertErr != nil {
			return insertErr
		}
		event.ID = strconv.FormatInt(id, 10)
		payload, marshalErr := json.Marshal(envelope{FirmID: firmID, Event: event})
		if marshalErr != nil {
			return marshalErr
		}
		if _, publishErr := transaction.Exec(publishContext, `SELECT pg_notify($1,$2)`, notificationChannel, string(payload)); publishErr != nil {
			return publishErr
		}
		return transaction.Commit(publishContext)
	})
	hub.SetPublishErrorHandler(func(publishErr error) {
		logger.Error("durable realtime publish failed; delivering locally", "error", publishErr)
	})
	hub.SetReplay(func(replayContext context.Context, firmID, lastID string) ([]Event, error) {
		if lastID == "" {
			return []Event{}, nil
		}
		id, parseErr := strconv.ParseInt(lastID, 10, 64)
		if parseErr != nil || id < 0 {
			return []Event{}, nil
		}
		rows, queryErr := pool.Query(replayContext, `SELECT id,event_type,resource_type,resource_id,published_at FROM realtime_events WHERE firm_id=$1 AND id>$2 AND expires_at>now() ORDER BY id LIMIT 500`, firmID, id)
		if queryErr != nil {
			return nil, queryErr
		}
		defer rows.Close()
		events := []Event{}
		for rows.Next() {
			var event Event
			var eventID int64
			if scanErr := rows.Scan(&eventID, &event.Type, &event.ResourceType, &event.ResourceID, &event.PublishedAt); scanErr != nil {
				return nil, scanErr
			}
			event.ID = strconv.FormatInt(eventID, 10)
			events = append(events, event)
		}
		return events, rows.Err()
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
