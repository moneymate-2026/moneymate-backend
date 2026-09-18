package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
)

// Mock implementations for testing
type mockUserRepo struct {
	domain.UserRepository
	users            map[uuid.UUID]*domain.User
	usersByEmail     map[string]*domain.User
	updatedPasswords map[uuid.UUID]string
	tokenVersions    map[uuid.UUID]int64
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := m.usersByEmail[email]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	m.updatedPasswords[userID] = passwordHash
	if u, ok := m.users[userID]; ok {
		u.PasswordHash = &passwordHash
	}
	return nil
}

func (m *mockUserRepo) IncrementTokenVersion(ctx context.Context, userID uuid.UUID) (int64, error) {
	m.tokenVersions[userID]++
	return m.tokenVersions[userID], nil
}

func (m *mockUserRepo) GetTokenVersion(ctx context.Context, userID uuid.UUID) (int64, error) {
	return m.tokenVersions[userID], nil
}

type mockRefreshTokenRepo struct {
	domain.RefreshTokenRepository
	revokedAllForUsers []uuid.UUID
}

func (m *mockRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	m.revokedAllForUsers = append(m.revokedAllForUsers, userID)
	return nil
}

type mockStore struct {
	domain.Store
	upgradedVersions []string
	resetOTPs        map[string]string
	resetOTPAttempts map[string]int64
	cooldownAllowed  bool
	cooldownRemain   time.Duration
}

func (m *mockStore) UpgradeTokenVersion(ctx context.Context, userID string) error {
	m.upgradedVersions = append(m.upgradedVersions, userID)
	return nil
}

func (m *mockStore) SetPasswordResetOTP(ctx context.Context, email, otpHash string, ttl time.Duration) error {
	if m.resetOTPs == nil {
		m.resetOTPs = make(map[string]string)
	}
	m.resetOTPs[email] = otpHash
	return nil
}

func (m *mockStore) GetPasswordResetOTP(ctx context.Context, email string) (string, bool, error) {
	hash, ok := m.resetOTPs[email]
	return hash, ok, nil
}

func (m *mockStore) DeletePasswordResetOTP(ctx context.Context, email string) error {
	delete(m.resetOTPs, email)
	return nil
}

func (m *mockStore) IncrementPasswordResetOTPAttempts(ctx context.Context, email string, ttl time.Duration) (int64, error) {
	if m.resetOTPAttempts == nil {
		m.resetOTPAttempts = make(map[string]int64)
	}
	m.resetOTPAttempts[email]++
	return m.resetOTPAttempts[email], nil
}

func (m *mockStore) TrySetPasswordResetResendCooldown(ctx context.Context, email string, ttl time.Duration) (bool, time.Duration, error) {
	return m.cooldownAllowed, m.cooldownRemain, nil
}

func (m *mockStore) ResetPasswordResetOTPAttempts(ctx context.Context, email string) error {
	delete(m.resetOTPAttempts, email)
	return nil
}

type mockHasher struct{}

func (m *mockHasher) Hash(password string) (string, error) {
	return "hashed_" + password, nil
}

func (m *mockHasher) Verify(hash, password string) (bool, error) {
	return hash == "hashed_"+password, nil
}

func TestAuthUsecase_ChangePassword(t *testing.T) {
	userID := uuid.New()
	currentHash := "hashed_CurrentPassword123!"

	setup := func() (*authUsecase, *mockUserRepo, *mockRefreshTokenRepo) {
		userRepo := &mockUserRepo{
			users: map[uuid.UUID]*domain.User{
				userID: {
					ID:           userID,
					Email:        "user@example.com",
					PasswordHash: &currentHash,
					Status:       domain.UserStatusActive,
				},
			},
			updatedPasswords: make(map[uuid.UUID]string),
			tokenVersions:    make(map[uuid.UUID]int64),
		}
		refreshRepo := &mockRefreshTokenRepo{}
		store := &mockStore{}
		hasher := &mockHasher{}

		uc := &authUsecase{
			userRepo:         userRepo,
			refreshTokenRepo: refreshRepo,
			store:            store,
			hasher:           hasher,
		}
		return uc, userRepo, refreshRepo
	}

	t.Run("success", func(t *testing.T) {
		uc, userRepo, refreshRepo := setup()
		err := uc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
			OldPassword:     "CurrentPassword123!",
			NewPassword:     "NewValidPassword456!",
			ConfirmPassword: "NewValidPassword456!",
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if userRepo.updatedPasswords[userID] != "hashed_NewValidPassword456!" {
			t.Errorf("expected password to be updated to hashed new password, got %s", userRepo.updatedPasswords[userID])
		}
		if len(refreshRepo.revokedAllForUsers) != 1 || refreshRepo.revokedAllForUsers[0] != userID {
			t.Errorf("expected refresh tokens revoked for user %s", userID)
		}
	})

	t.Run("mismatch confirm password", func(t *testing.T) {
		uc, _, _ := setup()
		err := uc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
			OldPassword:     "CurrentPassword123!",
			NewPassword:     "NewValidPassword456!",
			ConfirmPassword: "DifferentPassword456!",
		})
		if !errors.Is(err, apperrors.ErrPasswordMismatch) {
			t.Fatalf("expected ErrPasswordMismatch, got %v", err)
		}
	})

	t.Run("same new and old password", func(t *testing.T) {
		uc, _, _ := setup()
		err := uc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
			OldPassword:     "CurrentPassword123!",
			NewPassword:     "CurrentPassword123!",
			ConfirmPassword: "CurrentPassword123!",
		})
		if !errors.Is(err, apperrors.ErrSamePassword) {
			t.Fatalf("expected ErrSamePassword, got %v", err)
		}
	})

	t.Run("wrong old password", func(t *testing.T) {
		uc, _, _ := setup()
		err := uc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
			OldPassword:     "WrongPassword123!",
			NewPassword:     "NewValidPassword456!",
			ConfirmPassword: "NewValidPassword456!",
		})
		if !errors.Is(err, apperrors.ErrIncorrectPassword) {
			t.Fatalf("expected ErrIncorrectPassword, got %v", err)
		}
	})

	t.Run("invalid password complexity (too short)", func(t *testing.T) {
		uc, _, _ := setup()
		err := uc.ChangePassword(context.Background(), userID, ChangePasswordRequest{
			OldPassword:     "CurrentPassword123!",
			NewPassword:     "short",
			ConfirmPassword: "short",
		})
		if !errors.Is(err, apperrors.ErrInvalidPassword) {
			t.Fatalf("expected ErrInvalidPassword, got %v", err)
		}
	})
}

