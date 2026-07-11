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

type roleRepo struct {
	q *db.Queries
}

func NewRoleRepo(pool *pgxpool.Pool) domain.RoleRepository {
	return &roleRepo{
		q: db.New(pool),
	}
}

// ── Role CRUD ─────────────────────────────────────────────────────

func (r *roleRepo) Create(ctx context.Context, role *domain.Role) error {
	result, err := r.q.CreateRole(ctx, db.CreateRoleParams{
		ID:          uuidToPgtype(role.ID),
		Name:        role.Name,
		Description: stringPtrToText(role.Description),
	})
	if err != nil {
		mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err {
			return mappedErr
		}
		return fmt.Errorf("create role: %w", err)
	}

	mapAuthRoleToDomain(result, role)
	return nil
}

func (r *roleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	row, err := r.q.GetRoleByID(ctx, uuidToPgtype(id))
	if err != nil {
		mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err {
			return nil, mappedErr
		}
		return nil, fmt.Errorf("get role by id: %w", err)
	}
	return toDomainRole(row), nil
}

func (r *roleRepo) GetByName(ctx context.Context, name string) (*domain.Role, error) {
	row, err := r.q.GetRoleByName(ctx, name)
	if err != nil {
		mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err {
			return nil, mappedErr
		}
		return nil, fmt.Errorf("get role by name: %w", err)
	}
	return toDomainRole(row), nil
}

func (r *roleRepo) List(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.q.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	roles := make([]domain.Role, len(rows))
	for i, row := range rows {
		roles[i] = *toDomainRole(row)
	}

	return roles, nil
}

func (r *roleRepo) Update(ctx context.Context, role *domain.Role) error {
	err := r.q.UpdateRole(ctx, db.UpdateRoleParams{
		ID:          uuidToPgtype(role.ID),
		Name:        role.Name,
		Description: stringPtrToText(role.Description),
	})
	if err != nil {
		mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err {
			return mappedErr
		}
		return fmt.Errorf("update role: %w", err)
	}
	return nil
}

func (r *roleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.q.DeleteRole(ctx, uuidToPgtype(id))
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

// ── User-Role Relationships ───────────────────────────────────────

func (r *roleRepo) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	err := r.q.AssignRoleToUser(ctx, db.AssignRoleToUserParams{
		UserID: uuidToPgtype(userID),
		RoleID: uuidToPgtype(roleID),
	})
	if err != nil {
		mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err {
			return mappedErr
		}
		return fmt.Errorf("assign role to user: %w", err)
	}
	return nil
}

func (r *roleRepo) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	err := r.q.RemoveRoleFromUser(ctx, db.RemoveRoleFromUserParams{
		UserID: uuidToPgtype(userID),
		RoleID: uuidToPgtype(roleID),
	})
	if err != nil {
		return fmt.Errorf("remove role from user: %w", err)
	}
	return nil
}

func (r *roleRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	rows, err := r.q.GetUserRoles(ctx, uuidToPgtype(userID))
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	roles := make([]domain.Role, len(rows))
	for i, row := range rows {
		roles[i] = *toDomainRole(row)
	}

	return roles, nil
}

// ── Type Conversion Helpers ───────────────────────────────────────


func toDomainRole(row db.AuthRole) *domain.Role {
	role := &domain.Role{
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}

	if row.ID.Valid {
		role.ID = uuid.UUID(row.ID.Bytes)
	}
	if row.Description.Valid {
		role.Description = &row.Description.String
	}

	return role
}

func mapAuthRoleToDomain(row db.AuthRole, role *domain.Role) {
	mapped := toDomainRole(row)
	role.ID = mapped.ID
	role.CreatedAt = mapped.CreatedAt
	role.UpdatedAt = mapped.UpdatedAt
}