package usecase

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	jwtutil "github.com/moneymate-2026/moneymate-backend/shared/pkg/jwt"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/parallelrunners"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/qrcode"
)

const maxHandleAttempts = 5

type AuthUsecase interface {
	Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Logout(ctx context.Context, req LogoutRequest) error
	RefreshToken(ctx context.Context, req RefreshTokenRequest) (*RefreshTokenResponse, error)
	AdminLogin(ctx context.Context, req AdminLoginRequest) (*LoginResponse, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
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
	IssueRefreshToken(userID uuid.UUID) (token, tokenHash string, expiresAt time.Time, err error)
}

type authUsecase struct {
	userRepo         domain.UserRepository
	roleRepo         domain.RoleRepository
	refreshTokenRepo domain.RefreshTokenRepository
	outboxRepo       domain.OutboxRepository
	pinRepo          domain.UserPinRepository
	pinUsecase       UserPinUsecase
	store            domain.Store
	tx               domain.TxManager
	hasher           PasswordHasher
	idGen            IDGenerator
	issuer           TokenIssuer
	jwtCfg           jwtutil.Config
	staffRepo        domain.StaffRepository
}

func NewAuthUsecase(
	userRepo domain.UserRepository,
	roleRepo domain.RoleRepository,
	outboxRepo domain.OutboxRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	pinRepo domain.UserPinRepository,
	pinUsecase UserPinUsecase,
	store domain.Store,
	tx domain.TxManager,
	hasher PasswordHasher,
	idGen IDGenerator,
	issuer TokenIssuer,
	jwtCfg jwtutil.Config,
	staffRepo domain.StaffRepository,
) AuthUsecase {
	return &authUsecase{
		userRepo: userRepo, roleRepo: roleRepo, refreshTokenRepo: refreshTokenRepo,
		pinRepo: pinRepo, pinUsecase: pinUsecase, store: store, tx: tx,
		hasher: hasher, idGen: idGen, issuer: issuer, jwtCfg: jwtCfg, outboxRepo: outboxRepo,
		staffRepo: staffRepo,
	}
}

// ── Register ──────────────────────────────────────────────────────

func (u *authUsecase) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	email := normalizeEmail(req.Email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, apperrors.ErrInvalidInput
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}
	if len(req.PIN) != 6 {
		return nil, apperrors.ErrInvalidInput
	}
	if req.AccountType != domain.AccountTypeUser && req.AccountType != domain.AccountTypeMerchant &&
		req.AccountType != domain.AccountTypeAdmin {
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
	pinHash, err := u.hasher.Hash(req.PIN)
	if err != nil {
		return nil, fmt.Errorf("hash pin: %w", err)
	}

	userID, err := u.idGen.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}
	pinID, err := u.idGen.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate pin id: %w", err)
	}

	handle, err := u.generateHandle(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("generate handle: %w", err)
	}

	qrPayload := qrcode.BuildPaymentPayload(string(req.AccountType), handle)
	qrCode, err := qrcode.GenerateBase64(qrPayload)
	if err != nil {
		return nil, fmt.Errorf("generate qr code: %w", err)
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
		ID: userID, Email: email, Phone: phonePtr, FullName: strings.TrimSpace(req.FullName),
		Handle: handle, PasswordHash: &passwordHash, Status: domain.UserStatusActive,
		QRCode: qrCode,
	}

	outboxID, err := u.idGen.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate outbox id: %w", err)
	}
	eventPayload, err := json.Marshal(UserRegisteredEvent{UserID: userID, Handle: handle})
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
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
		if err := u.pinRepo.Create(ctx, &domain.UserPin{ID: pinID, UserID: user.ID, PinHash: pinHash}); err != nil {
			return fmt.Errorf("create pin: %w", err)
		}
		if err := u.outboxRepo.Insert(ctx, &domain.OutboxEvent{
			ID: outboxID, Topic: "user.registered", Payload: eventPayload,
		}); err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	accessToken, refreshToken, accessExp, refreshExp, err := u.issueAndPersistTokens(ctx, user)
	if err != nil {
		return nil, err
	}
	return &RegisterResponse{
		AccessToken: accessToken, RefreshToken: refreshToken,
		AccessExpiresAt: accessExp, RefreshExpiresAt: refreshExp,
		User: UserSummary{
			ID: user.ID, Email: user.Email, Handle: user.Handle, FullName: user.FullName,
			Status: string(user.Status), IsEmailVerified: true,
			QRCode: user.QRCode,
		},
	}, nil
}

// ── Login ─────────────────────────────────────────────────────────
func (u *authUsecase) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	email := normalizeEmail(req.Identifier)
	if email == "" || req.Password == "" || len(req.PIN) != 6 {
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

	if err := u.pinUsecase.VerifyPIN(ctx, user.ID, VerifyPINRequest{PIN: req.PIN}); err != nil {
		return nil, err
	}

	accessToken, refreshToken, accessExp, refreshExp, err := u.issueAndPersistTokens(ctx, user)
	if err != nil {
		return nil, err
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
			QRCode:          user.QRCode,
		},
	}, nil
}

