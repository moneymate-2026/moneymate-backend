package apperrors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// General Errors
var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrUnauthorized      = errors.New("you are unauthorized, please login")
	ErrForbidden         = errors.New("you do not own this account")
	ErrInternal          = errors.New("internal server error")
	ErrDependencyFailure = errors.New("dependency failure")
	ErrBadRequest        = errors.New("bad request")
)

// User , Auth Specific
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyTaken = errors.New("email already taken")
	ErrPhoneAlreadyTaken = errors.New("phone number already taken")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrIncorrectPassword = errors.New("incorrect current password")
	ErrSamePassword      = errors.New("new password cannot be the same as old password")
	ErrPasswordMismatch  = errors.New("confirm password does not match new password")
	ErrOTPExpired        = errors.New("otp expired")
	ErrOTPInvalid        = errors.New("otp invalid")
	ErrOTPTimout         = errors.New("otp max tries reached")
	ErrOTPTimeout        = ErrOTPTimout
	ErrOAuthFailure      = errors.New("oauth authentication failed")
	ErrEmailNotVerified  = errors.New("email not verified")
)

// Financial & Transaction Specific
var (
	ErrInsufficientFunds  = errors.New("insufficient funds for this transaction")
	ErrTransactionLocked  = errors.New("transaction is currently locked or processing")
	ErrDailyLimitReached  = errors.New("daily transaction limit reached")
	ErrIdempotencyKeyUsed = errors.New("this transaction has already been processed")
	ErrPodNonZeroBalance  = errors.New("cannot delete pod with non-zero balance; withdraw all funds first")
)

//jwt token errors
var(
	ErrTokenExpired = errors.New("token expired")
    ErrInvalidToken = errors.New("invalid token")
)

// PIN & transaction auth
var (
	ErrPinNotSet  = errors.New("no transaction pin set")
	ErrInvalidPIN = errors.New("invalid transaction pin")
	ErrPinLocked  = errors.New("transaction pin is temporarily locked")
)

// AppError represents a structured HTTP error safely returned to the frontend.
type AppError struct {
	StatusCode int                    `json:"status_code"`
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"`
}


// Error implements the standard Go error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap allows standard errors.Is and errors.As to work with AppError.
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError is a convenience constructor for standard errors with no extra metadata.
func NewAppError(statusCode int, code, message string, underlying error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Err:        underlying,
	}
}

// NewAppErrorWithDetails allows attaching contextual metadata to the error response.
func NewAppErrorWithDetails(statusCode int, code, message string, details map[string]interface{}, underlying error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Details:    details,
		Err:        underlying,
	}
}

// MapDBErrors translates raw PostgreSQL / pgx driver errors into strongly typed domain errors.
func MapDBErrors(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			switch pgErr.ConstraintName {
			case "users_email_key", "users_email_idx":
				return ErrEmailAlreadyTaken
			case "users_phone_key", "users_phone_idx":
				return ErrPhoneAlreadyTaken
			default:
				return ErrAlreadyExists
			}
		case "23503": // foreign_key_violation
			return ErrNotFound
		case "23514": // check_violation
			if pgErr.ConstraintName == "chk_positive_balance" {
				return ErrInsufficientFunds
			}
			return ErrInvalidInput
		case "40001":
			return ErrTransactionLocked
		}
	}

	return err
}

// ParseError looks at a domain error and packages it into a safe AppError for the frontend.
func ParseError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr // It's already an AppError, return it directly
	}
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrUserNotFound):
		return NewAppError(http.StatusNotFound, "NOT_FOUND", "The requested resource was not found.", err)

	case errors.Is(err, ErrIncorrectPassword):
		return NewAppError(http.StatusUnauthorized, "INCORRECT_PASSWORD", "The current password you entered is incorrect.", err)

	case errors.Is(err, ErrSamePassword):
		return NewAppError(http.StatusBadRequest, "SAME_PASSWORD", "New password cannot be identical to your old password.", err)

	case errors.Is(err, ErrPasswordMismatch):
		return NewAppError(http.StatusBadRequest, "PASSWORD_MISMATCH", "Confirm password does not match new password.", err)

	case errors.Is(err, ErrInvalidPassword):
		return NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password.", err)

	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrBadRequest):
		return NewAppError(http.StatusBadRequest, "BAD_REQUEST", "Invalid request parameters.", err)

	case errors.Is(err, ErrOTPExpired):
		return NewAppError(http.StatusBadRequest, "OTP_EXPIRED", "The verification code has expired.", err)

	case errors.Is(err, ErrOTPInvalid):
		return NewAppError(http.StatusBadRequest, "OTP_INVALID", "The code you entered is incorrect.", err)

	case errors.Is(err, ErrOTPTimout):
		return NewAppError(http.StatusTooManyRequests, "OTP_MAX_TRIES", "Maximum verification attempts exceeded.", err)

	case errors.Is(err, ErrEmailAlreadyTaken):
		return NewAppError(http.StatusConflict, "EMAIL_TAKEN", "This email is already in use.", err)

	case errors.Is(err, ErrInsufficientFunds):
		return NewAppError(http.StatusPaymentRequired, "INSUFFICIENT_FUNDS", "Your wallet balance is too low for this transaction.", err)

	case errors.Is(err, ErrPodNonZeroBalance):
		return NewAppError(http.StatusConflict, "POD_NON_ZERO_BALANCE", "Cannot delete pod with non-zero balance. Please withdraw all funds first.", err)

	case errors.Is(err, ErrTransactionLocked):
		return NewAppError(http.StatusConflict, "TRANSACTION_LOCKED", "This account is currently processing another transaction. Please try again in a few seconds.", err)

	case errors.Is(err, ErrUnauthorized):
		return NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "Please log in to continue.", err)
	case errors.Is(err, ErrEmailNotVerified):
		return NewAppError(http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Please verify your email before completing registration.", err)

	case errors.Is(err, ErrPinLocked):
		return NewAppError(http.StatusTooManyRequests, "PIN_LOCKED", "Too many failed attempts. Try again in 15 minutes.", err)
	case errors.Is(err, ErrPinNotSet):
		return NewAppError(http.StatusBadRequest, "PIN_NOT_SET", "Set a transaction PIN first.", err)
	case errors.Is(err, ErrInvalidPIN):
		return NewAppError(http.StatusBadRequest, "INVALID_PIN", "Incorrect transaction PIN.", err)
	case errors.Is(err, ErrForbidden):
		return NewAppError(http.StatusForbidden, "FORBIDDEN", "You do not have access to this resource.", err)

	default:
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Something went wrong on our end. Please try again later.", err)
	}
}

// how to use

// func (h *AuthHandler) Login(c *fiber.Ctx) error {
// 1. Call your usecase
// token, err := h.usecase.Login(c.Context(), req.Email, req.Password)

// 2. If it fails, parse it and return
// if err != nil {
// appErr := apperrors.ParseError(err)
// You would also log 'appErr.Err' to your internal logging system here
// return c.Status(appErr.StatusCode).JSON(appErr)
// }

// 3. Success
// return c.Status(200).JSON(token)
// }
