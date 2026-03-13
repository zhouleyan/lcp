package websocket

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireAndReleaseBasicFlow(t *testing.T) {
	m := NewSessionManager(5, 30*time.Minute)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess, err := m.Acquire("user-1", "host", "host-42", "root@10.0.1.5", cancel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.ID == "" {
		t.Fatal("expected non-empty session ID")
	}
	if sess.UserID != "user-1" {
		t.Fatalf("expected UserID %q, got %q", "user-1", sess.UserID)
	}
	if sess.Resource != "host" {
		t.Fatalf("expected Resource %q, got %q", "host", sess.Resource)
	}
	if sess.ResourceID != "host-42" {
		t.Fatalf("expected ResourceID %q, got %q", "host-42", sess.ResourceID)
	}
	if sess.Label != "root@10.0.1.5" {
		t.Fatalf("expected Label %q, got %q", "root@10.0.1.5", sess.Label)
	}
	if sess.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}

	if m.Count("user-1") != 1 {
		t.Fatalf("expected count 1, got %d", m.Count("user-1"))
	}

	m.Release(sess.ID)

	if m.Count("user-1") != 0 {
		t.Fatalf("expected count 0 after release, got %d", m.Count("user-1"))
	}
}

func TestConcurrencyLimitEnforcement(t *testing.T) {
	maxPerUser := 3
	m := NewSessionManager(maxPerUser, 30*time.Minute)

	// Acquire up to the limit
	ids := make([]string, 0, maxPerUser)
	for i := 0; i < maxPerUser; i++ {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		sess, err := m.Acquire("user-1", "host", "host-1", "root@10.0.1.5", cancel)
		if err != nil {
			t.Fatalf("unexpected error on acquire %d: %v", i, err)
		}
		ids = append(ids, sess.ID)
	}

	if m.Count("user-1") != maxPerUser {
		t.Fatalf("expected count %d, got %d", maxPerUser, m.Count("user-1"))
	}

	// One more should fail
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := m.Acquire("user-1", "host", "host-2", "root@10.0.1.6", cancel)
	if err == nil {
		t.Fatal("expected error when exceeding max sessions, got nil")
	}

	// Count should not change
	if m.Count("user-1") != maxPerUser {
		t.Fatalf("expected count %d after failed acquire, got %d", maxPerUser, m.Count("user-1"))
	}
}

func TestDifferentUsersHaveIndependentLimits(t *testing.T) {
	maxPerUser := 2
	m := NewSessionManager(maxPerUser, 30*time.Minute)

	// Fill up user-1
	for i := 0; i < maxPerUser; i++ {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		_, err := m.Acquire("user-1", "host", "host-1", "label", cancel)
		if err != nil {
			t.Fatalf("unexpected error for user-1 acquire %d: %v", i, err)
		}
	}

	// user-2 should still be able to acquire
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess, err := m.Acquire("user-2", "host", "host-1", "label", cancel)
	if err != nil {
		t.Fatalf("unexpected error for user-2: %v", err)
	}
	if sess.UserID != "user-2" {
		t.Fatalf("expected UserID %q, got %q", "user-2", sess.UserID)
	}

	if m.Count("user-1") != maxPerUser {
		t.Fatalf("expected user-1 count %d, got %d", maxPerUser, m.Count("user-1"))
	}
	if m.Count("user-2") != 1 {
		t.Fatalf("expected user-2 count 1, got %d", m.Count("user-2"))
	}
}

func TestReleaseNonExistentSession(t *testing.T) {
	m := NewSessionManager(5, 30*time.Minute)

	// Should not panic
	m.Release("non-existent-id")
	m.Release("")
}