func (u *authUsecase) AdminLogin(ctx context.Context, req AdminLoginRequest) (*LoginResponse, error) {
	email := normalizeEmail(req.Email)
	if email == "" || req.Password == "" {
		return nil, apperrors.ErrInvalidInput
	}

	staff, err := u.staffRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == apperrors.ErrUserNotFound {
			return nil, apperrors.ErrInvalidPassword
		}
		return nil, fmt.Errorf("get staff: %w", err)
	}
	if staff.PasswordHash == "" {
		return nil, apperrors.ErrInvalidPassword
	}
	ok, err := u.hasher.Verify(staff.PasswordHash, req.Password)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, apperrors.ErrInvalidPassword
	}
	if staff.Status != domain.UserStatusActive {
		return nil, apperrors.ErrForbidden
	}

	roles, err := u.staffRepo.GetRoles(ctx, staff.ID)
	if err != nil {
		return nil, fmt.Errorf("get staff roles: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}

	accessToken, accessExp, err := u.issuer.IssueAccessToken(staff.ID, "admin", roleNames, staff.TokenVersion)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	refreshToken, refreshHash, refreshExp, err := u.issuer.IssueRefreshToken(staff.ID)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	refreshID, err := u.idGen.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token id: %w", err)
	}
	if err := u.refreshTokenRepo.Create(ctx, &domain.RefreshToken{
		ID: refreshID, UserID: staff.ID, TokenHash: refreshHash, ExpiresAt: refreshExp,
	}); err != nil {
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken: accessToken, RefreshToken: refreshToken,
		AccessExpiresAt: accessExp, RefreshExpiresAt: refreshExp,
		User: UserSummary{
			ID: staff.ID, Email: staff.Email, Handle: "admin", FullName: staff.FullName,
			Status: string(staff.Status), IsEmailVerified: true,
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

// ── Refresh Token ──────────────────────────────────────────────────

func (u *authUsecase) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, apperrors.ErrInvalidInput
	}

	// 1. Parse the refresh token JWT
	claims, err := jwtutil.ParseRefreshToken(req.RefreshToken, u.jwtCfg.RefreshSecret)
	if err != nil {
		if err == apperrors.ErrTokenExpired {
			return nil, apperrors.ErrTokenExpired
		}
		return nil, apperrors.ErrInvalidToken
	}

	// 2. Look up the token hash in DB
	hash := jwtutil.HashToken(req.RefreshToken)
	stored, err := u.refreshTokenRepo.GetByTokenHash(ctx, hash)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return nil, apperrors.ErrInvalidToken
		}
		return nil, fmt.Errorf("lookup refresh token: %w", err)
	}

	// 3. Check if revoked
	if stored.RevokedAt != nil {
		return nil, apperrors.ErrInvalidToken
	}

	// 4. Check if expired
	if time.Now().After(stored.ExpiresAt) {
		return nil, apperrors.ErrTokenExpired
	}

	// 5. Parse user ID from claims
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, apperrors.ErrInvalidToken
	}

	// 6. Fetch user + roles + token version
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrInvalidToken
	}

	if user.Status != domain.UserStatusActive {
		return nil, apperrors.ErrForbidden
	}

	roles, tokenVersion, err := parallelrunners.Query2(ctx,
		func(ctx context.Context) ([]domain.Role, error) { return u.roleRepo.GetUserRoles(ctx, userID) },
		func(ctx context.Context) (int64, error) { return u.userRepo.GetTokenVersion(ctx, userID) },
	)
	if err != nil {
		return nil, fmt.Errorf("load refresh context: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}

	// 7. Issue new access token
	accessToken, accessExp, err := u.issuer.IssueAccessToken(user.ID, user.Handle, roleNames, tokenVersion)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	// 8. Issue new refresh token (rotation)
	newRefreshToken, newRefreshHash, refreshExp, err := u.issuer.IssueRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	// 9. Revoke old refresh token
	if err := u.refreshTokenRepo.Revoke(ctx, hash); err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	// 10. Persist new refresh token
	refreshID, err := u.idGen.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token id: %w", err)
	}
	if err := u.refreshTokenRepo.Create(ctx, &domain.RefreshToken{
		ID:        refreshID,
		UserID:    user.ID,
		TokenHash: newRefreshHash,
		ExpiresAt: refreshExp,
	}); err != nil {
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	return &RefreshTokenResponse{
		AccessToken:     accessToken,
		RefreshToken:    newRefreshToken,
		AccessExpiresAt: accessExp,
	}, nil
}

// ── Helpers ────────────────────────────────────────────────────

func (u *authUsecase) generateHandle(ctx context.Context, email string) (string, error) {
	local := strings.Split(email, "@")[0]
	local = sanitizeHandle(local)
	if local == "" {
		local = "user"
	}
	if len(local) > 15 {
		local = local[:15]
	}

	for range maxHandleAttempts {
		suffix, err := randomAlnum(4)
		if err != nil {
			return "", err
		}

		candidate := local + suffix + "@moneymate"

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

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (u *authUsecase) issueAndPersistTokens(ctx context.Context, user *domain.User) (accessToken, refreshToken string, accessExp, refreshExp time.Time, err error) {
	roles, tokenVersion, err := parallelrunners.Query2(ctx,
		func(ctx context.Context) ([]domain.Role, error) { return u.roleRepo.GetUserRoles(ctx, user.ID) },
		func(ctx context.Context) (int64, error) { return u.userRepo.GetTokenVersion(ctx, user.ID) },
	)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("load token context: %w", err)
	}
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}

	accessToken, accessExp, err = u.issuer.IssueAccessToken(user.ID, user.Handle, roleNames, tokenVersion)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("issue access token: %w", err)
	}
	var refreshHash string
	refreshToken, refreshHash, refreshExp, err = u.issuer.IssueRefreshToken(user.ID)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("issue refresh token: %w", err)
	}

	refreshID, err := u.idGen.NewV7()
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("generate refresh token id: %w", err)
	}
	if err := u.refreshTokenRepo.Create(ctx, &domain.RefreshToken{
		ID: refreshID, UserID: user.ID, TokenHash: refreshHash, ExpiresAt: refreshExp,
	}); err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("persist refresh token: %w", err)
	}
	return accessToken, refreshToken, accessExp, refreshExp, nil
}

