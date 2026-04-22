package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/yourorg/gobank/internal/delivery/http/middleware"
	"github.com/yourorg/gobank/internal/domain"
	kycUC "github.com/yourorg/gobank/internal/usecase/kyc"
	"github.com/yourorg/gobank/pkg/response"
)

type KYCHandler struct {
	uc       *kycUC.UseCase
	validate *validator.Validate
}

func NewKYCHandler(uc *kycUC.UseCase) *KYCHandler {
	return &KYCHandler{uc: uc, validate: validator.New()}
}

func (h *KYCHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := middleware.UserIDFromContext(r.Context())
	userID, _ := uuid.Parse(userIDStr)

	var req domain.KYCSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.JSONError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	record, err := h.uc.Submit(r.Context(), userID, req)
	if err != nil {
		if err == kycUC.ErrAlreadyVerified {
			response.JSONError(w, http.StatusConflict, "ALREADY_VERIFIED", err.Error())
			return
		}
		response.JSONError(w, http.StatusInternalServerError, "INTERNAL", "KYC submission failed")
		return
	}

	response.JSON(w, http.StatusCreated, record)
}

func (h *KYCHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := middleware.UserIDFromContext(r.Context())
	userID, _ := uuid.Parse(userIDStr)

	record, err := h.uc.GetStatus(r.Context(), userID)
	if err != nil {
		response.JSONError(w, http.StatusNotFound, "NOT_FOUND", "KYC record not found")
		return
	}

	response.JSON(w, http.StatusOK, record)
}

func (h *KYCHandler) AdminVerify(w http.ResponseWriter, r *http.Request) {
	adminIDStr, _ := middleware.UserIDFromContext(r.Context())
	adminID, _ := uuid.Parse(adminIDStr)

	var body struct {
		UserID string `json:"user_id" validate:"required,uuid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_BODY", "invalid body")
		return
	}
	targetID, _ := uuid.Parse(body.UserID)

	if err := h.uc.Verify(r.Context(), adminID, targetID); err != nil {
		response.JSONError(w, http.StatusInternalServerError, "INTERNAL", "verification failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "verified"})
}
