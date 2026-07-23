package usecase

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	jwtutil "github.com/moneymate-2026/moneymate-backend/shared/pkg/jwt"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/parallelrunners"
)

const maxHandleAttempts = 5

type AuthUsecase interface {
	Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Logout(ctx context.Context, req LogoutRequest) error
}

// ── DI interfaces ────────────────────────────────────────────────
// Structurally satisfied by hasher.Argon2Hasher, idgen.Generator,
// and tokenissuer.Issuer respectively — no changes needed to those.

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) (bool, error)
}

type IDGenerator interface {
	NewV7() (uuid.UUID, error)
}

type TokenIssuer interface {
	IssueAccessToken(userID uuid.UUID, handle string, roles []string, tokenVersion int64) (string, time.Time, error)
	IssueRefreshToken(userID uuid.UUID, deviceID string) (token, tokenHash string, expiresAt time.Time, err error)
}

type authUsecase struct {
	userRepo         domain.UserRepository
	roleRepo         domain.RoleRepository
	refreshTokenRepo domain.RefreshTokenRepository
	store            domain.Store
	tx               domain.TxManager
	hasher           PasswordHasher
	idGen            IDGenerator
	issuer           TokenIssuer
	jwtCfg           jwtutil.Config
}

func NewAuthUsecase(
	userRepo domain.UserRepository,
	roleRepo domain.RoleRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	store domain.Store,
	tx domain.TxManager,
	hasher PasswordHasher,
	idGen IDGenerator,
	issuer TokenIssuer,
	jwtCfg jwtutil.Config,
) AuthUsecase {
	return &authUsecase{
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		refreshTokenRepo: refreshTokenRepo,
		store:            store,
		tx:               tx,
		hasher:           hasher,
		idGen:            idGen,
		issuer:           issuer,
		jwtCfg:           jwtCfg,
	}
}

// ── Register ──────────────────────────────────────────────────────

func (u *authUsecase) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	email := normalizeEmail(req.Email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, apperrors.ErrInvalidInput
	}
	if err:=validatePassword(req.Password);err!=nil{
		return nil,err
	}
	if req.AccountType != domain.AccountTypeUser && req.AccountType != domain.AccountTypeMerchant {
		return nil, apperrors.ErrInvalidInput
	}
	phone := strings.TrimSpace(req.Phone)

	emailExists, phoneExists, err := parallelrunners.Query2(ctx,
		func(ctx context.Context) (bool, error) { return u.userRepo.EmailExists(ctx, email) },
		func(ctx context.Context) (bool, error) {
			if phone == "" {
				return false, nil
			}
			return u.userRepo.PhoneExists(ctx, phone)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("check uniqueness: %w", err)
	}
	if emailExists {
		return nil, apperrors.ErrEmailAlreadyTaken
	}
	if phoneExists {
		return nil, apperrors.ErrPhoneAlreadyTaken
	}

	verified, err := u.store.ConsumeEmailVerified(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("consume email verified: %w", err)
	}
	if !verified {
		return nil, apperrors.ErrEmailNotVerified
	}

	passwordHash, err := u.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	userID, err := u.idGen.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}

	handle, err := u.generateHandle(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("generate handle: %w", err)
	}

	role, err := u.roleRepo.GetByName(ctx, string(req.AccountType))
	if err != nil {
		return nil, fmt.Errorf("resolve role %q: %w", req.AccountType, err)
	}

	var phonePtr *string
	if phone != "" {
		phonePtr = &phone
	}

	user := &domain.User{
		ID:           userID,
		Email:        email,
		Phone:        phonePtr,
		FullName:     strings.TrimSpace(req.FullName),
		Handle:       handle,
		PasswordHash: &passwordHash,
		Status:       domain.UserStatusActive,
	}

	err = u.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := u.userRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if err := u.userRepo.VerifyEmail(ctx, user.ID); err != nil {
			return fmt.Errorf("verify email: %w", err)
		}
		if err := u.roleRepo.AssignRoleToUser(ctx, user.ID, role.ID, nil); err != nil {
			return fmt.Errorf("assign role: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &RegisterResponse{
		UserID: user.ID,
		Email:  user.Email,
		Handle: user.Handle,
		Status: string(domain.UserStatusActive),
	}, nil
}

// ── Login ─────────────────────────────────────────────────────────

