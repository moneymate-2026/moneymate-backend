package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
)

type Handlers struct {
	Auth       *AuthHandler
	Role       *RoleHandler
	User       *UserHandler
	Staff      *StaffHandler
	UserPin    *UserPinHandler
	Permission *PermissionHandler
	Profile *ProfilePictureHandler
}

func RegisterRoutes(router fiber.Router, h *Handlers, internalSecret string) {
	registerAuthRoutes(router, h.Auth, internalSecret)
	registerRoleRoutes(router, h.Role)
	registerUserRoutes(router, h.User)
	registerStaffRoutes(router, h.Staff)
	registerUserPinRoutes(router, h.UserPin)
	registerPermissionRoutes(router, h.Permission)
	registerProfilePictureRoutes(router, h.Profile,h.User)
}

func registerAuthRoutes(router fiber.Router, h *AuthHandler, internalSecret string) {
	auth := router.Group("/auth")
	auth.Get("/health", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "ok",
			"service": "auth",
		})
	})
	auth.Post("/admin/login", h.AdminLogin)
	auth.Post("/login", h.Login)
	auth.Post("/logout", RequireUserID, h.Logout)
	auth.Post("/otp/send", h.SendRegistrationOTP)
	auth.Post("/otp/verify", h.VerifyRegistrationOTP)
	auth.Post("/password/change", RequireUserID, h.ChangePassword)
	auth.Post("/password/forgot", h.ForgotPassword)
	auth.Post("/password/reset", h.ResetPassword)
	auth.Post("/user/register", h.Register(domain.AccountTypeUser))
	auth.Post("/merchant/register", h.Register(domain.AccountTypeMerchant))

	internal := router.Group("/internal", RequireInternalSecret(internalSecret))
	internal.Post("/auth/verify-access-token", h.VerifyAccessToken)
	internal.Post("/auth/verify-transaction-token", h.VerifyTransactionToken)
	internal.Get("/auth/users/:id", h.GetUserByID)
}
func registerProfilePictureRoutes(router fiber.Router, h *ProfilePictureHandler, u *UserHandler) {
	profile := router.Group("/users/me/profile", RequireUserID)
	profile.Get("/", RequireUserID, u.GetMe)
	profile.Post("/presign", h.Presign)
	profile.Post("/", h.Set)
}

func registerRoleRoutes(router fiber.Router, h *RoleHandler) {
	roles := router.Group("/admin/roles")
	roles.Post("", h.CreateRole)
	roles.Post("/", h.CreateRole)
	
	roles.Get("", h.ListRoles)
	roles.Get("/", h.ListRoles)
	roles.Get("/:id", h.GetRole)
	roles.Put("/:id", h.UpdateRole)
	roles.Delete("/:id", h.DeleteRole)
	roles.Post("/assign", h.AssignRoleToUser)
	roles.Delete("/users/:userId/roles/:roleId", h.RemoveRoleFromUser)
	roles.Get("/users/:userId", h.GetUserRoles)
}

func registerUserRoutes(router fiber.Router, h *UserHandler) {
	users := router.Group("/admin/users")
	users.Post("", h.CreateUser)
	users.Post("/", h.CreateUser)
	users.Get("", h.ListUsers)
	users.Get("/", h.ListUsers)
	users.Get("/:id", h.GetUser)
	users.Put("/:id", h.UpdateUser)
	users.Patch("/:id/status", h.UpdateUserStatus)
	users.Delete("/:id", h.DeleteUser)

	router.Get("/users/me", RequireUserID, h.GetMe)
	router.Get("/users/lookup", RequireUserID, h.LookupByPhone)
}

func registerUserPinRoutes(router fiber.Router, h *UserPinHandler) {
	pins := router.Group("/user/pin", RequireUserID)
	pins.Post("/", h.SetPIN)
	pins.Put("/", h.UpdatePIN)
	pins.Post("/verify", h.VerifyPIN)
}

func registerPermissionRoutes(router fiber.Router, h *PermissionHandler) {
	permissions := router.Group("/admin/permissions")
	permissions.Post("", h.Create)
	permissions.Post("/", h.Create)
	permissions.Get("", h.List)
	permissions.Get("/", h.List)
	permissions.Get("/:id", h.Get)
	permissions.Delete("/:id", h.Delete)
	permissions.Post("/assign", h.AssignToRole)
	permissions.Delete("/roles/:roleId/permissions/:permissionId", h.RemoveFromRole)
	permissions.Get("/roles/:roleId", h.GetRolePermissions)
}

func registerStaffRoutes(router fiber.Router, h *StaffHandler) {
	staff := router.Group("/admin/staff")
	staff.Post("", h.CreateStaff)
	staff.Post("/", h.CreateStaff)
	staff.Get("", h.ListStaff)
	staff.Get("/", h.ListStaff)
	staff.Get("/:id", h.GetStaff)
	staff.Put("/:id", h.UpdateStaff)
	staff.Patch("/:id/status", h.UpdateStaffStatus)
	staff.Delete("/:id", h.DeleteStaff)
}