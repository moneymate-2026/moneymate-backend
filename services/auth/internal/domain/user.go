package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusPending   UserStatus = "pending"
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
)

type User struct {
	ID              uuid.UUID
	Email           string
	Phone           *string 
	FullName        string
	Handle          string
	PasswordHash    *string 
	Status          UserStatus
	TokenVersion    int64
	IsEmailVerified bool
	IsPhoneVerified bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ListUsersFilter struct {
    Status   string   
    Search   string   
    SortBy   string   
    SortDesc bool     
}

type Pagination struct {
    Page     int 
    PageSize int 
}

type ListUsersResult struct {
    Users      []User
    TotalCount int64
}

type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id uuid.UUID) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    GetByHandle(ctx context.Context, handle string) (*User, error)
    GetByPhone(ctx context.Context, phone string) (*User, error)

    EmailExists(ctx context.Context, email string) (bool, error)
    HandleExists(ctx context.Context, handle string) (bool, error)
    PhoneExists(ctx context.Context, phone string) (bool, error)

    CheckUniqueFields(ctx context.Context, email, handle, phone string) error

    UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
    UpdateStatus(ctx context.Context, userID uuid.UUID, status UserStatus) error
    VerifyEmail(ctx context.Context, userID uuid.UUID) error
    VerifyPhone(ctx context.Context, userID uuid.UUID) error
    IncrementTokenVersion(ctx context.Context, userID uuid.UUID) (int64, error)
    GetTokenVersion(ctx context.Context, userID uuid.UUID) (int64, error)
    SoftDelete(ctx context.Context, userID uuid.UUID) error

    ListUsers(ctx context.Context, filter ListUsersFilter, page Pagination) (*ListUsersResult, error)
}