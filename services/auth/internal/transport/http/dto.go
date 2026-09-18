package http


//authentication

type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"omitempty,e164"`
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Password string `json:"password" validate:"required,min=8,max=72"`
	PIN      string `json:"pin" validate:"required,len=6,numeric"`
}
type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	PIN      string `json:"pin" validate:"required,len=6,numeric"`
}
type adminLoginRequest struct{
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
	AllDevices   bool   `json:"all_devices"`
}

type sendRegistrationOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type verifyRegistrationOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6,numeric"`
}

type changePasswordRequest struct {
	OldPassword     string `json:"old_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=256"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=8,max=256"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type resetPasswordRequest struct {
	Email           string `json:"email" validate:"required,email"`
	Code            string `json:"code" validate:"required,len=6,numeric"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=256"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=8,max=256"`
}


//users

type updateUserRequest struct {
	FullName *string `json:"full_name" validate:"omitempty,min=1"`
	Email    *string `json:"email" validate:"omitempty,email"`
	Phone    *string `json:"phone" validate:"omitempty"`
	Password *string `json:"password" validate:"omitempty,min=8"`
}

type updateUserStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending active suspended"`
}

//roles

type createRoleRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=50"`
	Description *string `json:"description" validate:"omitempty"`
}

type updateRoleRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=2,max=50"`
	Description *string `json:"description" validate:"omitempty"`
}

type assignRoleRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	RoleID string `json:"role_id" validate:"required,uuid"`
}

//permissions

type createPermissionRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type updatePermissionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type assignPermissionRequest struct {
	RoleID       string `json:"role_id" validate:"required"`
	PermissionID string `json:"permission_id" validate:"required"`
}

//admin

type createUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone"`
	FullName string `json:"full_name" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
	Role     string `json:"role" validate:"required"`
	PIN      string `json:"pin" validate:"required,len=6,numeric"`
}

// user pins

type setPINRequest struct {
	PIN string `json:"pin" validate:"required,len=6,numeric"`
}

type updatePINRequest struct {
	OldPIN string `json:"old_pin" validate:"required,len=6,numeric"`
	NewPIN string `json:"new_pin" validate:"required,len=6,numeric"`
}

type verifyPINRequest struct {
	PIN string `json:"pin" validate:"required,len=6,numeric"`
}

// transaction pin (payment flow)

