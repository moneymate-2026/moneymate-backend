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
	PIN         string
	AccountType domain.AccountType
}
type RegisterResponse struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	User             UserSummary
}
// ── Login ─────────────────────────────────────────────────────────

type LoginRequest struct {
	Identifier string
	Password   string
	PIN        string
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
	QRCode string
}
type AdminLoginRequest struct {
	Email    string
	Password string
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

// ── Password Management ────────────────────────────────────────

type ChangePasswordRequest struct {
	OldPassword     string
	NewPassword     string
	ConfirmPassword string
}

type ForgotPasswordRequest struct {
	Email string
}

type ForgotPasswordResponse struct {
	Email             string `json:"email"`
	ExpiresIn         int    `json:"expires_in"`
	ResendCooldownIn  int    `json:"resend_cooldown_in"`
	MaxVerifyAttempts int    `json:"max_verify_attempts"`
}

type ResetPasswordRequest struct {
	Email           string
	Code            string
	NewPassword     string
	ConfirmPassword string
}


//admin
type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone"`
	FullName string `json:"full_name" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
	Role     string `json:"role" validate:"required"`
	PIN      string `json:"pin" validate:"required,len=6,numeric"`
}

type AdminUserSummary struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	Phone           string `json:"phone,omitempty"`
	FullName        string `json:"full_name"`
	Handle          string `json:"handle"`
	Status          string `json:"status"`
	Role            string `json:"role"`
	IsEmailVerified bool   `json:"is_email_verified"`
	IsPhoneVerified bool   `json:"is_phone_verified"`
	CreatedAt       string `json:"created_at"`
	QRCode            string `json:"qr_code"`
	ProfilePictureURL string `json:"profile_picture_url,omitempty"`	
}

type UserDetail struct {
	AdminUserSummary
	Roles []string `json:"roles"`
}


type ListUsersRequest struct {
	Status   string
	Search   string
	SortBy   string
	SortDesc bool
	Page     int
	PageSize int
}

type ListUsersResponse struct {
	Users      []AdminUserSummary `json:"users"`
	TotalCount int64         `json:"total_count"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
}

type UpdateUserRequest struct {
	FullName *string `json:"full_name,omitempty"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Password *string `json:"password,omitempty"`
}

// ── Role Management ──────────────────────────────────────────────

type CreateRoleRequest struct {
	Name        string
	Description *string
}

type UpdateRoleRequest struct {
	Name        *string
	Description *string
}

type RoleSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type AssignRoleRequest struct {
	UserID    uuid.UUID
	RoleID    uuid.UUID
	GrantedBy *uuid.UUID 
}
type RefreshTokenResponse struct {
    AccessToken     string    `json:"access_token"`
    RefreshToken    string    `json:"refresh_token"`
    AccessExpiresAt time.Time `json:"access_expires_at"`
}

// ── PIN Management ────────────────────────────────────────────────

type SetPINRequest struct {
	PIN string
}

type UpdatePINRequest struct {
	OldPIN string
	NewPIN string
}

type VerifyPINRequest struct {
	PIN string
}

type UserRegisteredEvent struct {
	UserID uuid.UUID `json:"user_id"`
	Handle string    `json:"handle"`
}

type PhoneLookupResult struct {
	UserID            string `json:"user_id,omitempty"`
	Handle            string `json:"handle"`
	Phone             string `json:"phone"`
	FullName          string `json:"full_name"`
	Email             string `json:"email,omitempty"`
	ProfilePictureURL string `json:"profile_picture_url,omitempty"`
	QRCode            string `json:"qr_code,omitempty"`
}