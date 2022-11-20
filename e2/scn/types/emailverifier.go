package types

import "time"

// EmailVerifier is an email verification record.
type EmailVerifier struct {
	ID        int64      `json:"-" db:"id"`
	UUID      string     `json:"uuid" db:"uuid"`
	UserUUID  string     `json:"userUuid" db:"user_uuid"`
	Code      string     `json:"-" db:"code"`
	CreatedAt *time.Time `json:"createdAt" db:"created_at"`
	State     string     `json:"state" db:"state"`
}

// RequestVerifier is a verifier used in an HTTP request.
type RequestVerifier struct {
	UUID string `json:"uuid"`
	Code string `json:"code"`
}

// CreateEmailVerifierRequest is a request for an email verifier.
type CreateEmailVerifierRequest struct {
	Email string `json:"email"`
}

// CreateEmailVerifierResponse is a response to a CreateEmailVerifierRequest.
type CreateEmailVerifierResponse struct {
	Verifier EmailVerifier `json:"verifier"`
}
