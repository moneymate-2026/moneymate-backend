package domain

import (
	"context"
	"time"
)

type Store interface {
	UpgradeTokenVersion(ctx context.Context, userID string) error
	GetTokenVersion(ctx context.Context, userID string) (int64, error)
	ClaimRefreshToken(ctx context.Context, tokenID string, ttl time.Duration) (bool, error)

	SetRegistrationOTP(ctx context.Context, email, otpHash string, ttl time.Duration) error
	GetRegistrationOTP(ctx context.Context, email string) (otpHash string, found bool, err error)
	DeleteRegistrationOTP(ctx context.Context, email string) error
	IncrementOTPAttempts(ctx context.Context, email string, ttl time.Duration) (int64, error)
	TrySetResendCooldown(ctx context.Context, email string, ttl time.Duration) (bool, time.Duration, error)
	ResetOTPAttempts(ctx context.Context, email string) error
	
	MarkEmailVerified(ctx context.Context, email string, ttl time.Duration) error
	ConsumeEmailVerified(ctx context.Context, email string) (verified bool, err error)

	SetPasswordResetOTP(ctx context.Context, email, otpHash string, ttl time.Duration) error
	GetPasswordResetOTP(ctx context.Context, email string) (otpHash string, found bool, err error)
	DeletePasswordResetOTP(ctx context.Context, email string) error
	IncrementPasswordResetOTPAttempts(ctx context.Context, email string, ttl time.Duration) (int64, error)
	TrySetPasswordResetResendCooldown(ctx context.Context, email string, ttl time.Duration) (bool, time.Duration, error)
	ResetPasswordResetOTPAttempts(ctx context.Context, email string) error
}
