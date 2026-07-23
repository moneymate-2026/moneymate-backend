package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	db "github.com/moneymate-2026/moneymate-backend/auth/sqlc/generated"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/pgxtx"
)

type userRepo struct {
    q *db.Queries
}

func NewUserRepo(pool *pgxpool.Pool) domain.UserRepository {
    return &userRepo{
        q: db.New(pool),
    }
}


func (r *userRepo) queries(ctx context.Context) *db.Queries {
	if tx, ok := pgxtx.FromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

// ── Create ────────────────────────────────────────────────────────

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
    result, err := r.queries(ctx).CreateUser(ctx, db.CreateUserParams{
        ID:           uuidToPgtype(user.ID),
        Email:        user.Email,
        Phone:        stringPtrToText(user.Phone),
        FullName:     user.FullName,
        Handle:       user.Handle,
        PasswordHash: stringPtrToText(user.PasswordHash),
    })
    if err != nil {
        mappedErr := apperrors.MapDBErrors(err)
		if mappedErr != err { 
            return mappedErr 
        }
        return fmt.Errorf("create user: %w", err)
    }

    // map result back onto the domain struct
    // so caller has the full created user including DB defaults
    mapAuthUserToDomain(result, user)
    return nil
}



// ── Reads ─────────────────────────────────────────────────────────

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
    row, err := r.q.GetUserByID(ctx, uuidToPgtype(id))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apperrors.ErrUserNotFound
        }
        return nil, fmt.Errorf("get user by id: %w", err)
    }
    return toDomainUser(row), nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
    row, err := r.q.GetUserByEmail(ctx, email)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apperrors.ErrUserNotFound
        }
        return nil, fmt.Errorf("get user by email: %w", err)
    }
    return toDomainUser(row), nil
}

func (r *userRepo) GetByHandle(ctx context.Context, handle string) (*domain.User, error) {
    row, err := r.q.GetUserByHandle(ctx, handle)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apperrors.ErrUserNotFound
        }
        return nil, fmt.Errorf("get user by handle: %w", err)
    }
    return toDomainUser(row), nil
}

func (r *userRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
    row, err := r.q.GetUserByPhone(ctx, pgtype.Text{String: phone, Valid: true})
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apperrors.ErrUserNotFound
        }
        return nil, fmt.Errorf("get user by phone: %w", err)
    }
    return toDomainUser(row), nil
}

// ── Existence Checks ──────────────────────────────────────────────

func (r *userRepo) EmailExists(ctx context.Context, email string) (bool, error) {
    exists, err := r.q.EmailExists(ctx, email)
    if err != nil {
        return false, fmt.Errorf("email exists: %w", err)
    }
    return exists, nil
}

func (r *userRepo) HandleExists(ctx context.Context, handle string) (bool, error) {
    exists, err := r.q.HandleExists(ctx, handle)
    if err != nil {
        return false, fmt.Errorf("handle exists: %w", err)
    }
    return exists, nil
}

func (r *userRepo) PhoneExists(ctx context.Context, phone string) (bool, error) {
    exists, err := r.q.PhoneExists(ctx, pgtype.Text{String: phone, Valid: true})
    if err != nil {
        return false, fmt.Errorf("phone exists: %w", err)
    }
    return exists, nil
}

func (r *userRepo) CheckUniqueFields(ctx context.Context, email, handle, phone string) error {
    emailExists, err := r.EmailExists(ctx, email)
    if err != nil {
        return err
    }
    if emailExists {
        return apperrors.ErrAlreadyExists
    }

    handleExists, err := r.HandleExists(ctx, handle)
    if err != nil {
        return err
    }
    if handleExists {
        return apperrors.ErrAlreadyExists
    }

    if phone != "" {
        phoneExists, err := r.PhoneExists(ctx, phone)
        if err != nil {
            return err
        }
        if phoneExists {
            return apperrors.ErrAlreadyExists
        }
    }

    return nil
}

// ── Updates ───────────────────────────────────────────────────────

func (r *userRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
    err := r.q.UpdatePassword(ctx, db.UpdatePasswordParams{
        ID:           uuidToPgtype(userID),
        PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
    })
    if err != nil {
        return fmt.Errorf("update password: %w", err)
    }
    return nil
}

func (r *userRepo) UpdateStatus(ctx context.Context, userID uuid.UUID, status domain.UserStatus) error {
    err := r.q.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
        ID:     uuidToPgtype(userID),
        Status: db.AuthUserStatus(status),
    })
    if err != nil {
        return fmt.Errorf("update status: %w", err)
    }
    return nil
}

