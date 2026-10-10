package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"staffino/internal/identity"
	"staffino/internal/identity/memory"
)

func newService(t *testing.T, now time.Time) (*identity.Service, *memory.Store, *time.Time) {
	t.Helper()
	store := memory.New()
	svc, err := identity.NewService(store.Users(), store.Challenges(), store, store, []byte("pepper"))
	if err != nil {
		t.Fatal(err)
	}
	current := now
	svc.SetNow(func() time.Time { return current })
	svc.SetNewCode(func() (string, error) { return "123456", nil })
	return svc, store, &current
}

func TestVerifyOTP(t *testing.T) {
	ctx := context.Background()
	phone := "+989121234567"
	start := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	svc, store, clock := newService(t, start)

	if _, err := svc.RequestOTP(ctx, phone); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.VerifyOTP(ctx, phone, "000000"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("wrong code: got %v", err)
	}

	*clock = start.Add(3 * time.Minute)
	if _, err := svc.VerifyOTP(ctx, phone, "123456"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("expired code: got %v", err)
	}

	if _, err := svc.RequestOTP(ctx, phone); err != nil {
		t.Fatal(err)
	}
	first, err := svc.VerifyOTP(ctx, phone, store.LastCode(phone))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.VerifyOTP(ctx, phone, "123456"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("second use: got %v", err)
	}

	if _, err := svc.RequestOTP(ctx, phone); err != nil {
		t.Fatal(err)
	}
	second, err := svc.VerifyOTP(ctx, phone, "123456")
	if err != nil {
		t.Fatal(err)
	}
	if second.User.ID != first.User.ID {
		t.Fatalf("user changed: %s then %s", first.User.ID, second.User.ID)
	}
	got, err := svc.GetUser(ctx, first.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phone != phone {
		t.Fatalf("phone: got %s", got.Phone)
	}
}
