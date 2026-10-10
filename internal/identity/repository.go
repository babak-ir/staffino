package identity

import (
	"context"
	"time"
)

type UserRepository interface {
	Create(ctx context.Context, phone string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByPhone(ctx context.Context, phone string) (*User, error)
}

type OTPRepository interface {
	Create(ctx context.Context, phone, codeHash string, expiresAt time.Time) (*OTPChallenge, error)
	LatestActiveByPhone(ctx context.Context, phone string, now time.Time) (*OTPChallenge, error)
	MarkConsumed(ctx context.Context, id string, at time.Time) error
}