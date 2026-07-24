package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
)

// ── Register ──────────────────────────────────────────────────────

type RegisterRequest struct {
	Email       string
	Phone       string
	FullName    string
	Password    string
	AccountType domain.AccountType 
}
type RegisterResponse struct {
	UserID uuid.UUID
	Email  string
	Handle string
	Status string
}

// ── Login ─────────────────────────────────────────────────────────

type LoginRequest struct {
	Identifier string
	Password   string
	DeviceID   string
	UserAgent  string
	IPAddress  string
}

type LoginResponse struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	User             UserSummary
}

type UserSummary struct {
	ID              uuid.UUID
	Email           string
	Handle          string
	FullName        string
	Status          string
	IsEmailVerified bool
}

// ── Logout ────────────────────────────────────────────────────────

type LogoutRequest struct {
	UserID       uuid.UUID
	RefreshToken string
	AllDevices   bool
}

// ── Registration OTP ────────────────────────────────────────────

type SendRegistrationOTPRequest struct {
	Email string
}
type SendRegistrationOTPResponse struct {
    Email             string `json:"email"`
    ExpiresIn         int    `json:"expires_in"`         
    ResendCooldownIn  int    `json:"resend_cooldown_in"`  
    MaxVerifyAttempts int    `json:"max_verify_attempts"` 
}
type VerifyRegistrationOTPRequest struct {
	Email string
	Code  string
}

type VerifyRegistrationOTPResponse struct {
	Email    string
	Verified bool
}

type RefreshTokenRequest struct {
    RefreshToken string
}

type RefreshTokenResponse struct {
    AccessToken     string    `json:"access_token"`
    RefreshToken    string    `json:"refresh_token"`
    AccessExpiresAt time.Time `json:"access_expires_at"`
}