// ── Password Management ──────────────────────────────────────────

func (u *authUsecase) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	if req.OldPassword == "" || req.NewPassword == "" || req.ConfirmPassword == "" {
		return apperrors.ErrInvalidInput
	}
	if req.NewPassword != req.ConfirmPassword {
		return apperrors.ErrPasswordMismatch
	}
	if req.NewPassword == req.OldPassword {
		return apperrors.ErrSamePassword
	}
	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user == nil || user.PasswordHash == nil {
		return apperrors.ErrIncorrectPassword
	}

	ok, err := u.hasher.Verify(*user.PasswordHash, req.OldPassword)
	if err != nil {
		return fmt.Errorf("verify old password: %w", err)
	}
	if !ok {
		return apperrors.ErrIncorrectPassword
	}

	newHash, err := u.hasher.Hash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if err := u.userRepo.UpdatePassword(ctx, userID, newHash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Revoke all existing refresh tokens so other devices cannot renew sessions,
	// while leaving the current device's active access token valid until normal expiry.
	if err := u.refreshTokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}

	return nil
}

func (u *authUsecase) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	email := normalizeEmail(req.Email)
	code := strings.TrimSpace(req.Code)

	if email == "" || code == "" || req.NewPassword == "" || req.ConfirmPassword == "" {
		return apperrors.ErrInvalidInput
	}
	if req.NewPassword != req.ConfirmPassword {
		return apperrors.ErrPasswordMismatch
	}
	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}

	// 1. Verify OTP with attempts limit (same pattern as VerifyRegistrationOTP)
	attempts, err := u.store.IncrementPasswordResetOTPAttempts(ctx, email, 15*time.Minute)
	if err != nil {
		return fmt.Errorf("increment password reset otp attempts: %w", err)
	}
	if attempts > 5 {
		_ = u.store.DeletePasswordResetOTP(ctx, email)
		return apperrors.ErrOTPTimout
	}

	storedHash, found, err := u.store.GetPasswordResetOTP(ctx, email)
	if err != nil {
		return fmt.Errorf("get password reset otp: %w", err)
	}
	if !found {
		return apperrors.ErrOTPExpired
	}

	suppliedHash := hashOTP(code)
	if !constantTimeEqual(storedHash, suppliedHash) {
		attemptsLeft := 5 - attempts
		if attemptsLeft < 0 {
			attemptsLeft = 0
		}
		details := map[string]interface{}{
			"attempts_left": int(attemptsLeft),
			"max_attempts":  5,
		}
		return apperrors.NewAppErrorWithDetails(
			400,
			"OTP_INVALID",
			"The code you entered is incorrect.",
			details,
			nil,
		)
	}

	// 2. Fetch user
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("get user by email: %w", err)
	}

	// 3. Hash new password & update
	newHash, err := u.hasher.Hash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if err := u.userRepo.UpdatePassword(ctx, user.ID, newHash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// 4. Invalidate all existing sessions (IncrementTokenVersion + store.UpgradeTokenVersion + RevokeAllForUser)
	_, _, err = parallelrunners.Query2(ctx,
		func(ctx context.Context) (int64, error) { return u.userRepo.IncrementTokenVersion(ctx, user.ID) },
		func(ctx context.Context) (struct{}, error) {
			return struct{}{}, u.store.UpgradeTokenVersion(ctx, user.ID.String())
		},
	)
	if err != nil {
		return fmt.Errorf("revoke all sessions (access): %w", err)
	}
	if err := u.refreshTokenRepo.RevokeAllForUser(ctx, user.ID); err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}

	// 5. Clean up OTP
	_ = u.store.DeletePasswordResetOTP(ctx, email)
	_ = u.store.ResetPasswordResetOTPAttempts(ctx, email)

	return nil
}