func (r *userRepo) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
    if err :=  r.queries(ctx).VerifyEmail(ctx, uuidToPgtype(userID)); err != nil {
        return fmt.Errorf("verify email: %w", err)
    }
    return nil
}

func (r *userRepo) VerifyPhone(ctx context.Context, userID uuid.UUID) error {
    if err := r.q.VerifyPhone(ctx, uuidToPgtype(userID)); err != nil {
        return fmt.Errorf("verify phone: %w", err)
    }
    return nil
}

func (r *userRepo) SoftDelete(ctx context.Context, userID uuid.UUID) error {
    if err := r.q.SoftDeleteUser(ctx, uuidToPgtype(userID)); err != nil {
        return fmt.Errorf("soft delete user: %w", err)
    }
    return nil
}

// ── Token Version ─────────────────────────────────────────────────

func (r *userRepo) IncrementTokenVersion(ctx context.Context, userID uuid.UUID) (int64, error) {
    version, err := r.q.IncrementTokenVersion(ctx, uuidToPgtype(userID))
    if err != nil {
        return 0, fmt.Errorf("increment token version: %w", err)
    }
    return version, nil
}

func (r *userRepo) GetTokenVersion(ctx context.Context, userID uuid.UUID) (int64, error) {
    version, err := r.q.GetTokenVersion(ctx, uuidToPgtype(userID))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return 0, apperrors.ErrUserNotFound
        }
        return 0, fmt.Errorf("get token version: %w", err)
    }
    return version, nil
}

// ── List ──────────────────────────────────────────────────────────


func (r *userRepo) ListUsers(ctx context.Context, filter domain.ListUsersFilter, page domain.Pagination) (*domain.ListUsersResult, error) {
	if page.PageSize <= 0 {
		page.PageSize = 20
	}
	if page.PageSize > 100 {
		page.PageSize = 100
	}
	if page.Page <= 0 {
		page.Page = 1
	}
	offset := (page.Page - 1) * page.PageSize

	sortBy := filter.SortBy
	switch sortBy {
	case "email", "full_name", "created_at":
	default:
		sortBy = "created_at"
	}

	var statusParam db.NullAuthUserStatus 
	if filter.Status != "" {
		statusParam = db.NullAuthUserStatus{AuthUserStatus: db.AuthUserStatus(filter.Status), Valid: true}
	}

	var searchParam pgtype.Text
	if filter.Search != "" {
		searchParam = pgtype.Text{String: filter.Search, Valid: true}
	}

	rows, err := r.q.ListUsers(ctx, db.ListUsersParams{
		Status:   statusParam,
		Search:   searchParam,
		SortBy:   sortBy,
		SortDesc: filter.SortDesc,
		Limit:    int32(page.PageSize),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	total, err := r.q.CountUsers(ctx, db.CountUsersParams{
		Status: statusParam,
		Search: searchParam,
	})
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	users := make([]domain.User, len(rows))
	for i, row := range rows {
		users[i] = *toDomainUser(row)
	}

	return &domain.ListUsersResult{
		Users:      users,
		TotalCount: total,
	}, nil
}

// ── Type Conversion Helpers ───────────────────────────────────────

func toDomainUser(row db.AuthUser) *domain.User {
    user := &domain.User{
        Email:           row.Email,
        FullName:        row.FullName,
        Handle:          row.Handle,
        Status:          domain.UserStatus(row.Status),
        TokenVersion:    row.TokenVersion,
        IsEmailVerified: row.IsEmailVerified,
        IsPhoneVerified: row.IsPhoneVerified,
        CreatedAt:       row.CreatedAt.Time,
        UpdatedAt:       row.UpdatedAt.Time,
    }


    if row.ID.Valid {
        user.ID = uuid.UUID(row.ID.Bytes)
    }

    if row.Phone.Valid {
        user.Phone = &row.Phone.String
    }
    if row.PasswordHash.Valid {
        user.PasswordHash = &row.PasswordHash.String
    }

    return user
}

func mapAuthUserToDomain(row db.AuthUser, user *domain.User) {
    mapped := toDomainUser(row)
    user.ID        = mapped.ID
    user.Status    = mapped.Status
    user.CreatedAt = mapped.CreatedAt
    user.UpdatedAt = mapped.UpdatedAt
}

func uuidToPgtype(id uuid.UUID) pgtype.UUID {
    return pgtype.UUID{
        Bytes: id,
        Valid: true,
    }
}

func stringPtrToText(s *string) pgtype.Text {
    if s == nil {
        return pgtype.Text{Valid: false}
    }
    return pgtype.Text{String: *s, Valid: true}
}
