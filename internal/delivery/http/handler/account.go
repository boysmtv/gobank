package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/yourorg/gobank/internal/delivery/http/middleware"
	"github.com/yourorg/gobank/internal/domain"
	accountUC "github.com/yourorg/gobank/internal/usecase/account"
	"github.com/yourorg/gobank/pkg/response"
)

type AccountHandler struct {
	uc       *accountUC.UseCase
	validate *validator.Validate
}

func NewAccountHandler(uc *accountUC.UseCase) *AccountHandler {
	return &AccountHandler{uc: uc, validate: validator.New()}
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := middleware.UserIDFromContext(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user")
		return
	}

	var req domain.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.JSONError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	acct, err := h.uc.CreateAccount(r.Context(), userID, req)
	if err != nil {
		if err == accountUC.ErrMaxAccountsReached {
			response.JSONError(w, http.StatusConflict, "MAX_ACCOUNTS", err.Error())
			return
		}
		response.JSONError(w, http.StatusInternalServerError, "INTERNAL", "failed to create account")
		return
	}

	response.JSON(w, http.StatusCreated, acct)
}

func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := middleware.UserIDFromContext(r.Context())
	userID, _ := uuid.Parse(userIDStr)
	accountID, err := uuid.Parse(chi.URLParam(r, "accountID"))
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "INVALID_ID", "invalid account ID")
		return
	}

	acct, err := h.uc.GetAccount(r.Context(), userID, accountID)
	if err != nil {
		if err == accountUC.ErrForbidden {
			response.JSONError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
			return
		}
		response.JSONError(w, http.StatusNotFound, "NOT_FOUND", "account not found")
		return
	}

	response.JSON(w, http.StatusOK, acct)
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := middleware.UserIDFromContext(r.Context())
	userID, _ := uuid.Parse(userIDStr)

	accounts, err := h.uc.ListAccounts(r.Context(), userID)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, "INTERNAL", "failed to list accounts")
		return
	}

	response.JSON(w, http.StatusOK, accounts)
}
