package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/yourorg/gobank/internal/delivery/http/middleware"
	"github.com/yourorg/gobank/internal/domain"
	authUC "github.com/yourorg/gobank/internal/usecase/auth"
	"github.com/yourorg/gobank/pkg/response"
)

type AuthHandler struct {
	uc       *authUC.UseCase
	validate *validator.Validate
}

func NewAuthHandler(uc *authUC.UseCase) *AuthHandler {
	return &AuthHandler{uc: uc, validate: validator.New()}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.JSONError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	user, err := h.uc.Register(r.Context(), req, realIP(r))
	if err != nil {
		switch err {
		case authUC.ErrEmailTaken:
			response.JSONError(w, http.StatusConflict, "EMAIL_TAKEN", err.Error())
		default:
			response.JSONError(w, http.StatusInternalServerError, "INTERNAL", "registration failed")
		}
		return
	}

	response.JSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.JSONError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	tokens, err := h.uc.Login(r.Context(), req, realIP(r), r.UserAgent())
	if err != nil {
		switch err {
		case authUC.ErrInvalidCredentials:
			response.JSONError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		case authUC.ErrAccountSuspended:
			response.JSONError(w, http.StatusForbidden, "ACCOUNT_SUSPENDED", "account is suspended")
		default:
			response.JSONError(w, http.StatusInternalServerError, "INTERNAL", "login failed")
		}
		return
	}

	response.JSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if err := h.validate.Struct(body); err != nil {
		response.JSONError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	tokens, err := h.uc.RefreshTokens(r.Context(), body.RefreshToken)
	if err != nil {
		response.JSONError(w, http.StatusUnauthorized, "INVALID_TOKEN", "token is invalid or expired")
		return
	}

	response.JSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_ID", "invalid user ID")
		return
	}

	if err := h.uc.Logout(r.Context(), userID, realIP(r)); err != nil {
		response.JSONError(w, http.StatusInternalServerError, "INTERNAL", "logout failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
