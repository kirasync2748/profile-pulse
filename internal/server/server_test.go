package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kirasync2748/profile-pulse/internal/counter"
)

// testStore is a mock counter.Store for HTTP handler tests.
type testStore struct {
	value int64
	err   error
}

func (m *testStore) Incr(_ context.Context, _ string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.value++
	return m.value, nil
}

func (m *testStore) Close() error { return nil }

// --- RootHandler ------------------------------------------------------------

func TestRootHandler(t *testing.T) {
	srv := New(&testStore{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.RootHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
}

// --- HealthHandler ----------------------------------------------------------

func TestHealthHandler(t *testing.T) {
	srv := New(&testStore{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.HealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

// --- BadgeHandler -----------------------------------------------------------

func TestBadgeHandler(t *testing.T) {
	srv := New(&testStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/badge?username=testuser", nil)
	rec := httptest.NewRecorder()

	srv.BadgeHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml; charset=utf-8" {
		t.Errorf("Content-Type = %q, want image/svg+xml; charset=utf-8", ct)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "<svg") {
		t.Error("response is not an SVG")
	}
	if !strings.HasSuffix(body, "</svg>") && !strings.HasSuffix(body, "/>") {
		t.Error("response SVG incomplete")
	}
}

func TestBadgeHandlerWithOptions(t *testing.T) {
	srv := New(&testStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/badge?username=testuser&color=red&style=flat-square&label=VIEWS&base=100&abbreviated=true", nil)
	rec := httptest.NewRecorder()

	srv.BadgeHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "VIEWS") {
		t.Error("missing label text in SVG")
	}
	if !strings.Contains(body, "#e05d44") {
		t.Error("missing color in SVG")
	}
}

func TestBadgeHandlerInvalidUsername(t *testing.T) {
	srv := New(&testStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/badge?username=", nil)
	rec := httptest.NewRecorder()

	srv.BadgeHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml; charset=utf-8" {
		t.Errorf("Content-Type = %q, want SVG", ct)
	}
}

func TestBadgeHandlerStoreError(t *testing.T) {
	srv := New(&testStore{err: counter.ErrUnavailable})
	req := httptest.NewRequest(http.MethodGet, "/api/badge?username=testuser", nil)
	rec := httptest.NewRecorder()

	srv.BadgeHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestBadgeHandlerInvalidColor(t *testing.T) {
	srv := New(&testStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/badge?username=testuser&color=xxx", nil)
	rec := httptest.NewRecorder()

	srv.BadgeHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// --- CountHandler -----------------------------------------------------------

func TestCountHandler(t *testing.T) {
	srv := New(&testStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/count?username=testuser", nil)
	rec := httptest.NewRecorder()

	srv.CountHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["username"] != "testuser" {
		t.Errorf("username = %v, want testuser", body["username"])
	}
	if body["views"].(float64) != 1 {
		t.Errorf("views = %v, want 1", body["views"])
	}
	if body["raw"].(float64) != 1 {
		t.Errorf("raw = %v, want 1", body["raw"])
	}
}

func TestCountHandlerIncrement(t *testing.T) {
	srv := New(&testStore{})

	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/count?username=testuser", nil)
		rec := httptest.NewRecorder()
		srv.CountHandler(rec, req)

		var body map[string]interface{}
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if body["views"].(float64) != float64(i) {
			t.Errorf("call %d: views = %v, want %d", i, body["views"], i)
		}
	}
}

func TestCountHandlerWithBase(t *testing.T) {
	srv := New(&testStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/count?username=testuser&base=1000", nil)
	rec := httptest.NewRecorder()

	srv.CountHandler(rec, req)

	var body map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body["views"].(float64) != 1001 {
		t.Errorf("views = %v, want 1001", body["views"])
	}
	if body["raw"].(float64) != 1 {
		t.Errorf("raw = %v, want 1", body["raw"])
	}
}

func TestCountHandlerInvalidUsername(t *testing.T) {
	srv := New(&testStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/count?username=<script>", nil)
	rec := httptest.NewRecorder()

	srv.CountHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var body map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

func TestCountHandlerStoreError(t *testing.T) {
	srv := New(&testStore{err: counter.ErrUnavailable})
	req := httptest.NewRequest(http.MethodGet, "/api/count?username=testuser", nil)
	rec := httptest.NewRecorder()

	srv.CountHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
