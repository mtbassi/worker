package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"worker-project/shared/domain"
)

// getTestClient creates a Redis client for testing.
// Skips the test if Redis is not available.
func getTestClient(t *testing.T) *Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client, err := NewClient(Config{
		Addr:         addr,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           15, // Use a separate DB for tests
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     5,
		MinIdleConns: 1,
	})
	if err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
	}

	return client
}

func TestStateStore_SaveAndGetJourneyState(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewStateStore(client, 1*time.Hour)
	ctx := context.Background()

	state := &domain.JourneyState{
		JourneyID:         "test-journey",
		Step:              "step-1",
		CustomerNumber:    "5511999999999",
		TenantID:          "tenant-123",
		ContactID:         "contact-456",
		LastInteractionAt: time.Now(),
		StepStartedAt:     time.Now(),
		JourneyStartedAt:  time.Now().Add(-1 * time.Hour),
		Metadata:          map[string]any{"key": "value"},
	}

	// Clean up before test
	_ = store.DeleteJourneyState(ctx, state.JourneyID, state.CustomerNumber)

	// Save state
	err := store.SaveJourneyState(ctx, state)
	if err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Get state
	retrieved, err := store.GetJourneyState(ctx, state.JourneyID, state.CustomerNumber)
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}

	if retrieved.JourneyID != state.JourneyID {
		t.Errorf("JourneyID mismatch: got %s, want %s", retrieved.JourneyID, state.JourneyID)
	}
	if retrieved.Step != state.Step {
		t.Errorf("Step mismatch: got %s, want %s", retrieved.Step, state.Step)
	}
	if retrieved.CustomerNumber != state.CustomerNumber {
		t.Errorf("CustomerNumber mismatch: got %s, want %s", retrieved.CustomerNumber, state.CustomerNumber)
	}

	// Clean up
	_ = store.DeleteJourneyState(ctx, state.JourneyID, state.CustomerNumber)
}

func TestStateStore_GetJourneyState_NotFound(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewStateStore(client, 1*time.Hour)
	ctx := context.Background()

	_, err := store.GetJourneyState(ctx, "nonexistent-journey", "nonexistent-customer")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStateStore_DeleteJourneyState(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewStateStore(client, 1*time.Hour)
	ctx := context.Background()

	state := &domain.JourneyState{
		JourneyID:         "test-journey-delete",
		Step:              "step-1",
		CustomerNumber:    "5511888888888",
		TenantID:          "tenant-123",
		LastInteractionAt: time.Now(),
	}

	// Save and then delete
	_ = store.SaveJourneyState(ctx, state)
	err := store.DeleteJourneyState(ctx, state.JourneyID, state.CustomerNumber)
	if err != nil {
		t.Fatalf("failed to delete state: %v", err)
	}

	// Verify deleted
	_, err = store.GetJourneyState(ctx, state.JourneyID, state.CustomerNumber)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestStateStore_GetRepiqueHistory_Empty(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewStateStore(client, 1*time.Hour)
	ctx := context.Background()

	// Clean up first
	key := "journey:test-empty:5511777777777:repiques"
	_ = client.Del(ctx, key)

	history, err := store.GetRepiqueHistory(ctx, "test-empty", "5511777777777")
	if err != nil {
		t.Fatalf("failed to get empty history: %v", err)
	}

	if len(history.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(history.Entries))
	}
}

func TestStateStore_AppendRepiqueHistory_Atomic(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewStateStore(client, 1*time.Hour)
	ctx := context.Background()

	journeyID := "test-atomic"
	customerNumber := "5511666666666"

	// Clean up first
	key := "journey:test-atomic:5511666666666:repiques"
	_ = client.Del(ctx, key)

	// Append first entry
	entry1 := domain.RepiqueEntry{
		Step:          "step-1",
		Rule:          "rule-1",
		SentAt:        time.Now(),
		TemplateUsed:  "template-1",
		AttemptNumber: 1,
	}

	err := store.AppendRepiqueHistory(ctx, journeyID, customerNumber, entry1)
	if err != nil {
		t.Fatalf("failed to append first entry: %v", err)
	}

	// Append second entry
	entry2 := domain.RepiqueEntry{
		Step:          "step-1",
		Rule:          "rule-2",
		SentAt:        time.Now(),
		TemplateUsed:  "template-2",
		AttemptNumber: 1,
	}

	err = store.AppendRepiqueHistory(ctx, journeyID, customerNumber, entry2)
	if err != nil {
		t.Fatalf("failed to append second entry: %v", err)
	}

	// Verify both entries exist
	history, err := store.GetRepiqueHistory(ctx, journeyID, customerNumber)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(history.Entries))
	}

	// Clean up
	_ = client.Del(ctx, key)
}

func TestStateStore_AcquireMessageLock(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewStateStore(client, 1*time.Hour)
	ctx := context.Background()

	journeyID := "test-lock"
	customerNumber := "5511555555555"
	ruleName := "test-rule"
	attemptNumber := 1

	// Clean up first
	lockKey := "journey:test-lock:5511555555555:lock:test-rule:1"
	_ = client.Del(ctx, lockKey)

	// First acquisition should succeed
	acquired1, err := store.AcquireMessageLock(ctx, journeyID, customerNumber, ruleName, attemptNumber)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	if !acquired1 {
		t.Error("first acquisition should succeed")
	}

	// Second acquisition should fail (already locked)
	acquired2, err := store.AcquireMessageLock(ctx, journeyID, customerNumber, ruleName, attemptNumber)
	if err != nil {
		t.Fatalf("failed on second acquisition: %v", err)
	}
	if acquired2 {
		t.Error("second acquisition should fail (duplicate)")
	}

	// Different attempt number should succeed
	acquired3, err := store.AcquireMessageLock(ctx, journeyID, customerNumber, ruleName, attemptNumber+1)
	if err != nil {
		t.Fatalf("failed on different attempt: %v", err)
	}
	if !acquired3 {
		t.Error("different attempt number should succeed")
	}

	// Clean up
	_ = client.Del(ctx, lockKey)
	_ = client.Del(ctx, "journey:test-lock:5511555555555:lock:test-rule:2")
}

func TestStateStore_UpdateLastInteractionAt(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewStateStore(client, 1*time.Hour)
	ctx := context.Background()

	originalTime := time.Now().Add(-30 * time.Minute)
	state := &domain.JourneyState{
		JourneyID:         "test-update-interaction",
		Step:              "step-1",
		CustomerNumber:    "5511444444444",
		TenantID:          "tenant-123",
		LastInteractionAt: originalTime,
	}

	// Clean up and setup
	_ = store.DeleteJourneyState(ctx, state.JourneyID, state.CustomerNumber)
	_ = store.SaveJourneyState(ctx, state)

	// Update LastInteractionAt
	newTime := time.Now()
	err := store.UpdateLastInteractionAt(ctx, state.JourneyID, state.CustomerNumber, newTime)
	if err != nil {
		t.Fatalf("failed to update LastInteractionAt: %v", err)
	}

	// Verify update
	updated, err := store.GetJourneyState(ctx, state.JourneyID, state.CustomerNumber)
	if err != nil {
		t.Fatalf("failed to get updated state: %v", err)
	}

	// Check that LastInteractionAt was updated (within 1 second tolerance)
	timeDiff := updated.LastInteractionAt.Sub(newTime)
	if timeDiff < -time.Second || timeDiff > time.Second {
		t.Errorf("LastInteractionAt not updated correctly: got %v, want %v", updated.LastInteractionAt, newTime)
	}

	// Clean up
	_ = store.DeleteJourneyState(ctx, state.JourneyID, state.CustomerNumber)
}
