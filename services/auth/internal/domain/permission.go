package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID          uuid.UUID
	Name        string
	Description *string
	CreatedAt   time.Time
}

// PermissionRepository handles pure Permission CRUD and Role-Permission relationships.
type PermissionRepository interface {
	// ── Permission CRUD ───────────────────────────────────────────
	Create(ctx context.Context, permission *Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*Permission, error)
	GetByName(ctx context.Context, name string) (*Permission, error)
	List(ctx context.Context) ([]Permission, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// ── Role-Permission Relationships ─────────────────────────────
	AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]Permission, error)
}