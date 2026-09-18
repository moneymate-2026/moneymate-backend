package mailer

import (
	"context"
	"fmt"

	sharedmailer "github.com/moneymate-2026/moneymate-backend/shared/pkg/mailer"
)


type OtpMailer struct{
	Client *sharedmailer.Client
}

func NewOtpMail(client *sharedmailer.Client)*OtpMailer{
	return &OtpMailer{Client: client}
}


func (m *OtpMailer) SendOTP(ctx context.Context, toEmail, otp string) error {
    subject := "Your MoneyMate verification code"
    textBody := fmt.Sprintf(
        "Your MoneyMate verification code is: %s\n\nThis code expires in 5 minutes. If you didn't request this, you can safely ignore this email.",
        otp,
    )

    htmlBody := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
  <h2>Verify your email</h2>
  <p>Your MoneyMate verification code is:</p>
  <p style="font-size: 32px; font-weight: bold; letter-spacing: 4px;">%s</p>
  <p style="color: #666; font-size: 14px;">This code expires shortly. If you didn't request this, you can safely ignore this email.</p>
</div>`, otp)

    return m.Client.Send(ctx, toEmail, subject, htmlBody, textBody)
}

func (m *OtpMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string) error {
    subject := "Reset your MoneyMate password"
    textBody := fmt.Sprintf(
        "Your MoneyMate password reset code is: %s\n\nThis code expires in 5 minutes. If you didn't request a password reset, please secure your account immediately.",
        otp,
    )

    htmlBody := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
  <h2>Reset your password</h2>
  <p>Your MoneyMate password reset code is:</p>
  <p style="font-size: 32px; font-weight: bold; letter-spacing: 4px; color: #111;">%s</p>
  <p style="color: #666; font-size: 14px;">This code expires in 5 minutes. If you didn't request this reset, you can safely ignore this email.</p>
</div>`, otp)

    return m.Client.Send(ctx, toEmail, subject, htmlBody, textBody)
}