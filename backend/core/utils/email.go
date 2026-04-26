package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/resend/resend-go/v2"
)

var resendClient *resend.Client

func init() {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		apiKey = "re_placeholder_replace_with_real_key"
	}
	resendClient = resend.NewClient(apiKey)
}

// SendOTPEmail sends a 6-digit OTP to the given email address via Resend.com.
func SendOTPEmail(ctx context.Context, toEmail, username, otp, purpose string) error {
	subject := "Your Echo OTP Code"
	action := "verify your email"
	if purpose == "reset" {
		subject = "Echo Password Reset Code"
		action = "reset your password"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: 'Inter', sans-serif; background: #f7fafc; padding: 40px; margin: 0;">
  <div style="max-width: 480px; margin: 0 auto; background: #ffffff; border-radius: 24px; padding: 40px; box-shadow: 0 4px 24px rgba(0,77,95,0.08);">
    <div style="text-align: center; margin-bottom: 32px;">
      <div style="display: inline-block; background: #b3ebff; padding: 16px; border-radius: 16px; margin-bottom: 16px;">
        <span style="font-size: 32px;">🔒</span>
      </div>
      <h1 style="font-family: 'Manrope', sans-serif; font-size: 24px; font-weight: 800; color: #004d5f; margin: 0;">The Echo</h1>
    </div>

    <p style="color: #3f484c; font-size: 15px; line-height: 1.6;">Hello <strong>%s</strong>,</p>
    <p style="color: #3f484c; font-size: 15px; line-height: 1.6;">Use the code below to %s. This code expires in <strong>10 minutes</strong>.</p>

    <div style="background: #f1f4f6; border-radius: 16px; padding: 24px; text-align: center; margin: 24px 0;">
      <span style="font-family: monospace; font-size: 40px; font-weight: 800; letter-spacing: 12px; color: #004d5f;">%s</span>
    </div>

    <p style="color: #6f797c; font-size: 13px; line-height: 1.6; text-align: center;">
      If you didn't request this, ignore this email. Your account is safe.
    </p>

    <hr style="border: none; border-top: 1px solid #ebeef0; margin: 24px 0;" />
    <p style="color: #6f797c; font-size: 11px; text-align: center; margin: 0;">
      Echo · End-to-End Encrypted · Zero Tracking
    </p>
  </div>
</body>
</html>`, username, action, otp)

	fromAddr := os.Getenv("RESEND_FROM_EMAIL")
	if fromAddr == "" {
		fromAddr = "onboarding@resend.dev" // Resend default sandbox address
	}

	params := &resend.SendEmailRequest{
		From:    fromAddr,
		To:      []string{toEmail},
		Subject: subject,
		Html:    html,
	}

	_, err := resendClient.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}
	return nil
}
