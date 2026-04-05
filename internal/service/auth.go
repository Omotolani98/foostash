package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Omotolani98/foostash/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	APIKeyPrefix   = "fst_"
	APIKeyRawBytes = 32
	JWTExpiry      = 24 * time.Hour
)

type AuthService struct {
	users     *store.UsersStore
	orgs      *store.OrganizationsStore
	apiKeys   *store.APIKeysStore
	jwtSecret []byte
}

func NewAuthService(users *store.UsersStore, orgs *store.OrganizationsStore, apiKeys *store.APIKeysStore, jwtSecret []byte) *AuthService {
	return &AuthService{users: users, orgs: orgs, apiKeys: apiKeys, jwtSecret: jwtSecret}
}

type RegisterInput struct {
	OrgName  string
	Email    string
	Password string
}

type AuthResult struct {
	User  *store.UserRow
	Org   *store.OrganizationRow
	Token string
}

func (a *AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	if in.OrgName == "" || in.Email == "" || len(in.Password) < 8 {
		return nil, fmt.Errorf("%w: org_name, email, and password (min 8 chars) required", ErrValidation)
	}

	org, err := a.orgs.Create(ctx, in.OrgName)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := a.users.Create(ctx, org.ID, in.Email, string(hash), "admin")
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}

	token, err := a.issueJWT(user.ID, org.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{User: user, Org: org, Token: token}, nil
}

func (a *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := a.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !user.PasswordHash.Valid {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	org, err := a.orgs.GetByID(ctx, user.OrgID)
	if err != nil {
		return nil, err
	}

	token, err := a.issueJWT(user.ID, user.OrgID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{User: user, Org: org, Token: token}, nil
}

func (a *AuthService) issueJWT(userID, orgID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":    userID,
		"org_id": orgID,
		"exp":    time.Now().Add(JWTExpiry).Unix(),
		"iat":    time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

type JWTClaims struct {
	UserID string
	OrgID  string
}

func (a *AuthService) VerifyJWT(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrUnauthorized
	}
	sub, _ := claims["sub"].(string)
	orgID, _ := claims["org_id"].(string)
	if sub == "" || orgID == "" {
		return nil, ErrUnauthorized
	}
	return &JWTClaims{UserID: sub, OrgID: orgID}, nil
}

type CreatedAPIKey struct {
	Row    *store.APIKeyRow
	RawKey string
}

func (a *AuthService) CreateAPIKey(ctx context.Context, userID, orgID, name string, scopes []string) (*CreatedAPIKey, error) {
	raw := make([]byte, APIKeyRawBytes)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}
	rawKey := APIKeyPrefix + hex.EncodeToString(raw)
	hash := hashKey(rawKey)

	prefix := rawKey
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}

	if len(scopes) == 0 {
		scopes = []string{"secrets:read", "secrets:write"}
	}

	row, err := a.apiKeys.Create(ctx, userID, orgID, hash, prefix, name, scopes)
	if err != nil {
		return nil, err
	}
	return &CreatedAPIKey{Row: row, RawKey: rawKey}, nil
}

func (a *AuthService) ResolveAPIKey(ctx context.Context, rawKey string) (*store.APIKeyRow, error) {
	if !strings.HasPrefix(rawKey, APIKeyPrefix) {
		return nil, ErrUnauthorized
	}
	row, err := a.apiKeys.GetByHash(ctx, hashKey(rawKey))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	if row.ExpiresAt.Valid && time.Now().After(row.ExpiresAt.Time) {
		return nil, ErrUnauthorized
	}
	go a.apiKeys.TouchLastUsed(context.Background(), row.ID)
	return row, nil
}

func (a *AuthService) ListAPIKeys(ctx context.Context, userID string) ([]store.APIKeyRow, error) {
	return a.apiKeys.ListByUser(ctx, userID)
}

func (a *AuthService) RevokeAPIKey(ctx context.Context, userID, id string) error {
	if err := a.apiKeys.Revoke(ctx, userID, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func hashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