func TestListReturnsSessions(t *testing.T) {
	m := NewSessionManager(5, 30*time.Minute)

	_, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	_, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	_, err := m.Acquire("user-1", "host", "host-1", "root@10.0.1.1", cancel1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = m.Acquire("user-1", "container", "ctr-5", "nginx-pod", cancel2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Also add a session for a different user
	_, cancel3 := context.WithCancel(context.Background())
	defer cancel3()
	_, err = m.Acquire("user-2", "host", "host-2", "admin@10.0.1.2", cancel3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sessions := m.List("user-1")
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions for user-1, got %d", len(sessions))
	}

	// Verify sessions are copies and cancel is nil
	for _, s := range sessions {
		if s.UserID != "user-1" {
			t.Fatalf("expected UserID %q, got %q", "user-1", s.UserID)
		}
		if s.ID == "" {
			t.Fatal("expected non-empty session ID in list")
		}
	}

	// Verify different user's sessions are separate
	sessions2 := m.List("user-2")
	if len(sessions2) != 1 {
		t.Fatalf("expected 1 session for user-2, got %d", len(sessions2))
	}

	// List for non-existent user should return empty
	sessions3 := m.List("user-3")
	if len(sessions3) != 0 {
		t.Fatalf("expected 0 sessions for user-3, got %d", len(sessions3))
	}
}

func TestCancelCallsCancelFunc(t *testing.T) {
	m := NewSessionManager(5, 30*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess, err := m.Acquire("user-1", "host", "host-1", "root@10.0.1.1", cancel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Context should not be cancelled yet
	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled yet")
	default:
	}

	// Cancel the session
	m.Cancel(sess.ID)

	// Context should now be cancelled
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Fatal("context should be cancelled after Cancel()")
	}
}

func TestCancelSessionStillTrackedUntilRelease(t *testing.T) {
	m := NewSessionManager(5, 30*time.Minute)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess, err := m.Acquire("user-1", "host", "host-1", "root@10.0.1.1", cancel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cancel the session
	m.Cancel(sess.ID)

	// Session should still be tracked
	if m.Count("user-1") != 1 {
		t.Fatalf("expected count 1 after Cancel, got %d", m.Count("user-1"))
	}

	sessions := m.List("user-1")
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after Cancel, got %d", len(sessions))
	}

	// Release should remove it
	m.Release(sess.ID)

	if m.Count("user-1") != 0 {
		t.Fatalf("expected count 0 after Release, got %d", m.Count("user-1"))
	}
}

func TestCancelNonExistentSession(t *testing.T) {
	m := NewSessionManager(5, 30*time.Minute)

	// Should not panic
	m.Cancel("non-existent-id")
	m.Cancel("")
}

func TestSessionIDsAreUnique(t *testing.T) {
	m := NewSessionManager(100, 30*time.Minute)

	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		sess, err := m.Acquire("user-1", "host", "host-1", "label", cancel)
		if err != nil {
			t.Fatalf("unexpected error on acquire %d: %v", i, err)
		}
		if seen[sess.ID] {
			t.Fatalf("duplicate session ID: %s", sess.ID)
		}
		seen[sess.ID] = true
	}
}

func TestIdleTimeout(t *testing.T) {
	timeout := 5 * time.Minute
	m := NewSessionManager(5, timeout)

	if m.IdleTimeout() != timeout {
		t.Fatalf("expected idle timeout %v, got %v", timeout, m.IdleTimeout())
	}
}

func TestReleaseFreesSlotForNewSession(t *testing.T) {
	maxPerUser := 1
	m := NewSessionManager(maxPerUser, 30*time.Minute)

	_, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	sess, err := m.Acquire("user-1", "host", "host-1", "label", cancel1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail — at max
	_, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	_, err = m.Acquire("user-1", "host", "host-2", "label2", cancel2)
	if err == nil {
		t.Fatal("expected error when at max, got nil")
	}

	// Release the first session
	m.Release(sess.ID)

	// Now a new acquire should succeed
	_, cancel3 := context.WithCancel(context.Background())
	defer cancel3()
	sess2, err := m.Acquire("user-1", "host", "host-3", "label3", cancel3)
	if err != nil {
		t.Fatalf("unexpected error after release: %v", err)
	}
	if sess2.ID == sess.ID {
		t.Fatal("new session should have a different ID")
	}
}

func TestCancelCalledMultipleTimes(t *testing.T) {
	m := NewSessionManager(5, 30*time.Minute)

	var cancelCount atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	// Wrap cancel to count calls; context.CancelFunc is safe to call multiple times
	wrappedCancel := func() {
		cancelCount.Add(1)
		cancel()
	}

	sess, err := m.Acquire("user-1", "host", "host-1", "label", wrappedCancel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m.Cancel(sess.ID)
	m.Cancel(sess.ID)

	// cancel was called both times
	if cancelCount.Load() != 2 {
		t.Fatalf("expected cancel called 2 times, got %d", cancelCount.Load())
	}

	// Context should be done
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Fatal("context should be cancelled")
	}
}

func TestReleaseCleanupEmptyUserEntry(t *testing.T) {
	m := NewSessionManager(5, 30*time.Minute)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess, err := m.Acquire("user-1", "host", "host-1", "label", cancel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m.Release(sess.ID)

	// After releasing the only session, List should return empty (not nil panic etc.)
	sessions := m.List("user-1")
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions after release, got %d", len(sessions))
	}
	if m.Count("user-1") != 0 {
		t.Fatalf("expected count 0 after release, got %d", m.Count("user-1"))
	}
}
