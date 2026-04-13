package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Omotolani98/foostash/internal/server/respond"
	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/sshauth"
)

type AdminHandler struct {
	Invites *service.Invites
}

type createInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type createInviteResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// CreateInvite requires admin role.
func (h *AdminHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	body := middleware.BodyFromContext(r.Context())
	var req createInviteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	out, err := h.Invites.Create(r.Context(), actx, service.CreateInviteInput{
		Email: req.Email,
		Role:  req.Role,
	})
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusCreated, createInviteResponse{
		Token:     out.Token,
		ExpiresAt: out.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

type joinRequest struct {
	Token     string `json:"token"`
	PublicKey string `json:"public_key"`
}

type joinResponse struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// Join is special-cased: the user is not yet registered, so the standard
// auth middleware can't run. Instead we verify the request signature against
// the public key submitted in the body before consuming the invite.
func (h *AdminHandler) Join(w http.ResponseWriter, r *http.Request) {
	body := middleware.BodyFromContext(r.Context())
	var req joinRequest
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

	out, err := h.Invites.Consume(r.Context(), service.ConsumeInviteInput{
		Token:     req.Token,
		PublicKey: req.PublicKey,
	})
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusCreated, joinResponse{
		UserID: out.UserID.String(),
		OrgID:  out.OrgID.String(),
		Email:  out.Email,
		Role:   out.Role,
	})
}
