package identity

import "time"

type User struct {
	ID        string
	Phone     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OTPChallenge struct {
	ID         string
	Phone      string
	CodeHash   string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}