func (u *authUsecase) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	email := normalizeEmail(req.Identifier)
	if email == "" || req.Password == "" || req.DeviceID == "" {
		return nil, apperrors.ErrInvalidInput
	}

	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == apperrors.ErrUserNotFound {
			return nil, apperrors.ErrInvalidPassword
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if user.PasswordHash == nil {
		return nil, apperrors.ErrInvalidPassword
	}
	ok, err := u.hasher.Verify(*user.PasswordHash, req.Password)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, apperrors.ErrInvalidPassword
	}
	if user.Status != domain.UserStatusActive {
		return nil, apperrors.ErrForbidden
	}

	roles, tokenVersion, err := parallelrunners.Query2(ctx,
		func(ctx context.Context) ([]domain.Role, error) { return u.roleRepo.GetUserRoles(ctx, user.ID) },
		func(ctx context.Context) (int64, error) { return u.userRepo.GetTokenVersion(ctx, user.ID) },
	)
	if err != nil {
		return nil, fmt.Errorf("load login context: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}

	accessToken, accessExp, err := u.issuer.IssueAccessToken(user.ID, user.Handle, roleNames, tokenVersion)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	refreshToken, refreshHash, refreshExp, err := u.issuer.IssueRefreshToken(user.ID, req.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	refreshID, err := u.idGen.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token id: %w", err)
	}
	if err := u.refreshTokenRepo.Create(ctx, &domain.RefreshToken{
		ID:        refreshID,
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: refreshExp,
	}); err != nil {
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		User: UserSummary{
			ID:              user.ID,
			Email:           user.Email,
			Handle:          user.Handle,
			FullName:        user.FullName,
			Status:          string(user.Status),
			IsEmailVerified: user.IsEmailVerified,
		},
	}, nil
}

// ── Logout ────────────────────────────────────────────────────────

func (u *authUsecase) Logout(ctx context.Context, req LogoutRequest) error {
	if req.AllDevices {
		_, _, err := parallelrunners.Query2(ctx,
			func(ctx context.Context) (int64, error) { return u.userRepo.IncrementTokenVersion(ctx, req.UserID) },
			func(ctx context.Context) (struct{}, error) {
				return struct{}{}, u.store.UpgradeTokenVersion(ctx, req.UserID.String())
			},
		)
		if err != nil {
			return fmt.Errorf("revoke all sessions (access): %w", err)
		}
		if err := u.refreshTokenRepo.RevokeAllForUser(ctx, req.UserID); err != nil {
			return fmt.Errorf("revoke all sessions (refresh): %w", err)
		}
		return nil
	}

	if req.RefreshToken == "" {
		return apperrors.ErrInvalidInput
	}
	claims, err := jwtutil.ParseRefreshToken(req.RefreshToken, u.jwtCfg.RefreshSecret)
	if err != nil {
		if err == apperrors.ErrTokenExpired {
			return nil // already unusable, nothing to revoke
		}
		return err
	}
	if claims.UserID != req.UserID.String() {
		return apperrors.ErrForbidden
	}

	hash := jwtutil.HashToken(req.RefreshToken)
	stored, err := u.refreshTokenRepo.GetByTokenHash(ctx, hash)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return nil // unknown token, treat as already logged out
		}
		return fmt.Errorf("lookup refresh token: %w", err)
	}
	if stored.RevokedAt != nil {
		return nil // already revoked, idempotent
	}

	if err := u.refreshTokenRepo.Revoke(ctx, hash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// ── Helpers ────────────────────────────────────────────────────

func (u *authUsecase) generateHandle(ctx context.Context, email string) (string, error) {
	local := strings.Split(email, "@")[0]
	local = sanitizeHandle(local)
	if local == "" {
		local = "user"
	}
	if len(local) > 20 {
		local = local[:20]
	}

	for i := 0; i < maxHandleAttempts; i++ {
		suffix, err := randomAlnum(4)
		if err != nil {
			return "", err
		}
		candidate := local + suffix
		exists, err := u.userRepo.HandleExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not generate unique handle after %d attempts", maxHandleAttempts)
}

func sanitizeHandle(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

const alnumCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomAlnum(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = alnumCharset[int(b[i])%len(alnumCharset)]
	}
	return string(b), nil
}
// ── Helpers ──────────────────────────────────────────────────────

func normalizeEmail(email string) string {
    return strings.ToLower(strings.TrimSpace(email))
}

func validatePassword(pw string) error {
    if len(pw) < 8 {
        return apperrors.ErrInvalidPassword
    }
    if len(pw) > 256 {
        return apperrors.ErrInvalidPassword
    }
    return nil
}

// func (u *authUsecase) getDummyHash() string {
//     u.dummyHashOnce.Do(func() {
//         u.dummyHash, _ = u.hasher.Hash("dummy-password-for-timing-safety-only")
//     })
//     return u.dummyHash
// }

func constantTimeEqual(a, b string) bool {
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}


func hashTokenForLookup(raw string) string {
    return jwtutil.HashToken(raw)
}