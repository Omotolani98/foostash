// Package handlers houses HTTP handlers for the foostash server.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/server/respond"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/sshauth"
)

type AuthHandler struct {
	Auth *service.Auth
}

type registerRequest struct {
	Email     string `json:"email"`
	OrgName   string `json:"org_name"`
	PublicKey string `json:"public_key"`
}

type registerResponse struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Role   string `json:"role"`
}

// Register is open: no auth middleware. Anyone with a valid SSH keypair can
// create a new org, but they must prove possession of the private key: like
// Join, we verify the request signature against the public key in the body
// before creating anything.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	body := middleware.BodyFromContext(r.Context())
	var req registerRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.PublicKey == "" {
		respond.WriteError(w, http.StatusBadRequest, "invalid_argument", "public_key required")
		return
	}
	pub, err := sshauth.ParseAuthorizedKey(req.PublicKey)
	if err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_argument", "bad public key")
		return
	}
	timestamp := r.Header.Get(sshauth.HeaderTimestamp)
	sig := r.Header.Get(sshauth.HeaderSignature)
	if timestamp == "" || sig == "" {
		respond.WriteError(w, http.StatusUnauthorized, "missing_signature", "signature headers missing")
		return
	}
	if err := sshauth.Verify(pub, r.Method, r.URL.Path, timestamp, sig, body); err != nil {
		respond.WriteServiceError(w, err)
		return
	}

	out, err := h.Auth.Register(r.Context(), service.RegisterInput{
		Email:     req.Email,
		OrgName:   req.OrgName,
		PublicKey: req.PublicKey,
	})
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusCreated, registerResponse{
		UserID: out.UserID.String(),
		OrgID:  out.OrgID.String(),
		Role:   out.Role,
	})
}

type loginResponse struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// Login just echoes the resolved AuthContext (the middleware did the work).
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	if actx == nil {
		respond.WriteError(w, http.StatusUnauthorized, "unknown_key", "not authenticated")
		return
	}
	respond.WriteJSON(w, http.StatusOK, loginResponse{
		UserID: actx.UserID.String(),
		OrgID:  actx.OrgID.String(),
		Email:  actx.Email,
		Role:   actx.Role,
	})
}
