// Package messaging NATSメッセージングユーティリティ
package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

// Client NATS接続をラップしpublish/subscribe機能を提供
type Client struct {
	conn   *nats.Conn
	logger *slog.Logger
}

// NewClient リトライロジック付きで新規NATSクライアントを生成
func NewClient(ctx context.Context, logger *slog.Logger) (*Client, error) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	var conn *nats.Conn
	var err error
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		conn, err = nats.Connect(
			natsURL,
			nats.RetryOnFailedConnect(true),
			nats.MaxReconnects(10),
			nats.ReconnectWait(time.Second),
			nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
				if err != nil {
					logger.Warn("NATS disconnected", "error", err)
				}
			}),
			nats.ReconnectHandler(func(_ *nats.Conn) {
				logger.Info("NATS reconnected")
			}),
		)
		if err == nil {
			logger.Info("Connected to NATS", "url", natsURL)
			return &Client{conn: conn, logger: logger}, nil
		}

		logger.Warn("Failed to connect to NATS, retrying...", "attempt", i+1, "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(i+1) * time.Second):
		}
	}

	return nil, fmt.Errorf("connect to NATS after %d retries: %w", maxRetries, err)
}

// Publish 指定subjectにメッセージを発行
func (c *Client) Publish(subject string, data []byte) error {
	if err := c.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	c.logger.Debug("Published message", "subject", subject, "size", len(data))
	return nil
}

// Subscribe ロードバランシング用キューグループでsubjectを購読
func (c *Client) Subscribe(subject, queue string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	sub, err := c.conn.QueueSubscribe(subject, queue, handler)
	if err != nil {
		return nil, fmt.Errorf("subscribe to %s: %w", subject, err)
	}
	c.logger.Info("Subscribed to subject", "subject", subject, "queue", queue)
	return sub, nil
}

// Close NATS接続をクローズ
func (c *Client) Close() {
	if c.conn != nil {
		_ = c.conn.Drain()
		c.conn.Close()
		c.logger.Info("NATS connection closed")
	}
}

// Conn 内部NATS接続を返却
func (c *Client) Conn() *nats.Conn {
	return c.conn
}
