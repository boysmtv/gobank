package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/yourorg/gobank/internal/delivery/http/middleware"
	"github.com/yourorg/gobank/internal/domain"
	txUC "github.com/yourorg/gobank/internal/usecase/transaction"
	"github.com/yourorg/gobank/pkg/response"
)

type TransactionHandler struct {
	uc       *txUC.UseCase
	validate *validator.Validate
}

func NewTransactionHandler(uc *txUC.UseCase) *TransactionHandler {
	return &TransactionHandler{uc: uc, validate: validator.New()}
}

func (h *TransactionHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := middleware.UserIDFromContext(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user")
		return
	}

	var req domain.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.JSONError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	txn, err := h.uc.Transfer(r.Context(), userID, req, realIP(r))
	if err != nil {
		switch err {
		case txUC.ErrInsufficientFunds:
			response.JSONError(w, http.StatusUnprocessableEntity, "INSUFFICIENT_FUNDS", err.Error())
		case txUC.ErrAccountNotActive:
			response.JSONError(w, http.StatusUnprocessableEntity, "ACCOUNT_NOT_ACTIVE", err.Error())
		case txUC.ErrCurrencyMismatch:
			response.JSONError(w, http.StatusUnprocessableEntity, "CURRENCY_MISMATCH", err.Error())
		case txUC.ErrSameAccount:
			response.JSONError(w, http.StatusUnprocessableEntity, "SAME_ACCOUNT", err.Error())
		case txUC.ErrUnauthorized:
			response.JSONError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			response.JSONError(w, http.StatusInternalServerError, "INTERNAL", "transfer failed")
		}
		return
	}

	response.JSON(w, http.StatusCreated, txn)
}

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := middleware.UserIDFromContext(r.Context())
	userID, _ := uuid.Parse(userIDStr)

	txnID, err := uuid.Parse(chi.URLParam(r, "txnID"))
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_ID", "invalid transaction ID")
		return
	}

	txn, err := h.uc.GetTransaction(r.Context(), userID, txnID)
	if err != nil {
		response.JSONError(w, http.StatusNotFound, "NOT_FOUND", "transaction not found")
		return
	}

	response.JSON(w, http.StatusOK, txn)
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := middleware.UserIDFromContext(r.Context())
	userID, _ := uuid.Parse(userIDStr)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := domain.TransactionListRequest{
		AccountID: r.URL.Query().Get("account_id"),
		Page:      page,
		PageSize:  pageSize,
	}
	if err := h.validate.Struct(req); err != nil {
		response.JSONError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	txns, total, err := h.uc.ListTransactions(r.Context(), userID, req)
	if err != nil {
		if err == txUC.ErrForbidden {
			response.JSONError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
			return
		}
		response.JSONError(w, http.StatusInternalServerError, "INTERNAL", "failed to list transactions")
		return
	}

	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}

	response.JSONPaginated(w, http.StatusOK, txns, response.Meta{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	})
}
