package counter

import (
	"context"
	"testing"
)

// mockStore is a test Store that returns a configurable sequence of values.
type mockStore struct {
	values []int64
	calls  int
	err    error
}

func (m *mockStore) Incr(ctx context.Context, key string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	if m.calls >= len(m.values) {
		return 0, ErrUnavailable
	}
	v := m.values[m.calls]
	m.calls++
	return v, nil
}

func (m *mockStore) Close() error { return nil }

func TestNoopStore(t *testing.T) {
	s := NoopStore{}
	if _, err := s.Incr(context.Background(), "k"); err != ErrUnavailable {
		t.Errorf("NoopStore.Incr err = %v, want ErrUnavailable", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("NoopStore.Close err = %v", err)
	}
}

func TestMockStoreSequence(t *testing.T) {
	m := &mockStore{values: []int64{1, 2, 3}}
	for i := int64(1); i <= 3; i++ {
		v, err := m.Incr(context.Background(), "pv:v1:user:test")
		if err != nil {
			t.Fatalf("Incr err = %v", err)
		}
		if v != i {
			t.Errorf("Incr = %d, want %d", v, i)
		}
	}
	if _, err := m.Incr(context.Background(), "k"); err != ErrUnavailable {
		t.Errorf("exhausted store err = %v, want ErrUnavailable", err)
	}
}

func TestMockStoreError(t *testing.T) {
	m := &mockStore{err: ErrUnavailable}
	if _, err := m.Incr(context.Background(), "k"); err != ErrUnavailable {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
}

func TestMapErr(t *testing.T) {
	if mapErr(nil) != nil {
		t.Error("mapErr(nil) should be nil")
	}
	if mapErr(ErrUnavailable) != ErrUnavailable {
		t.Error("mapErr should pass ErrUnavailable through")
	}
	if mapErr(context.DeadlineExceeded) != ErrUnavailable {
		t.Error("mapErr should wrap unknown errors as ErrUnavailable")
	}
}
