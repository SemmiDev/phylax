package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/semmidev/phylax/internal/infrastructure/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

// OAuthService defines the interface for OAuth-related operations.
type OAuthService interface {
	GetConfig() *oauth2.Config
	StartAuthServer(ctx context.Context, addr string) error
	Shutdown(ctx context.Context) error
}

// GoogleOAuthService handles Google OAuth configuration and server.
type GoogleOAuthService struct {
	config     *oauth2.Config
	logger     *logger.Logger
	authServer *http.Server
}

// NewGoogleOAuthService creates a new GoogleOAuthService.
func NewGoogleOAuthService(logger *logger.Logger, clientSecretPath string) (*GoogleOAuthService, error) {
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	if clientSecretPath == "" {
		return nil, errors.New("client secret path cannot be empty")
	}

	b, err := os.ReadFile(clientSecretPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client_secret.json: %w", err)
	}

	cfg, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret: %w", err)
	}

	return &GoogleOAuthService{
		config: cfg,
		logger: logger,
	}, nil
}

// GetConfig returns the OAuth2 configuration.
func (s *GoogleOAuthService) GetConfig() *oauth2.Config {
	return s.config
}

// StartAuthServer starts the OAuth HTTP server in a goroutine.
func (s *GoogleOAuthService) StartAuthServer(ctx context.Context, addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /auth/google/drive", func(w http.ResponseWriter, r *http.Request) {
		authURL := s.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	})

	mux.HandleFunc("GET /auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code parameter", http.StatusBadRequest)
			return
		}

		token, err := s.config.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, fmt.Sprintf("token exchange failed: %v", err), http.StatusInternalServerError)
			return
		}

		tokenJSON, err := json.MarshalIndent(token, "", "  ")
		if err != nil {
			http.Error(w, "failed to marshal token", http.StatusInternalServerError)
			return
		}

		refresh := token.RefreshToken
		if refresh == "" {
			fmt.Fprintln(w, "⚠️ No refresh token returned. Revoke app access & re-authorize.")
			return
		}

		fmt.Fprintf(w, "✅ Refresh Token:\n%s\n\nFull Token JSON:\n%s", refresh, tokenJSON)
	})

	s.authServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		s.logger.Infof("Google Drive OAuth server listening on %s", s.authServer.Addr)
		if err := s.authServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Errorf("OAuth server error: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully stops the OAuth server.
func (s *GoogleOAuthService) Shutdown(ctx context.Context) error {
	if s.authServer == nil {
		return nil
	}

	if err := s.authServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown OAuth server: %w", err)
	}
	s.logger.Infof("OAuth server stopped successfully")
	return nil
}
