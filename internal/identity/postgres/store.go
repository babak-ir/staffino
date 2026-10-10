package postgres

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"staffino/internal/identity"
)

type Users struct {
	q *Queries
}

type Challenges struct {
	q *Queries
}

func NewUsers(db DBTX) *Users {
	return &Users{q: New(db)}
}

func NewChallenges(db DBTX) *Challenges {
	return &Challenges{q: New(db)}
}

func (u *Users) Create(ctx context.Context, phone string) (*identity.User, error) {
	row, err := u.q.CreateUser(ctx, phone)
	if err != nil {
		return nil, mapErr(err)
	}
	return toIdentityUser(row)
}

func (u *Users) GetByID(ctx context.Context, id string) (*identity.User, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return nil, identity.ErrNotFound
	}
	row, err := u.q.GetUserByID(ctx, uid)
	if err != nil {
		return nil, mapErr(err)
	}
	return toIdentityUser(row)
}

func (u *Users) GetByPhone(ctx context.Context, phone string) (*identity.User, error) {
	row, err := u.q.GetUserByPhone(ctx, phone)
	if err != nil {
		return nil, mapErr(err)
	}
	return toIdentityUser(row)
}

func (c *Challenges) Create(ctx context.Context, phone, codeHash string, expiresAt time.Time) (*identity.OTPChallenge, error) {
	row, err := c.q.CreateOTPChallenge(ctx, CreateOTPChallengeParams{
		Phone:     phone,
		CodeHash:  codeHash,
		ExpiresAt: timestamptz(expiresAt),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return toIdentityChallenge(row)
}

func (c *Challenges) LatestActiveByPhone(ctx context.Context, phone string, now time.Time) (*identity.OTPChallenge, error) {
	row, err := c.q.LatestActiveOTPByPhone(ctx, LatestActiveOTPByPhoneParams{
		Phone:     phone,
		ExpiresAt: timestamptz(now),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return toIdentityChallenge(row)
}

func (c *Challenges) MarkConsumed(ctx context.Context, id string, at time.Time) error {
	uid, err := parseUUID(id)
	if err != nil {
		return identity.ErrNotFound
	}
	rows, err := c.q.MarkOTPConsumed(ctx, MarkOTPConsumedParams{
		ID:         uid,
		ConsumedAt: timestamptz(at),
	})
	if err != nil {
		return mapErr(err)
	}
	if rows == 0 {
		return identity.ErrNotFound
	}
	return nil
}

func toIdentityUser(row User) (*identity.User, error) {
	id, err := uuidString(row.ID)
	if err != nil {
		return nil, err
	}
	return &identity.User{
		ID:        id,
		Phone:     row.Phone,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func toIdentityChallenge(row OtpChallenge) (*identity.OTPChallenge, error) {
	id, err := uuidString(row.ID)
	if err != nil {
		return nil, err
	}
	challenge := &identity.OTPChallenge{
		ID:        id,
		Phone:     row.Phone,
		CodeHash:  row.CodeHash,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.ConsumedAt.Valid {
		consumed := row.ConsumedAt.Time
		challenge.ConsumedAt = &consumed
	}
	return challenge, nil
}

func parseUUID(id string) (pgtype.UUID, error) {
	var uid pgtype.UUID
	err := uid.Scan(id)
	return uid, err
}

func uuidString(id pgtype.UUID) (string, error) {
	if !id.Valid {
		return "", identity.ErrNotFound
	}
	var buf [36]byte
	hex.Encode(buf[0:8], id.Bytes[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], id.Bytes[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], id.Bytes[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], id.Bytes[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], id.Bytes[10:16])
	return string(buf[:]), nil
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return identity.ErrConflict
	}
	return err
}