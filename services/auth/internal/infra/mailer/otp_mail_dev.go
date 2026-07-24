package mailer

import (
	"context"
	"log"
)

type DevOtpMailer struct{}

func NewDevOtpMail() *DevOtpMailer {
	return &DevOtpMailer{}
}

func (m *DevOtpMailer) SendOTP(ctx context.Context, toEmail, otp string) error {
	log.Printf("[DEV OTP] To: %s | Code: %s", toEmail, otp)
	return nil
}