func TestAuthUsecase_ResetPassword(t *testing.T) {
	userID := uuid.New()
	email := "user@example.com"
	rawOTP := "123456"
	hashedCode := hashOTP(rawOTP)

	setup := func() (*authUsecase, *mockUserRepo, *mockStore, *mockRefreshTokenRepo) {
		userRepo := &mockUserRepo{
			users: map[uuid.UUID]*domain.User{
				userID: {
					ID:     userID,
					Email:  email,
					Status: domain.UserStatusActive,
				},
			},
			usersByEmail: map[string]*domain.User{
				email: {
					ID:     userID,
					Email:  email,
					Status: domain.UserStatusActive,
				},
			},
			updatedPasswords: make(map[uuid.UUID]string),
			tokenVersions:    make(map[uuid.UUID]int64),
		}
		store := &mockStore{
			resetOTPs: map[string]string{
				email: hashedCode,
			},
			resetOTPAttempts: make(map[string]int64),
		}
		refreshRepo := &mockRefreshTokenRepo{}
		hasher := &mockHasher{}

		uc := &authUsecase{
			userRepo:         userRepo,
			refreshTokenRepo: refreshRepo,
			store:            store,
			hasher:           hasher,
		}
		return uc, userRepo, store, refreshRepo
	}

	t.Run("success", func(t *testing.T) {
		uc, userRepo, store, refreshRepo := setup()
		err := uc.ResetPassword(context.Background(), ResetPasswordRequest{
			Email:           email,
			Code:            rawOTP,
			NewPassword:     "NewResetPassword123!",
			ConfirmPassword: "NewResetPassword123!",
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if userRepo.updatedPasswords[userID] != "hashed_NewResetPassword123!" {
			t.Errorf("expected password updated, got %s", userRepo.updatedPasswords[userID])
		}
		if userRepo.tokenVersions[userID] != 1 {
			t.Errorf("expected token version incremented to 1, got %d", userRepo.tokenVersions[userID])
		}
		if len(store.upgradedVersions) != 1 || store.upgradedVersions[0] != userID.String() {
			t.Errorf("expected redis token version upgraded for %s", userID)
		}
		if len(refreshRepo.revokedAllForUsers) != 1 || refreshRepo.revokedAllForUsers[0] != userID {
			t.Errorf("expected all refresh tokens revoked")
		}
		if _, exists := store.resetOTPs[email]; exists {
			t.Errorf("expected OTP to be deleted after reset")
		}
	})

	t.Run("wrong OTP code", func(t *testing.T) {
		uc, _, _, _ := setup()
		err := uc.ResetPassword(context.Background(), ResetPasswordRequest{
			Email:           email,
			Code:            "999999",
			NewPassword:     "NewResetPassword123!",
			ConfirmPassword: "NewResetPassword123!",
		})
		if err == nil {
			t.Fatalf("expected error for wrong OTP, got nil")
		}
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != "OTP_INVALID" {
			t.Fatalf("expected OTP_INVALID error, got %v", err)
		}
	})

	t.Run("expired OTP", func(t *testing.T) {
		uc, _, store, _ := setup()
		delete(store.resetOTPs, email)

		err := uc.ResetPassword(context.Background(), ResetPasswordRequest{
			Email:           email,
			Code:            rawOTP,
			NewPassword:     "NewResetPassword123!",
			ConfirmPassword: "NewResetPassword123!",
		})
		if !errors.Is(err, apperrors.ErrOTPExpired) {
			t.Fatalf("expected ErrOTPExpired, got %v", err)
		}
	})

	t.Run("password mismatch", func(t *testing.T) {
		uc, _, _, _ := setup()
		err := uc.ResetPassword(context.Background(), ResetPasswordRequest{
			Email:           email,
			Code:            rawOTP,
			NewPassword:     "NewResetPassword123!",
			ConfirmPassword: "DifferentResetPassword123!",
		})
		if !errors.Is(err, apperrors.ErrPasswordMismatch) {
			t.Fatalf("expected ErrPasswordMismatch, got %v", err)
		}
	})
}
