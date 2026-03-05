package oidc

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrCodeNotFound    = errors.New("authorization code not found")
	ErrCodeExpired     = errors.New("authorization code expired")
	ErrCodeAlreadyUsed = errors.New("authorization code already used")
)

// AuthCodeStore manages authorization codes.
type AuthCodeStore interface {
	Store(code *AuthorizationCode)
	Consume(codeStr string) (*AuthorizationCode, error)
}

// memAuthCodeStore is an in-memory implementation with TTL.
type memAuthCodeStore struct {
	mu    sync.Mutex
	codes map[string]*AuthorizationCode
}

// NewMemAuthCodeStore creates a new in-memory auth code store.
func NewMemAuthCodeStore() AuthCodeStore {
	return &memAuthCodeStore{
		codes: make(map[string]*AuthorizationCode),
	}
}

func (s *memAuthCodeStore) Store(code *AuthorizationCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Lazy cleanup of expired codes
	now := time.Now()
	for k, v := range s.codes {
		if now.After(v.ExpiresAt) {
			delete(s.codes, k)
		}
	}
	s.codes[code.Code] = code
}

func (s *memAuthCodeStore) Consume(codeStr string) (*AuthorizationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	code, ok := s.codes[codeStr]
	if !ok {
		return nil, ErrCodeNotFound
	}

	if time.Now().After(code.ExpiresAt) {
		delete(s.codes, codeStr)
		return nil, ErrCodeExpired
	}

	if code.Consumed {
		delete(s.codes, codeStr)
		return nil, ErrCodeAlreadyUsed
	}

	code.Consumed = true
	delete(s.codes, codeStr)
	return code, nil
}

// GenerateCode generates a cryptographically random authorization code.
func GenerateCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
