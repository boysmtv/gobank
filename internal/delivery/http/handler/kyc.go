package handler

import (
	"encoding/json"
	"net/http"

	kycuc "gobank/internal/usecase/kyc"
	"gobank/pkg/response"
)

type KYCHandler struct {
	usecase *kycuc.UseCase
}

func NewKYCHandler(usecase *kycuc.UseCase) *KYCHandler {
	return &KYCHandler{usecase: usecase}
}

func (h *KYCHandler) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID         string `json:"user_id"`
		DocumentType   string `json:"document_type"`
		DocumentNumber string `json:"document_number"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.usecase.Submit(r.Context(), kycuc.SubmitInput{
		UserID:         req.UserID,
		DocumentType:   req.DocumentType,
		DocumentNumber: req.DocumentNumber,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, response.Envelope{
		"success": true,
		"data":    result,
	})
}
