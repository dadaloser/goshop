package db

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"goshop/app/pkg/code"
	dv1 "goshop/app/user/srv/internal/data/v1"
	"goshop/pkg/errors"
)

func TestRotateSessionAllowsOnlyOneConcurrentRefreshRealDB(t *testing.T) {
	db, _ := mustOpenSchemaIntegrationDB(t)
	prepareUserSchemaMigrations(t, db)

	store := &users{db: db}
	now := time.Now().UTC().Truncate(time.Millisecond)
	currentHash := repeatedSessionHash(1)
	session := &dv1.UserSessionDO{
		ID:               "session-refresh-e2e-1",
		UserID:           1001,
		RefreshTokenHash: currentHash,
		DeviceID:         "device-1",
		DeviceName:       "iPhone 16",
		CreatedAt:        now,
		LastUsedAt:       now,
		ExpiresAt:        now.Add(30 * time.Minute),
	}
	if err := store.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	const attempts = 12
	type result struct {
		hash []byte
		err  error
	}

	start := make(chan struct{})
	results := make(chan result, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			nextHash := repeatedSessionHash(byte(index + 2))
			<-start
			_, err := store.RotateSession(
				context.Background(),
				session.ID,
				currentHash,
				nextHash,
				now.Add(2*time.Hour+time.Duration(index)*time.Minute),
				now.Add(time.Duration(index+1)*time.Second),
			)
			results <- result{hash: nextHash, err: err}
		}(i)
	}

	close(start)
	wg.Wait()
	close(results)

	var success int
	var winnerHash []byte
	for item := range results {
		switch {
		case item.err == nil:
			success++
			winnerHash = item.hash
		case errors.IsCode(item.err, code.ErrUserAccountInactive):
		default:
			t.Fatalf("RotateSession() unexpected error = %v", item.err)
		}
	}

	if success != 1 {
		t.Fatalf("successful refreshes = %d, want 1", success)
	}

	var stored dv1.UserSessionDO
	if err := db.WithContext(context.Background()).First(&stored, "id = ?", session.ID).Error; err != nil {
		t.Fatalf("reload session error = %v", err)
	}
	if !bytes.Equal(stored.RefreshTokenHash, winnerHash) {
		t.Fatalf("stored hash = %x, want winner %x", stored.RefreshTokenHash, winnerHash)
	}
	if !stored.LastUsedAt.After(now) {
		t.Fatalf("last_used_at = %v, want after %v", stored.LastUsedAt, now)
	}
}

func repeatedSessionHash(fill byte) []byte {
	return bytes.Repeat([]byte{fill}, 32)
}
