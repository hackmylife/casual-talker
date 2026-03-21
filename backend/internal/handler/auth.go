package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/naoki-watanabe/casual-talker/backend/internal/middleware"
	"github.com/naoki-watanabe/casual-talker/backend/internal/service"
)

// validate is a package-level validator instance shared across handlers.
var validate = validator.New()

// AuthHandler exposes HTTP endpoints for user authentication.
type AuthHandler struct {
	service *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

// --- request / response types ---

type registerRequest struct {
	Email       string  `json:"email"        validate:"required,email"`
	Password    string  `json:"password"     validate:"required,min=8,max=72"`
	DisplayName *string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type userResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName *string `json:"display_name"`
	Level       int     `json:"level"`
}

type authResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         userResponse `json:"user"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeAndValidate(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return errors.New("invalid request body")
	}
	return validate.Struct(dst)
}

// --- handlers ---

// Register handles POST /api/v1/auth/register.
// It creates a new user account for whitelisted email addresses.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	accessToken, refreshToken, user, err := h.service.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailNotAllowed):
			writeError(w, http.StatusForbidden, "email not in whitelist")
		case errors.Is(err, service.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email already registered")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: userResponse{
			ID:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Level:       user.Level,
		},
	})
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	accessToken, refreshToken, user, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: userResponse{
			ID:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Level:       user.Level,
		},
	})
}

// Refresh handles POST /api/v1/auth/refresh.
// It exchanges a valid refresh token for a new access token and rotated refresh token.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	newAccessToken, newRefreshToken, err := h.service.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	})
}

// Logout handles POST /api/v1/auth/logout.
// It revokes the provided refresh token.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Best-effort: silently ignore unknown tokens to avoid leaking information.
	_ = h.service.Logout(r.Context(), req.RefreshToken)

	w.WriteHeader(http.StatusNoContent)
}

// Me handles GET /api/v1/users/me.
// It requires a valid JWT access token (enforced by the auth middleware).
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Level:       user.Level,
	})
}
