package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	db "github.com/moneymate-2026/moneymate-backend/auth/sqlc/generated"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
)

type permissionRepo struct {
	q *db.Queries
}

func NewPermissionRepo(pool *pgxpool.Pool) domain.PermissionRepository {
	return &permissionRepo{
		q: db.New(pool),
	}
}

// ── Permission CRUD ───────────────────────────────────────────────

func (r *permissionRepo) Create(ctx context.Context, permission *domain.Permission) error {
	result, err := r.q.CreatePermission(ctx, db.CreatePermissionParams{
		ID:          uuidToPgtype(permission.ID),
		Name:        permission.Name,
		Description: stringPtrToText(permission.Description),
	})
	if err != nil {
		mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err {
			return mappedErr
		}
		return fmt.Errorf("create permission: %w", err)
	}

	mapAuthPermissionToDomain(result, permission)
	return nil
}

func (r *permissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	row, err := r.q.GetPermissionByID(ctx, uuidToPgtype(id))
	if err != nil {
		mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err {
			return nil, mappedErr
		}
		return nil, fmt.Errorf("get permission by id: %w", err)
	}
	return toDomainPermission(row), nil
}

func (r *permissionRepo) GetByName(ctx context.Context, name string) (*domain.Permission, error) {
	row, err := r.q.GetPermissionByName(ctx, name)
	if err != nil {
		mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err {
			return nil, mappedErr
		}
		return nil, fmt.Errorf("get permission by name: %w", err)
	}
	return toDomainPermission(row), nil
}

func (r *permissionRepo) List(ctx context.Context) ([]domain.Permission, error) {
	rows, err := r.q.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}

	permissions := make([]domain.Permission, len(rows))
	for i, row := range rows {
		permissions[i] = *toDomainPermission(row)
	}

	return permissions, nil
}

func (r *permissionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.q.DeletePermission(ctx, uuidToPgtype(id))
	if err != nil {
		return fmt.Errorf("delete permission: %w", err)
	}
	return nil
}

// ── Role-Permission Relationships ─────────────────────────────────

func (r *permissionRepo) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	err := r.q.AssignPermissionToRole(ctx, db.AssignPermissionToRoleParams{
		RoleID:       uuidToPgtype(roleID),
		PermissionID: uuidToPgtype(permissionID),
	})
	if err != nil {
		mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err {
			return mappedErr
		}
		return fmt.Errorf("assign permission to role: %w", err)
	}
	return nil
}

func (r *permissionRepo) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	err := r.q.RemovePermissionFromRole(ctx, db.RemovePermissionFromRoleParams{
		RoleID:       uuidToPgtype(roleID),
		PermissionID: uuidToPgtype(permissionID),
	})
	if err != nil {
		return fmt.Errorf("remove permission from role: %w", err)
	}
	return nil
}

func (r *permissionRepo) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	rows, err := r.q.GetRolePermissions(ctx, uuidToPgtype(roleID))
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}

	permissions := make([]domain.Permission, len(rows))
	for i, row := range rows {
		permissions[i] = *toDomainPermission(row)
	}

	return permissions, nil
}

// ── Type Conversion Helpers ───────────────────────────────────────

func toDomainPermission(row db.AuthPermission) *domain.Permission {
	perm := &domain.Permission{
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
	}

	if row.ID.Valid {
		perm.ID = uuid.UUID(row.ID.Bytes)
	}
	if row.Description.Valid {
		perm.Description = &row.Description.String
	}

	return perm
}

func mapAuthPermissionToDomain(row db.AuthPermission, perm *domain.Permission) {
	mapped := toDomainPermission(row)
	perm.ID = mapped.ID
	perm.CreatedAt = mapped.CreatedAt
}