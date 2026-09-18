package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/auth/config"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
)

type mockMailer struct {
	sentOTPs          []string
	sentPasswordOTPs  []string
}

func (m *mockMailer) SendOTP(ctx context.Context, toEmail, otp string) error {
	m.sentOTPs = append(m.sentOTPs, toEmail+":"+otp)
	return nil
}

func (m *mockMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string) error {
	m.sentPasswordOTPs = append(m.sentPasswordOTPs, toEmail+":"+otp)
	return nil
}

func TestOTPUsecase_ForgotPassword(t *testing.T) {
	realEmail := "realuser@example.com"
	fakeEmail := "fakeuser@example.com"
	userID := uuid.New()

	otpCfg := config.OTPConfig{
		Length:            6,
		TTL:               15 * time.Minute,
		ResendCooldown:    60 * time.Second,
		MaxVerifyAttempts: 5,
		EmailVerifiedTTL:  30 * time.Minute,
	}

	setup := func() (*otpUsecase, *mockUserRepo, *mockStore, *mockMailer) {
		userRepo := &mockUserRepo{
			usersByEmail: map[string]*domain.User{
				realEmail: {
					ID:    userID,
					Email: realEmail,
				},
			},
		}
		store := &mockStore{
			cooldownAllowed: true,
			resetOTPs:       make(map[string]string),
		}
		mailer := &mockMailer{}
		uc := &otpUsecase{
			userRepo: userRepo,
			store:    store,
			mailer:   mailer,
			cfg:      otpCfg,
		}
		return uc, userRepo, store, mailer
	}

	t.Run("real user sends email and sets OTP", func(t *testing.T) {
		uc, _, store, mailer := setup()
		resp, err := uc.ForgotPassword(context.Background(), ForgotPasswordRequest{
			Email: realEmail,
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if resp.Email != realEmail {
			t.Errorf("expected email in response %s, got %s", realEmail, resp.Email)
		}
		if len(mailer.sentPasswordOTPs) != 1 {
			t.Fatalf("expected 1 password reset email sent, got %d", len(mailer.sentPasswordOTPs))
		}
		if _, exists := store.resetOTPs[realEmail]; !exists {
			t.Errorf("expected OTP to be stored in Redis store")
		}
	})

	t.Run("fake user returns standard response without email or OTP (anti-enumeration)", func(t *testing.T) {
		uc, _, store, mailer := setup()
		resp, err := uc.ForgotPassword(context.Background(), ForgotPasswordRequest{
			Email: fakeEmail,
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if resp.Email != fakeEmail {
			t.Errorf("expected email in response %s, got %s", fakeEmail, resp.Email)
		}
		if len(mailer.sentPasswordOTPs) != 0 {
			t.Errorf("expected 0 password reset emails sent for non-existent user, got %d", len(mailer.sentPasswordOTPs))
		}
		if _, exists := store.resetOTPs[fakeEmail]; exists {
			t.Errorf("expected no OTP stored in Redis for non-existent user")
		}
	})

	t.Run("cooldown active returns OTP_COOLDOWN", func(t *testing.T) {
		uc, _, store, mailer := setup()
		store.cooldownAllowed = false
		store.cooldownRemain = 45 * time.Second

		_, err := uc.ForgotPassword(context.Background(), ForgotPasswordRequest{
			Email: realEmail,
		})
		if err == nil {
			t.Fatalf("expected cooldown error, got nil")
		}
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != "OTP_COOLDOWN" {
			t.Fatalf("expected OTP_COOLDOWN error code, got %v", err)
		}
		if len(mailer.sentPasswordOTPs) != 0 {
			t.Errorf("expected no email sent when on cooldown")
		}
	})
}
