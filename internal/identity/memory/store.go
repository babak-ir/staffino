package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"staffino/internal/identity"
)

type Store struct {
	mu         sync.Mutex
	users      map[string]*identity.User
	byPhone    map[string]string
	challenges []*identity.OTPChallenge
	codes      map[string]string
	nextID     int
}

func New() *Store {
	return &Store{
		users:   map[string]*identity.User{},
		byPhone: map[string]string{},
		codes:   map[string]string{},
	}
}

type Users struct{ s *Store }

type Challenges struct{ s *Store }

func (s *Store) Users() *Users { return &Users{s} }

func (s *Store) Challenges() *Challenges { return &Challenges{s} }

func (u *Users) Create(_ context.Context, phone string) (*identity.User, error) {
	u.s.mu.Lock()
	defer u.s.mu.Unlock()
	if _, ok := u.s.byPhone[phone]; ok {
		return nil, identity.ErrConflict
	}
	u.s.nextID++
	user := &identity.User{
		ID:        fmt.Sprintf("user-%d", u.s.nextID),
		Phone:     phone,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	u.s.users[user.ID] = user
	u.s.byPhone[phone] = user.ID
	copy := *user
	return &copy, nil
}

func (u *Users) GetByID(_ context.Context, id string) (*identity.User, error) {
	u.s.mu.Lock()
	defer u.s.mu.Unlock()
	user, ok := u.s.users[id]
	if !ok {
		return nil, identity.ErrNotFound
	}
	copy := *user
	return &copy, nil
}

func (u *Users) GetByPhone(_ context.Context, phone string) (*identity.User, error) {
	u.s.mu.Lock()
	defer u.s.mu.Unlock()
	id, ok := u.s.byPhone[phone]
	if !ok {
		return nil, identity.ErrNotFound
	}
	copy := *u.s.users[id]
	return &copy, nil
}

func (c *Challenges) Create(_ context.Context, phone, codeHash string, expiresAt time.Time) (*identity.OTPChallenge, error) {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	c.s.nextID++
	challenge := &identity.OTPChallenge{
		ID:        fmt.Sprintf("otp-%d", c.s.nextID),
		Phone:     phone,
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	c.s.challenges = append(c.s.challenges, challenge)
	copy := *challenge
	return &copy, nil
}

func (c *Challenges) LatestActiveByPhone(_ context.Context, phone string, now time.Time) (*identity.OTPChallenge, error) {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	for i := len(c.s.challenges) - 1; i >= 0; i-- {
		challenge := c.s.challenges[i]
		if challenge.Phone != phone || challenge.ConsumedAt != nil || !challenge.ExpiresAt.After(now) {
			continue
		}
		copy := *challenge
		return &copy, nil
	}
	return nil, identity.ErrNotFound
}

func (c *Challenges) MarkConsumed(_ context.Context, id string, at time.Time) error {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	for _, challenge := range c.s.challenges {
		if challenge.ID != id {
			continue
		}
		if challenge.ConsumedAt != nil {
			return identity.ErrConflict
		}
		challenge.ConsumedAt = &at
		return nil
	}
	return identity.ErrNotFound
}

func (s *Store) SendOTP(_ context.Context, phone, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[phone] = code
	return nil
}

func (s *Store) LastCode(phone string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.codes[phone]
}

func (s *Store) Issue(_ context.Context, user *identity.User, now time.Time) (string, time.Time, error) {
	return "token-" + user.ID, now.Add(15 * time.Minute), nil
}
