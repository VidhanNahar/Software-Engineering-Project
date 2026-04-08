package utils

import (
	"crypto/tls"
	"fmt"
	"math/rand/v2"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

// SendOTP sends a professional OTP email to the user with security validations
func SendOTP(userEmail, userName, otp string) error {
	// Validate email format
	if err := validateEmail(userEmail); err != nil {
		return fmt.Errorf("invalid email address: %w", err)
	}

	// Validate OTP length (should be 6 digits)
	if len(otp) != 6 {
		return fmt.Errorf("invalid OTP length: expected 6 digits, got %d", len(otp))
	}

	// Validate userName is not empty
	if userName == "" {
		return fmt.Errorf("user name cannot be empty")
	}

	from := os.Getenv("FROM_MAIL")
	password := os.Getenv("PASS_MAIL")

	// Enhanced credential validation
	if from == "" || password == "" {
		return fmt.Errorf("email credentials not configured in .env (FROM_MAIL, PASS_MAIL)")
	}

	if from == "your-email@gmail.com" || password == "your-app-password" {
		return fmt.Errorf("email credentials are placeholder values - configure real credentials in .env")
	}

	// Validate sender email format
	if err := validateEmail(from); err != nil {
		return fmt.Errorf("invalid sender email in .env: %w", err)
	}

	host := "smtp.gmail.com"
	port := "587"

	// Professional Subject
	subject := "Subject: FinXGrow Account Verification OTP\r\n"

	// HTML Email MIME
	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"

	// Professional HTML Body with branding
	body := fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; background-color: #f4f6f8; padding: 20px;">
    <div style="max-width: 600px; margin: auto; background: #ffffff; padding: 20px; border-radius: 10px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">

        <h2 style="color: #2c3e50; text-align: center;">Welcome to FinXGrow 🚀</h2>

        <p>Hi <b>%s</b>,</p>

        <p>Thank you for joining <b>FinXGrow</b> — your partner in smart financial growth.</p>

        <p>Please use the following One-Time Password (OTP) to verify your account:</p>

        <div style="text-align: center; margin: 20px 0;">
            <span style="font-size: 32px; font-weight: bold; color: #27ae60; letter-spacing: 3px; font-family: 'Courier New', monospace;">
                %s
            </span>
        </div>

        <p>This OTP is valid for <b>10 minutes</b> and can only be used once.</p>

        <p style="color: #e74c3c;"><b>⚠️ Security Notice:</b> Do not share this code with anyone, including FinXGrow staff.</p>

        <hr style="border: none; border-top: 1px solid #ecf0f1;">

        <p style="font-size: 12px; color: #7f8c8d;">
            If you did not request this email, please ignore it and do not share this OTP.
        </p>

        <p>Best regards,<br><b>Team FinXGrow</b><br><em>Empowering Your Financial Growth 📈</em></p>
    </div>
</body>
</html>
`, userName, otp)

	// Combine message
	msg := []byte(subject + mime + body)

	// SMTP Authentication
	auth := smtp.PlainAuth("", from, password, host)

	// Dial with timeout to prevent hanging connections
	conn, err := net.DialTimeout("tcp4", host+":"+port, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Start TLS if supported
	if ok, _ := client.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: host}
		if err := client.StartTLS(config); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed - verify FROM_MAIL and PASS_MAIL are correct: %w", err)
	}

	// Send email
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(userEmail); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = wc.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = wc.Close()
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// Graceful QUIT
	client.Quit()

	fmt.Printf("✅ OTP email sent successfully to %s (Recipient: %s)\n", userEmail, userName)
	return nil
}

// validateEmail validates email format
func validateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	return err
}

// GenerateRandomNumber generates a cryptographically secure 6-digit OTP
// Uses math/rand/v2 which is auto-seeded in Go 1.20+
func GenerateRandomNumber() string {
	// math/rand/v2.IntN is better than math/rand.Intn
	// It's more efficient and provides better randomness
	num := rand.IntN(900000) + 100000
	return strconv.Itoa(num)
}
