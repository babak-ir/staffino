package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const otpTTL = 2 * time.Minute

type VerifyResult struct {
	User                   User
	AccessToken            string
	AccessTokenExpiresAt   time.Time
}

type TokenIssuer interface {
	Issue(ctx context.Context, user *User, now time.Time) (token string, expiresAt time.Time, err error)
}

type OTPNotifier interface {
	SendOTP(ctx context.Context, phone, code string) error
}

type Service struct {
	users   UserRepository
	otps    OTPRepository
	tokens  TokenIssuer
	notify  OTPNotifier
	pepper  []byte
	now     func() time.Time
	newCode func() (string, error)
}

func NewService(users UserRepository, otps OTPRepository, tokens TokenIssuer, notify OTPNotifier, pepper []byte) (*Service, error) {
	if len(pepper) == 0 {
		return nil, errors.New("otp pepper is required")
	}
	return &Service{
		users:   users,
		otps:    otps,
		tokens:  tokens,
		notify:  notify,
		pepper:  pepper,
		now:     time.Now,
		newCode: randomCode,
	}, nil
}

func (s *Service) RequestOTP(ctx context.Context, phone string) (int32, error) {
	if err := validatePhone(phone); err != nil {
		return 0, err
	}
	code, err := s.newCode()
	if err != nil {
		return 0, err
	}
	expiresAt := s.now().Add(otpTTL)
	if _, err := s.otps.Create(ctx, phone, s.hash(code), expiresAt); err != nil {
		return 0, err
	}
	if err := s.notify.SendOTP(ctx, phone, code); err != nil {
		return 0, err
	}
	return int32(otpTTL.Seconds()), nil
}

func (s *Service) VerifyOTP(ctx context.Context, phone, otp string) (*VerifyResult, error) {
	if err := validatePhone(phone); err != nil {
		return nil, err
	}
	if len(otp) != 6 {
		return nil, ErrValidation
	}
	now := s.now()
	challenge, err := s.otps.LatestActiveByPhone(ctx, phone, now)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	if !hmac.Equal([]byte(challenge.CodeHash), []byte(s.hash(otp))) {
		return nil, ErrUnauthorized
	}
	if err := s.otps.MarkConsumed(ctx, challenge.ID, now); err != nil {
		return nil, err
	}
	user, err := s.users.GetByPhone(ctx, phone)
	if errors.Is(err, ErrNotFound) {
		user, err = s.users.Create(ctx, phone)
	}
	if err != nil {
		return nil, err
	}
	token, expiresAt, err := s.tokens.Issue(ctx, user, now)
	if err != nil {
		return nil, err
	}
	return &VerifyResult{
		User:                 *user,
		AccessToken:          token,
		AccessTokenExpiresAt: expiresAt,
	}, nil
}

func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
	if id == "" {
		return nil, ErrValidation
	}
	return s.users.GetByID(ctx, id)
}

func (s *Service) hash(code string) string {
	mac := hmac.New(sha256.New, s.pepper)
	_, _ = mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func validatePhone(phone string) error {
	if len(phone) < 8 || len(phone) > 16 || phone[0] != '+' {
		return ErrValidation
	}
	return nil
}

func randomCode() (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint32(buf[:]) % 1000000
	return fmt.Sprintf("%06d", n), nil
}