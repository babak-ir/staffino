package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	identityv1connect "staffino/gen/staffino/identity/v1/identityv1connect"
	"staffino/internal/identity"
	"staffino/internal/identity/postgres"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	pepper := os.Getenv("OTP_PEPPER")
	if dbURL == "" || pepper == "" {
		slog.Error("DATABASE_URL and OTP_PEPPER are required")
		os.Exit(1)
	}
	addr := os.Getenv("APP_PORT")
	if addr == "" {
		addr = ":8080"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	svc, err := identity.NewService(
		postgres.NewUsers(pool),
		postgres.NewChallenges(pool),
		devIssuer{},
		logNotifier{},
		[]byte(pepper),
	)
	if err != nil {
		slog.Error("service", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	path, handler := identityv1connect.NewIdentityServiceHandler(identity.NewHandler(svc))
	mux.Handle(path, handler)

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		slog.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

type logNotifier struct{}

func (logNotifier) SendOTP(_ context.Context, phone, code string) error {
	slog.Info("otp", "phone", phone, "code", code)
	return nil
}

type devIssuer struct{}

func (devIssuer) Issue(_ context.Context, user *identity.User, now time.Time) (string, time.Time, error) {
	return "dev-" + user.ID, now.Add(15 * time.Minute), nil
}