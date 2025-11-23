package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/astaxie/beego"
)

// CSRF Token Store
type CSRFTokenStore struct {
	tokens sync.Map // map[string]time.Time
}

var csrfStore = &CSRFTokenStore{}

// GenerateCSRFToken generates a new CSRF token
func (s *BaseController) GenerateCSRFToken() string {
	// Generate random token
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)

	// Store token with expiration
	csrfStore.tokens.Store(token, time.Now().Add(24*time.Hour))

	// Set token in session and cookie
	s.SetSession("csrf_token", token)
	s.Ctx.SetCookie("csrf_token", token, 86400, "/", "", false, true)

	return token
}

// ValidateCSRFToken validates the CSRF token
func (s *BaseController) ValidateCSRFToken() bool {
	// Get token from request (header or form)
	tokenFromHeader := s.Ctx.Input.Header("X-CSRF-Token")
	tokenFromForm := s.GetString("csrf_token")

	token := tokenFromHeader
	if token == "" {
		token = tokenFromForm
	}

	if token == "" {
		return false
	}

	// Get token from session
	sessionToken := s.GetSession("csrf_token")
	if sessionToken == nil {
		return false
	}

	// Validate token matches
	if token != sessionToken.(string) {
		return false
	}

	// Check if token exists and not expired
	if expiryTime, ok := csrfStore.tokens.Load(token); ok {
		if time.Now().Before(expiryTime.(time.Time)) {
			return true
		}
		// Token expired, remove it
		csrfStore.tokens.Delete(token)
	}

	return false
}

// RequireCSRF is a middleware to enforce CSRF protection
func (s *BaseController) RequireCSRF() {
	// Skip CSRF for GET, HEAD, OPTIONS
	method := s.Ctx.Request.Method
	if method == "GET" || method == "HEAD" || method == "OPTIONS" {
		// Generate token for future requests
		if s.GetSession("csrf_token") == nil {
			s.GenerateCSRFToken()
		}
		return
	}

	// Validate CSRF for POST, PUT, DELETE, PATCH
	if !s.ValidateCSRFToken() {
		s.Ctx.Output.SetStatus(403)
		s.Data["json"] = map[string]interface{}{
			"status": 0,
			"msg":    "CSRF token validation failed",
		}
		s.ServeJSON()
		s.StopRun()
	}
}

// CleanupExpiredTokens removes expired CSRF tokens (run periodically)
func CleanupExpiredCSRFTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		csrfStore.tokens.Range(func(key, value interface{}) bool {
			if time.Now().After(value.(time.Time)) {
				csrfStore.tokens.Delete(key)
			}
			return true
		})
	}
}

func init() {
	// Start cleanup goroutine
	go CleanupExpiredCSRFTokens()
}
