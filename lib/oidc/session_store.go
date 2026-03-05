package oidc

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// Session represents an authenticated user session.
type Session struct {
	SessionID string
	UserID    int64
	AuthTime  time.Time
	ExpiresAt time.Time
}

// SessionStore manages user sessions.
type SessionStore interface {
	Create(session *Session)
	Get(sessionID string) (*Session, error)
	Delete(sessionID string)
}

// memSessionStore is an in-memory session store.
type memSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

// NewMemSessionStore creates a new in-memory session store.
func NewMemSessionStore() SessionStore {
	return &memSessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *memSessionStore) Create(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Lazy cleanup
	now := time.Now()
	for k, v := range s.sessions {
		if now.After(v.ExpiresAt) {
			delete(s.sessions, k)
		}
	}
	s.sessions[session.SessionID] = session
}

func (s *memSessionStore) Get(sessionID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, sessionID)
		return nil, errors.New("session expired")
	}
	return session, nil
}

func (s *memSessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// GenerateSessionID generates a cryptographically random session ID.
func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
