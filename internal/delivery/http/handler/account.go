package handler

import (
	"encoding/json"
	"net/http"

	"gobank/internal/domain"
	accountuc "gobank/internal/usecase/account"
	"gobank/pkg/response"
)

type AccountHandler struct {
	usecase *accountuc.UseCase
}

func NewAccountHandler(usecase *accountuc.UseCase) *AccountHandler {
	return &AccountHandler{usecase: usecase}
}

func (h *AccountHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		h.list(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *AccountHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		Type     string `json:"type"`
		Currency string `json:"currency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.usecase.Create(r.Context(), accountuc.CreateInput{
		UserID:   req.UserID,
		Type:     domain.AccountType(req.Type),
		Currency: req.Currency,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, response.Envelope{
		"success": true,
		"data":    account,
	})
}

func (h *AccountHandler) list(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	accounts, err := h.usecase.ListByUser(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, response.Envelope{
		"success": true,
		"data":    accounts,
	})
}
