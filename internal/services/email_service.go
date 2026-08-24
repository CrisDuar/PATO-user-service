package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const emailSendMaxRetries = 3

type EmailService struct {
	baseURL string
	client  *http.Client
}

func NewEmailService(baseURL string) *EmailService {
	return &EmailService{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type sendEmailTokenRequest struct {
	User         string `json:"user"`
	Token        string `json:"token"`
	Minutos      string `json:"minutos"`
	Destinatario string `json:"destinatario"`
}

func (s *EmailService) SendPasswordResetEmail(
	username string,
	email string,
	token string,
) error {

	payload := sendEmailTokenRequest{
		User:         username,
		Token:        token,
		Minutos:      "15",
		Destinatario: email,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf(
			"failed to marshal password reset email payload: %w",
			err,
		)
	}

	var lastErr error

	for attempt := 1; attempt <= emailSendMaxRetries; attempt++ {
		lastErr = s.sendPasswordReset(body)

		if lastErr == nil {
			log.Printf(
				"password reset email sent to %s (attempt %d)",
				email,
				attempt,
			)
			return nil
		}

		log.Printf(
			"password reset email attempt %d/%d to %s failed: %v",
			attempt,
			emailSendMaxRetries,
			email,
			lastErr,
		)

		if attempt < emailSendMaxRetries {
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf(
		"failed to send password reset email after %d attempts: %w",
		emailSendMaxRetries,
		lastErr,
	)
}
func (s *EmailService) sendPasswordReset(body []byte) error {
	req, err := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/api/sendEmailToken",
		bytes.NewReader(body),
	)

	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("email service request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf(
			"email service returned status %d",
			resp.StatusCode,
		)
	}

	return nil
}

func (s *EmailService) SendVerificationEmail(username, email, token string) error {
	payload := sendEmailTokenRequest{
		User:         username,
		Token:        token,
		Minutos:      "15",
		Destinatario: email,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal verification email payload: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= emailSendMaxRetries; attempt++ {
		lastErr = s.send(body)
		if lastErr == nil {
			log.Printf("verification email sent to %s (attempt %d)", email, attempt)
			return nil
		}

		log.Printf("verification email attempt %d/%d to %s failed: %v", attempt, emailSendMaxRetries, email, lastErr)
		if attempt < emailSendMaxRetries {
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("failed to send verification email after %d attempts: %w", emailSendMaxRetries, lastErr)
}

func (s *EmailService) send(body []byte) error {
	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/api/sendEmailToken", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("email service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("email service returned status %d", resp.StatusCode)
	}

	return nil
}
