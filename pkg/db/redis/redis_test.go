package redis

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SCE-Development/SCEvents/pkg/db/stores"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestRedisStore(t *testing.T) stores.RedisStore {
	t.Helper()

	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := goredis.NewClient(&goredis.Options{
		Addr: srv.Addr(),
	})

	store := NewRedisStore(client)

	t.Cleanup(func() {
		_ = store.Close()
		srv.Close()
	})

	return store
}

func TestSetAndGetEventHeadcount(t *testing.T) {
	store := newTestRedisStore(t)

	if err := store.SetEventHeadcount(context.Background(), "event-1", 10); err != nil {
		t.Fatalf("SetEventHeadcount failed: %v", err)
	}

	got, err := store.GetEventHeadcount(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("GetEventHeadcount failed: %v", err)
	}
	if got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
}

func TestTryTakeEventSeatSuccessAndDecrement(t *testing.T) {
	store := newTestRedisStore(t)

	if err := store.SetEventHeadcount(context.Background(), "event-1", 2); err != nil {
		t.Fatalf("SetEventHeadcount failed: %v", err)
	}

	ok, err := store.TryTakeEventSeat(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("TryTakeEventSeat failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected seat acquisition to succeed")
	}

	remaining, err := store.GetEventHeadcount(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("GetEventHeadcount failed: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected remaining 1, got %d", remaining)
	}
}

func TestTryTakeEventSeatFullEvent(t *testing.T) {
	store := newTestRedisStore(t)

	if err := store.SetEventHeadcount(context.Background(), "event-1", 0); err != nil {
		t.Fatalf("SetEventHeadcount failed: %v", err)
	}

	ok, err := store.TryTakeEventSeat(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("TryTakeEventSeat failed: %v", err)
	}
	if ok {
		t.Fatalf("expected seat acquisition to fail when full")
	}

	remaining, err := store.GetEventHeadcount(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("GetEventHeadcount failed: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", remaining)
	}
}

func TestTryTakeEventSeatMissingHeadcount(t *testing.T) {
	store := newTestRedisStore(t)

	ok, err := store.TryTakeEventSeat(context.Background(), "missing-event")
	if err == nil {
		t.Fatalf("expected error for missing event headcount")
	}
	if ok {
		t.Fatalf("expected ok=false for missing event headcount")
	}
	if !strings.Contains(err.Error(), "missing Redis headcount") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseEventSeat(t *testing.T) {
	store := newTestRedisStore(t)

	if err := store.SetEventHeadcount(context.Background(), "event-1", 1); err != nil {
		t.Fatalf("SetEventHeadcount failed: %v", err)
	}
	if err := store.ReleaseEventSeat(context.Background(), "event-1"); err != nil {
		t.Fatalf("ReleaseEventSeat failed: %v", err)
	}

	remaining, err := store.GetEventHeadcount(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("GetEventHeadcount failed: %v", err)
	}
	if remaining != 2 {
		t.Fatalf("expected remaining 2, got %d", remaining)
	}
}

func TestTryTakeEventSeatConcurrentNoOversell(t *testing.T) {
	store := newTestRedisStore(t)

	const capacity = 25
	const attempts = 100

	if err := store.SetEventHeadcount(context.Background(), "event-1", capacity); err != nil {
		t.Fatalf("SetEventHeadcount failed: %v", err)
	}

	var wg sync.WaitGroup
	var successCount atomic.Int64

	wg.Add(attempts)
	for range attempts {
		go func() {
			defer wg.Done()
			ok, err := store.TryTakeEventSeat(context.Background(), "event-1")
			if err != nil {
				t.Errorf("TryTakeEventSeat failed: %v", err)
				return
			}
			if ok {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := int(successCount.Load()); got != capacity {
		t.Fatalf("expected %d successes, got %d", capacity, got)
	}

	remaining, err := store.GetEventHeadcount(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("GetEventHeadcount failed: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", remaining)
	}
}
