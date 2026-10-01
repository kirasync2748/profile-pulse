package counter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// upstashResponse is the body of an Upstash Redis REST API reply.
type upstashResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
}

// UpstashStore implements Store using the Upstash Redis REST API. This is the
// production backend: it works from Vercel serverless functions over plain
// HTTPS and requires no persistent connection.
type UpstashStore struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewUpstashStore creates an UpstashStore. The URL should be the REST base URL
// (e.g. https://xxx.upstash.io) and the token the Upstash REST token.
func NewUpstashStore(url, token string) *UpstashStore {
	return &UpstashStore{
		baseURL: strings.TrimRight(url, "/"),
		token:   token,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *UpstashStore) Incr(ctx context.Context, key string) (int64, error) {
	// Upstash REST API expects a POST to the base URL with a JSON array body
	// containing the command and its arguments: ["INCR", key]
	body, err := json.Marshal([]string{"INCR", key})
	if err != nil {
		return 0, ErrUnavailable
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return 0, ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, ErrUnavailable
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, ErrUnavailable
	}
	if resp.StatusCode >= 500 {
		return 0, ErrUnavailable
	}

	var ur upstashResponse
	if err := json.Unmarshal(raw, &ur); err != nil {
		return 0, ErrUnavailable
	}
	if ur.Error != "" {
		return 0, ErrUnavailable
	}

	switch v := ur.Result.(type) {
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, ErrUnavailable
		}
		return n, nil
	case float64:
		return int64(v), nil
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, ErrUnavailable
		}
		return n, nil
	default:
		return 0, ErrUnavailable
	}
}

func (s *UpstashStore) Close() error { return nil }

// LocalStore implements Store using a standard Redis connection (go-redis).
// It is intended for local development and testing against a local Redis
// instance; production uses UpstashStore.
type LocalStore struct {
	client *redis.Client
}

// NewLocalStore creates a LocalStore from a Redis URL (e.g. redis://localhost:6379).
// It pings Redis once to verify the connection is reachable; a failure here
// means the caller can fall back to a no-op store instead of serving every
// badge as "Counter unavailable".
func NewLocalStore(redisURL string) (*LocalStore, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &LocalStore{client: client}, nil
}

func (s *LocalStore) Incr(ctx context.Context, key string) (int64, error) {
	n, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, ErrUnavailable
	}
	return n, nil
}

func (s *LocalStore) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}

// errStore wraps any error from a store backend, mapping unknown errors to
// ErrUnavailable so callers never receive backend-specific details.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrUnavailable) {
		return err
	}
	return ErrUnavailable
}
