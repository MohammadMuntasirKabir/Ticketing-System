package handler

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/api/helper"
	"github.com/you/airline-ticketing/internal/domain/user"
	"github.com/you/airline-ticketing/internal/pkg/contextkeys"
	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

type UserHandler struct {
	svc *user.Service
	log *zap.Logger
}

func NewUserHandler(svc *user.Service, log *zap.Logger) *UserHandler {
	return &UserHandler{svc: svc, log: log}
}

type credentialsRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type registerRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type profileRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Phone     *string `json:"phone"`
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in registerRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	pair, err := h.svc.Register(r.Context(), in.Email, in.Password, in.FirstName, in.LastName)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusCreated, toTokenResponse(pair))
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in credentialsRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	pair, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, toTokenResponse(pair))
}

func (h *UserHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var in refreshRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	pair, err := h.svc.RefreshToken(r.Context(), in.RefreshToken)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, toTokenResponse(pair))
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := contextkeys.UserIDFromContext(r.Context())
	if !ok {
		helper.Error(w, domainErrors.Unauthorized("authentication required"))
		return
	}
	u, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, toUserResponse(u))
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := contextkeys.UserIDFromContext(r.Context())
	if !ok {
		helper.Error(w, domainErrors.Unauthorized("authentication required"))
		return
	}
	var in profileRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	if err := h.svc.UpdateProfile(r.Context(), userID, user.ProfileUpdate{
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Phone:     in.Phone,
	}); err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

type userResponse struct {
	ID        int64   `json:"id"`
	Email     string  `json:"email"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Phone     *string `json:"phone,omitempty"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
}

func toTokenResponse(pair *user.TokenPair) tokenResponse {
	return tokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}
}

func toUserResponse(u *user.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Phone:     u.Phone,
		Role:      string(u.Role),
		Status:    u.Status,
	}
}
