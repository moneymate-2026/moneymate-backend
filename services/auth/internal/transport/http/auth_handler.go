package http

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	usecase "github.com/moneymate-2026/moneymate-backend/auth/internal/usecases"
	jwtutil "github.com/moneymate-2026/moneymate-backend/shared/pkg/jwt"
	response "github.com/moneymate-2026/moneymate-backend/shared/pkg/responses"
)

var validate = validator.New()

type AuthHandler struct {
	authUsecase usecase.AuthUsecase
	otpUsecase  usecase.OTPUsecase
	userRepo    domain.UserRepository
	jwtSecret   string
	userPin     usecase.UserPinUsecase
	redisClient *redis.Client
}

func NewAuthHandler(authUsecase usecase.AuthUsecase, otpUsecase usecase.OTPUsecase, userRepo domain.UserRepository, jwtSecret string, redisClient *redis.Client) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
		otpUsecase:  otpUsecase,
		userRepo:    userRepo,
		jwtSecret:   jwtSecret,
		redisClient: redisClient,
	}
}

func (h *AuthHandler) Register(accountType domain.AccountType) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req registerRequest
		if err := c.Bind().Body(&req); err != nil {
			return response.BadRequest(c, nil, "invalid request body")
		}
		if err := validate.Struct(req); err != nil {
			return response.BadRequest(c, formatValidationErrors(err), "validation failed")
		}

		ucReq := usecase.RegisterRequest{
			Email:       req.Email,
			Phone:       req.Phone,
			FullName:    req.FullName,
			Password:    req.Password,
			PIN:         req.PIN,
			AccountType: accountType,
		}

		resp, err := h.authUsecase.Register(c.Context(), ucReq)
		if err != nil {
			return handleError(c, err)
		}
		return response.Created(c, "account created successfully", resp)
	}
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationErrors(err), "validation failed")
	}

	ucReq := usecase.LoginRequest{
		Identifier: req.Email,
		Password:   req.Password,
		PIN:        req.PIN,
		UserAgent:  c.Get("User-Agent"),
		IPAddress:  c.IP(),
	}

	resp, err := h.authUsecase.Login(c.Context(), ucReq)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "login successful", resp)
}

func (h *AuthHandler) AdminLogin(c fiber.Ctx) error {
	var req adminLoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationErrors(err), "validation failed")
	}
	resp, err := h.authUsecase.AdminLogin(c.Context(), usecase.AdminLoginRequest{
		Email: req.Email, Password: req.Password,
	})
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "admin login successful", resp)
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return response.Unauthorized(c, "authentication required")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.Unauthorized(c, "invalid user session")
	}

	var req logoutRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}

	ucReq := usecase.LogoutRequest{
		UserID:       userID,
		RefreshToken: req.RefreshToken,
		AllDevices:   req.AllDevices,
	}

	if err := h.authUsecase.Logout(c.Context(), ucReq); err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "logged out successfully", nil)
}

func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationErrors(err), "validation failed")
	}

	resp, err := h.authUsecase.RefreshToken(c.Context(), usecase.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "token refreshed", resp)
}

func (h *AuthHandler) SendRegistrationOTP(c fiber.Ctx) error {
	var req sendRegistrationOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationErrors(err), "validation failed")
	}

	resp, err := h.otpUsecase.SendRegistrationOTP(c.Context(), usecase.SendRegistrationOTPRequest{
		Email: req.Email,
	})
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "verification code sent", resp)
}

func (h *AuthHandler) VerifyRegistrationOTP(c fiber.Ctx) error {
	var req verifyRegistrationOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationErrors(err), "validation failed")
	}

	resp, err := h.otpUsecase.VerifyRegistrationOTP(c.Context(), usecase.VerifyRegistrationOTPRequest{
		Email: req.Email,
		Code:  req.Code,
	})
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "email verified", resp)
}

func (h *AuthHandler) ChangePassword(c fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return response.Unauthorized(c, "authentication required")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.Unauthorized(c, "invalid user session")
	}

	var req changePasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationErrors(err), "validation failed")
	}

	ucReq := usecase.ChangePasswordRequest{
		OldPassword:     req.OldPassword,
		NewPassword:     req.NewPassword,
		ConfirmPassword: req.ConfirmPassword,
	}

	if err := h.authUsecase.ChangePassword(c.Context(), userID, ucReq); err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "password changed successfully", nil)
}

func (h *AuthHandler) ForgotPassword(c fiber.Ctx) error {
	var req forgotPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationErrors(err), "validation failed")
	}

	resp, err := h.otpUsecase.ForgotPassword(c.Context(), usecase.ForgotPasswordRequest{
		Email: req.Email,
	})
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "if your email is registered, a password reset code has been sent", resp)
}

func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var req resetPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationErrors(err), "validation failed")
	}

	ucReq := usecase.ResetPasswordRequest{
		Email:           req.Email,
		Code:            req.Code,
		NewPassword:     req.NewPassword,
		ConfirmPassword: req.ConfirmPassword,
	}

	if err := h.authUsecase.ResetPassword(c.Context(), ucReq); err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "password reset successfully", nil)
}

func (h *AuthHandler) VerifyAccessToken(c fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Token == "" {
		return response.BadRequest(c, nil, "token is required")
	}

	claims, err := jwtutil.ParseAccessToken(req.Token, h.jwtSecret)
	if err != nil {
		return response.Unauthorized(c, "invalid or expired token")
	}
	role := "user"
	if len(claims.Roles) > 0 {
		role = claims.Roles[0]
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return response.Unauthorized(c, "invalid user ID in token")
	}
	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil {
		return response.Unauthorized(c, "user not found")
	}

	return response.OK(c, "token verified", fiber.Map{
		"valid":       true,
		"user_id":     claims.UserID,
		"email":       user.Email,
		"role":        role,
		"merchant_id": "",
		"expires_at":  claims.ExpiresAt.Unix(),
	})
}

func (h *AuthHandler) VerifyTransactionToken(c fiber.Ctx) error {
	var req struct {
		Token         string `json:"token"`
		TransactionID string `json:"transaction_id"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Token == "" {
		return response.BadRequest(c, nil, "token and transaction_id are required")
	}

	claims, err := jwtutil.ParseTransactionToken(req.Token, h.jwtSecret)
	if err != nil {
		return response.Unauthorized(c, "invalid or expired transaction token")
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	claimed, err := h.redisClient.SetNX(c.Context(), "txtoken:used:"+claims.ID, "1", ttl).Result()
	if err != nil {
		return response.InternalServerError(c)
	}
	if !claimed {
		return response.Unauthorized(c, "transaction token already used")
	}

	return response.OK(c, "transaction token verified", fiber.Map{
		"valid":          true,
		"user_id":        claims.UserID,
		"transaction_id": req.TransactionID,
	})
}

func (h *AuthHandler) GetUserByID(c fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return response.BadRequest(c, nil, "user ID is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, nil, "invalid user ID format")
	}

	user, err := h.userRepo.GetByID(c.Context(), id)
	if err != nil {
		return response.NotFound(c, "user not found")
	}

	phone := ""
	if user.Phone != nil {
		phone = *user.Phone
	}

	profilePic := ""
	if user.ProfilePictureURL != nil {
		profilePic = *user.ProfilePictureURL
	}

	return response.OK(c, "user found", fiber.Map{
		"user_id":             user.ID.String(),
		"email":               user.Email,
		"full_name":           user.FullName,
		"handle":              user.Handle,
		"phone":               phone,
		"role":                "user",
		"qr_code":             user.QRCode,
		"profile_picture_url": profilePic,
	})
}
