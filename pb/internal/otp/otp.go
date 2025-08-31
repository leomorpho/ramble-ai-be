package otp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/mail"
	"os"
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

// SendOTPEmail sends an OTP via email using appropriate method based on environment
func SendOTPEmail(app core.App, email, otpCode, purpose string) error {
	isDevelopment := os.Getenv("DEVELOPMENT") == "true"
	
	if isDevelopment {
		// Development: Use PocketBase's built-in SMTP (Mailpit)
		return sendOTPEmailSMTP(app, email, otpCode, purpose)
	} else {
		// Production: Use Resend HTTP API
		return sendOTPEmailResend(app, email, otpCode, purpose)
	}
}

// sendOTPEmailSMTP sends OTP via SMTP (development with Mailpit)
func sendOTPEmailSMTP(app core.App, email, otpCode, purpose string) error {
	subject, body := getOTPEmailContent(otpCode, purpose)
	
	message := &mailer.Message{
		From: mail.Address{
			Address: app.Settings().Meta.SenderAddress,
			Name:    app.Settings().Meta.SenderName,
		},
		To:      []mail.Address{{Address: email}},
		Subject: subject,
		HTML:    body,
	}

	log.Printf("[OTP] Sending email via SMTP to %s for purpose: %s", email, purpose)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- app.NewMailClient().Send(message)
	}()
	
	select {
	case err := <-done:
		if err != nil {
			log.Printf("[OTP] SMTP email sending failed: %v", err)
			return fmt.Errorf("failed to send email via SMTP: %v", err)
		}
		log.Printf("[OTP] SMTP email sent successfully to %s", email)
		return nil
	case <-ctx.Done():
		log.Printf("[OTP] SMTP email sending timed out after 30 seconds for %s", email)
		return fmt.Errorf("SMTP email sending timed out")
	}
}

// sendOTPEmailResend sends OTP via Resend HTTP API (production)
func sendOTPEmailResend(app core.App, email, otpCode, purpose string) error {
	resendAPIKey := os.Getenv("RESEND_API_KEY")
	if resendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not configured")
	}
	
	subject, body := getOTPEmailContent(otpCode, purpose)
	
	// Resend API payload
	payload := map[string]interface{}{
		"from":    fmt.Sprintf("%s <%s>", app.Settings().Meta.SenderName, app.Settings().Meta.SenderAddress),
		"to":      []string{email},
		"subject": subject,
		"html":    body,
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %v", err)
	}
	
	log.Printf("[OTP] Sending email via Resend API to %s for purpose: %s", email, purpose)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+resendAPIKey)
	
	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[OTP] Resend API request failed: %v", err)
		return fmt.Errorf("failed to send email via Resend: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		// Read response body for debugging
		var respBody bytes.Buffer
		respBody.ReadFrom(resp.Body)
		log.Printf("[OTP] Resend API error - Status: %d, Body: %s", resp.StatusCode, respBody.String())
		return fmt.Errorf("Resend API returned status %d", resp.StatusCode)
	}
	
	log.Printf("[OTP] Resend email sent successfully to %s", email)
	return nil
}

// getOTPEmailContent returns subject and HTML body for OTP emails
func getOTPEmailContent(otpCode, purpose string) (string, string) {
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
		subject = "Verification Code"
		body = fmt.Sprintf(`
		<h2>Verification Code</h2>
		<p>Please enter this verification code:</p>
		<div style="background: #f8f9fa; padding: 20px; margin: 20px 0; text-align: center; border-radius: 8px;">
			<h1 style="font-size: 32px; color: #007bff; margin: 0; letter-spacing: 8px;">%s</h1>
		</div>
		<p>This code will expire in 10 minutes.</p>
		`, otpCode)
	}
	
	return subject, body
}

// SendOTPHandler handles OTP generation and sending
func SendOTPHandler(e *core.RequestEvent, app core.App) error {
	// Set CORS headers - restrict to your frontend domain in production
	origin := os.Getenv("FRONTEND_URL")
	if origin == "" {
		origin = "*" // fallback for development
	}
	e.Response.Header().Set("Access-Control-Allow-Origin", origin)
	e.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	e.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	e.Response.Header().Set("Access-Control-Allow-Credentials", "true")
	
	// Handle preflight OPTIONS requests
	if e.Request.Method == "OPTIONS" {
		e.Response.WriteHeader(204)
		return nil
	}

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
	log.Printf("[OTP] Attempting to send OTP email to %s (UserID: %s, Purpose: %s)", data.Email, data.UserID, data.Purpose)
	if err := SendOTPEmail(app, data.Email, otpCode, data.Purpose); err != nil {
		log.Printf("[OTP] Failed to send OTP email to %s: %v", data.Email, err)
		return apis.NewInternalServerError("Failed to send OTP email", err)
	}
	log.Printf("[OTP] OTP email sent successfully to %s", data.Email)

	return e.JSON(http.StatusOK, map[string]any{
		"message": "OTP sent successfully",
	})
}

// VerifyOTPHandler handles OTP verification
func VerifyOTPHandler(e *core.RequestEvent, app core.App) error {
	// Set CORS headers - restrict to your frontend domain in production
	origin := os.Getenv("FRONTEND_URL")
	if origin == "" {
		origin = "*" // fallback for development
	}
	e.Response.Header().Set("Access-Control-Allow-Origin", origin)
	e.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	e.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	e.Response.Header().Set("Access-Control-Allow-Credentials", "true")
	
	// Handle preflight OPTIONS requests
	if e.Request.Method == "OPTIONS" {
		e.Response.WriteHeader(204)
		return nil
	}

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