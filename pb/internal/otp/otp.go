package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"net/mail"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/types"
)

// GenerateOTP generates a 6-digit OTP code
func GenerateOTP() (string, error) {
	max := big.NewInt(999999)
	min := big.NewInt(100000)
	
	n, err := rand.Int(rand.Reader, max.Sub(max, min).Add(max, big.NewInt(1)))
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%06d", n.Add(n, min).Int64()), nil
}

// CreateOTP creates and stores an OTP for a user
func CreateOTP(app core.App, userID, email, purpose string) (string, error) {
	// Generate OTP code
	otpCode, err := GenerateOTP()
	if err != nil {
		return "", err
	}

	// Set expiration time (10 minutes from now)
	expiresAt := time.Now().Add(10 * time.Minute)

	// Create OTP record
	collection, err := app.FindCollectionByNameOrId("user_otps")
	if err != nil {
		return "", err
	}

	record := core.NewRecord(collection)
	record.Set("user_id", userID)
	record.Set("otp_code", otpCode)
	record.Set("purpose", purpose)
	record.Set("expires_at", expiresAt)
	record.Set("used", false)
	record.Set("email", email)

	if err := app.Save(record); err != nil {
		return "", err
	}

	return otpCode, nil
}

// VerifyOTP verifies an OTP code for a user
func VerifyOTP(app core.App, userID, otpCode, purpose string) error {
	// Find the OTP record
	collection, err := app.FindCollectionByNameOrId("user_otps")
	if err != nil {
		return err
	}

	record, err := app.FindFirstRecordByFilter(
		collection,
		"user_id = {:userId} && otp_code = {:otpCode} && purpose = {:purpose} && used = false",
		map[string]any{
			"userId":  userID,
			"otpCode": otpCode,
			"purpose": purpose,
		},
	)
	if err != nil {
		return fmt.Errorf("invalid or expired OTP")
	}

	// Check if OTP has expired
	expiresAtField := record.Get("expires_at")
	var expiresAt time.Time
	
	// Handle both types.DateTime and time.Time
	switch v := expiresAtField.(type) {
	case types.DateTime:
		expiresAt = v.Time()
	case time.Time:
		expiresAt = v
	default:
		return fmt.Errorf("invalid expires_at field type")
	}
	
	if time.Now().After(expiresAt) {
		return fmt.Errorf("OTP has expired")
	}

	// Mark OTP as used
	record.Set("used", true)
	if err := app.Save(record); err != nil {
		return err
	}

	return nil
}

// SendOTPEmail sends an OTP via email
func SendOTPEmail(app core.App, email, otpCode, purpose string) error {
	var subject, body string
	
	switch purpose {
	case "signup_verification":
		subject = "Verify Your Account - OTP Code"
		body = fmt.Sprintf(`
		<h2>Welcome to Pulse!</h2>
		<p>Thank you for signing up. To complete your registration, please enter this verification code:</p>
		<div style="background: #f8f9fa; padding: 20px; margin: 20px 0; text-align: center; border-radius: 8px;">
			<h1 style="font-size: 32px; color: #007bff; margin: 0; letter-spacing: 8px;">%s</h1>
		</div>
		<p>This code will expire in 10 minutes.</p>
		<p>If you didn't create this account, please ignore this email.</p>
		<p>Best regards,<br>The Pulse Team</p>
		`, otpCode)
	case "email_change":
		subject = "Confirm Email Change - OTP Code"
		body = fmt.Sprintf(`
		<h2>Confirm Your New Email Address</h2>
		<p>Please enter this verification code to confirm your email change:</p>
		<div style="background: #f8f9fa; padding: 20px; margin: 20px 0; text-align: center; border-radius: 8px;">
			<h1 style="font-size: 32px; color: #007bff; margin: 0; letter-spacing: 8px;">%s</h1>
		</div>
		<p>This code will expire in 10 minutes.</p>
		<p>If you didn't request this email change, please contact support.</p>
		<p>Best regards,<br>The Pulse Team</p>
		`, otpCode)
	case "password_reset":
		subject = "Password Reset - OTP Code"
		body = fmt.Sprintf(`
		<h2>Reset Your Password</h2>
		<p>Please enter this verification code to reset your password:</p>
		<div style="background: #f8f9fa; padding: 20px; margin: 20px 0; text-align: center; border-radius: 8px;">
			<h1 style="font-size: 32px; color: #007bff; margin: 0; letter-spacing: 8px;">%s</h1>
		</div>
		<p>This code will expire in 10 minutes.</p>
		<p>If you didn't request a password reset, please ignore this email.</p>
		<p>Best regards,<br>The Pulse Team</p>
		`, otpCode)
	default:
		return fmt.Errorf("unknown OTP purpose: %s", purpose)
	}

	message := &mailer.Message{
		From: mail.Address{
			Address: app.Settings().Meta.SenderAddress,
			Name:    app.Settings().Meta.SenderName,
		},
		To:      []mail.Address{{Address: email}},
		Subject: subject,
		HTML:    body,
	}

	return app.NewMailClient().Send(message)
}

// SendOTPHandler handles OTP generation and sending
func SendOTPHandler(e *core.RequestEvent, app core.App) error {
	data := struct {
		Email   string `json:"email" form:"email"`
		UserID  string `json:"user_id" form:"user_id"`
		Purpose string `json:"purpose" form:"purpose"`
	}{}

	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Invalid request data", err)
	}

	// Validate required fields
	if data.Email == "" || data.UserID == "" || data.Purpose == "" {
		return apis.NewBadRequestError("Missing required fields", nil)
	}

	// Generate and store OTP
	otpCode, err := CreateOTP(app, data.UserID, data.Email, data.Purpose)
	if err != nil {
		return apis.NewInternalServerError("Failed to generate OTP", err)
	}

	// Send OTP via email
	if err := SendOTPEmail(app, data.Email, otpCode, data.Purpose); err != nil {
		return apis.NewInternalServerError("Failed to send OTP email", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"message": "OTP sent successfully",
	})
}

// VerifyOTPHandler handles OTP verification
func VerifyOTPHandler(e *core.RequestEvent, app core.App) error {
	data := struct {
		UserID  string `json:"user_id" form:"user_id"`
		OTPCode string `json:"otp_code" form:"otp_code"`
		Purpose string `json:"purpose" form:"purpose"`
	}{}

	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Invalid request data", err)
	}

	// Validate required fields
	if data.UserID == "" || data.OTPCode == "" || data.Purpose == "" {
		return apis.NewBadRequestError("Missing required fields", nil)
	}

	// Verify OTP
	if err := VerifyOTP(app, data.UserID, data.OTPCode, data.Purpose); err != nil {
		return apis.NewBadRequestError("Invalid or expired OTP", err)
	}

	// If this is signup verification, mark the user as verified
	if data.Purpose == "signup_verification" {
		userRecord, err := app.FindRecordById("users", data.UserID)
		if err != nil {
			return apis.NewInternalServerError("User not found", err)
		}

		userRecord.Set("verified", true)
		if err := app.Save(userRecord); err != nil {
			return apis.NewInternalServerError("Failed to verify user", err)
		}
	}

	return e.JSON(http.StatusOK, map[string]any{
		"message": "OTP verified successfully",
	})
}