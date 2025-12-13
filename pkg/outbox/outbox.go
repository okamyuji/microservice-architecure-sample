// Package outbox トランザクショナルOutboxパターン実装
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Message Outboxメッセージ
type Message struct {
	ID            string          `db:"id"`
	AggregateType string          `db:"aggregate_type"`
	AggregateID   string          `db:"aggregate_id"`
	EventType     string          `db:"event_type"`
	Payload       json.RawMessage `db:"payload"`
	Status        string          `db:"status"`
	RetryCount    int             `db:"retry_count"`
	CreatedAt     time.Time       `db:"created_at"`
	ProcessedAt   *time.Time      `db:"processed_at"`
}

// ステータス定数
const (
	StatusPending = "PENDING"
	StatusSent    = "SENT"
	StatusFailed  = "FAILED"
	MaxRetryCount = 5
)

// Publisher トランザクション内でイベントをOutboxテーブルに書き込む
type Publisher struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPublisher 新規Outbox Publisherを生成
func NewPublisher(pool *pgxpool.Pool, logger *slog.Logger) *Publisher {
	return &Publisher{pool: pool, logger: logger}
}

// PublishInTx 指定トランザクション内でイベントをOutboxテーブルに書き込む
func (p *Publisher) PublishInTx(ctx context.Context, tx pgx.Tx, aggregateType, aggregateID, eventType string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	id := uuid.New().String()
	query := `
		INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(ctx, query, id, aggregateType, aggregateID, eventType, payloadBytes, StatusPending)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	p.logger.Debug("Event written to outbox",
		"id", id,
		"aggregate_type", aggregateType,
		"aggregate_id", aggregateID,
		"event_type", eventType,
	)

	return nil
}

// Relay Outboxから未送信メッセージを読み取り指定関数でpublish
type Relay struct {
	pool      *pgxpool.Pool
	logger    *slog.Logger
	publisher func(subject string, data []byte) error
	interval  time.Duration
}

// NewRelay 新規Outbox Relayを生成
func NewRelay(pool *pgxpool.Pool, logger *slog.Logger, publisher func(subject string, data []byte) error) *Relay {
	return &Relay{
		pool:      pool,
		logger:    logger,
		publisher: publisher,
		interval:  5 * time.Second,
	}
}

// Start ポーリングループを開始
func (r *Relay) Start(ctx context.Context) {
	r.logger.Info("Starting outbox relay", "interval", r.interval)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Outbox relay stopped")
			return
		case <-ticker.C:
			if err := r.processPendingMessages(ctx); err != nil {
				r.logger.Error("Failed to process pending messages", "error", err)
			}
		}
	}
}

func (r *Relay) processPendingMessages(ctx context.Context) error {
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, created_at
		FROM outbox
		WHERE status = $1 AND retry_count < $2
		ORDER BY created_at ASC
		LIMIT 100
		FOR UPDATE SKIP LOCKED
	`

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, query, StatusPending, MaxRetryCount)
	if err != nil {
		return fmt.Errorf("query pending messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(
			&msg.ID, &msg.AggregateType, &msg.AggregateID,
			&msg.EventType, &msg.Payload, &msg.Status,
			&msg.RetryCount, &msg.CreatedAt,
		); err != nil {
			return fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}

	for _, msg := range messages {
		if err := r.publishMessage(ctx, tx, msg); err != nil {
			r.logger.Error("Failed to publish message",
				"id", msg.ID,
				"event_type", msg.EventType,
				"error", err,
			)
			// リトライカウント増加
			if _, err := tx.Exec(ctx,
				`UPDATE outbox SET retry_count = retry_count + 1 WHERE id = $1`,
				msg.ID,
			); err != nil {
				r.logger.Error("Failed to update retry count", "id", msg.ID, "error", err)
			}
			continue
		}

		// 送信済みとしてマーク
		now := time.Now()
		if _, err := tx.Exec(ctx,
			`UPDATE outbox SET status = $1, processed_at = $2 WHERE id = $3`,
			StatusSent, now, msg.ID,
		); err != nil {
			return fmt.Errorf("mark message as sent: %w", err)
		}

		r.logger.Info("Message published",
			"id", msg.ID,
			"event_type", msg.EventType,
			"aggregate_id", msg.AggregateID,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *Relay) publishMessage(_ context.Context, _ pgx.Tx, msg Message) error {
	return r.publisher(msg.EventType, msg.Payload)
}
