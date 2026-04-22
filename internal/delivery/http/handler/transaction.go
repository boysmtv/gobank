package handler

import (
	"encoding/json"
	"net/http"

	transactionuc "gobank/internal/usecase/transaction"
	"gobank/pkg/response"
)

type TransactionHandler struct {
	usecase *transactionuc.UseCase
}

func NewTransactionHandler(usecase *transactionuc.UseCase) *TransactionHandler {
	return &TransactionHandler{usecase: usecase}
}

func (h *TransactionHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ReferenceID   string `json:"reference_id"`
		SourceID      string `json:"source_id"`
		DestinationID string `json:"destination_id"`
		Amount        int64  `json:"amount"`
		Description   string `json:"description"`
		ActorID       string `json:"actor_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tx, err := h.usecase.Transfer(r.Context(), transactionuc.TransferInput{
		ReferenceID:   req.ReferenceID,
		SourceID:      req.SourceID,
		DestinationID: req.DestinationID,
		Amount:        req.Amount,
		Description:   req.Description,
		ActorID:       req.ActorID,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, response.Envelope{
		"success": true,
		"data":    tx,
	})
}

func (h *TransactionHandler) History(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	accountID := r.URL.Query().Get("account_id")
	items, err := h.usecase.History(r.Context(), accountID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, response.Envelope{
		"success": true,
		"data":    items,
	})
}
