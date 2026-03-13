package websocket

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// sessionCounter is an atomic counter used to generate unique session IDs.
var sessionCounter atomic.Int64

// Session represents an active WebSocket session.
type Session struct {
	ID         string
	UserID     string
	Resource   string // "host", "container", "database" ...
	ResourceID string
	Label      string // display label, e.g. "root@10.0.1.5"
	CreatedAt  time.Time
	cancel     context.CancelFunc // unexported
}

// SessionManager tracks active WebSocket sessions and enforces per-user
// concurrency limits. It is safe for concurrent use.
type SessionManager struct {
	mu          sync.Mutex
	sessions    map[string][]*Session // key: userID
	byID        map[string]*Session   // key: sessionID
	maxPerUser  int
	idleTimeout time.Duration
}

// NewSessionManager creates a SessionManager that allows at most maxPerUser
// concurrent sessions per user. The idleTimeout is stored for callers to
// query but is not enforced by the manager itself.
func NewSessionManager(maxPerUser int, idleTimeout time.Duration) *SessionManager {
	return &SessionManager{
		sessions:    make(map[string][]*Session),
		byID:        make(map[string]*Session),
		maxPerUser:  maxPerUser,
		idleTimeout: idleTimeout,
	}
}

// IdleTimeout returns the configured idle timeout duration.
func (m *SessionManager) IdleTimeout() time.Duration {
	return m.idleTimeout
}

// Acquire creates a new session for the given user. It returns an error if the
// user already has maxPerUser active sessions.
func (m *SessionManager) Acquire(userID, resource, resourceID, label string, cancel context.CancelFunc) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions[userID]) >= m.maxPerUser {
		return nil, fmt.Errorf("user %s has reached the maximum number of concurrent sessions (%d)", userID, m.maxPerUser)
	}

	id := fmt.Sprintf("sess-%d", sessionCounter.Add(1))
	sess := &Session{
		ID:         id,
		UserID:     userID,
		Resource:   resource,
		ResourceID: resourceID,
		Label:      label,
		CreatedAt:  time.Now(),
		cancel:     cancel,
	}

	m.sessions[userID] = append(m.sessions[userID], sess)
	m.byID[id] = sess

	return sess, nil
}

// Release removes a session from tracking. If the session ID does not exist,
// this is a no-op.
func (m *SessionManager) Release(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.byID[sessionID]
	if !ok {
		return
	}

	delete(m.byID, sessionID)

	userSessions := m.sessions[sess.UserID]
	for i, s := range userSessions {
		if s.ID == sessionID {
			m.sessions[sess.UserID] = append(userSessions[:i], userSessions[i+1:]...)
			break
		}
	}

	// Clean up empty user entries
	if len(m.sessions[sess.UserID]) == 0 {
		delete(m.sessions, sess.UserID)
	}
}

// Cancel calls the cancel function associated with the session for forced
// disconnect. The session remains tracked until Release is called.
// If the session ID does not exist, this is a no-op.
func (m *SessionManager) Cancel(sessionID string) {
	m.mu.Lock()
	sess, ok := m.byID[sessionID]
	m.mu.Unlock()

	if !ok {
		return
	}
	sess.cancel()
}

// Count returns the number of active sessions for the given user.
func (m *SessionManager) Count(userID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.sessions[userID])
}

// List returns copies of all sessions for the given user. The cancel function
// is nil in the returned copies to avoid exposing internal state.
func (m *SessionManager) List(userID string) []Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	userSessions := m.sessions[userID]
	if len(userSessions) == 0 {
		return nil
	}

	result := make([]Session, len(userSessions))
	for i, s := range userSessions {
		result[i] = Session{
			ID:         s.ID,
			UserID:     s.UserID,
			Resource:   s.Resource,
			ResourceID: s.ResourceID,
			Label:      s.Label,
			CreatedAt:  s.CreatedAt,
			// cancel intentionally left nil
		}
	}
	return result
}
