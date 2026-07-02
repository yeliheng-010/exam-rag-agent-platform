package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"golang.org/x/crypto/bcrypt"
)

type passwordResetUserRepo struct {
	interfaces.UserRepository
	byEmail map[string]*types.User
	byID    map[string]*types.User
}

func newPasswordResetUserRepo(users ...*types.User) *passwordResetUserRepo {
	r := &passwordResetUserRepo{
		byEmail: map[string]*types.User{},
		byID:    map[string]*types.User{},
	}
	for _, user := range users {
		cp := *user
		r.byEmail[strings.ToLower(user.Email)] = &cp
		r.byID[user.ID] = &cp
	}
	return r
}

func (r *passwordResetUserRepo) GetUserByEmail(_ context.Context, email string) (*types.User, error) {
	user := r.byEmail[strings.ToLower(email)]
	if user == nil {
		return nil, apprepo.ErrUserNotFound
	}
	cp := *user
	return &cp, nil
}

func (r *passwordResetUserRepo) GetUserByID(_ context.Context, id string) (*types.User, error) {
	user := r.byID[id]
	if user == nil {
		return nil, apprepo.ErrUserNotFound
	}
	cp := *user
	return &cp, nil
}

func (r *passwordResetUserRepo) UpdateUser(_ context.Context, user *types.User) error {
	if _, ok := r.byID[user.ID]; !ok {
		return apprepo.ErrUserNotFound
	}
	cp := *user
	r.byID[user.ID] = &cp
	r.byEmail[strings.ToLower(user.Email)] = &cp
	return nil
}

type passwordResetTokenRepo struct {
	interfaces.AuthTokenRepository
	byValue        map[string]*types.AuthToken
	created        []*types.AuthToken
	revokeAllCalls []string
}

func newPasswordResetTokenRepo(tokens ...*types.AuthToken) *passwordResetTokenRepo {
	r := &passwordResetTokenRepo{byValue: map[string]*types.AuthToken{}}
	for _, token := range tokens {
		cp := *token
		r.byValue[token.Token] = &cp
	}
	return r
}

func (r *passwordResetTokenRepo) CreateToken(_ context.Context, token *types.AuthToken) error {
	cp := *token
	r.created = append(r.created, &cp)
	r.byValue[token.Token] = &cp
	return nil
}

func (r *passwordResetTokenRepo) GetTokenByValue(_ context.Context, value string) (*types.AuthToken, error) {
	token := r.byValue[value]
	if token == nil {
		return nil, apprepo.ErrTokenNotFound
	}
	cp := *token
	return &cp, nil
}

func (r *passwordResetTokenRepo) UpdateToken(_ context.Context, token *types.AuthToken) error {
	if _, ok := r.byValue[token.Token]; !ok {
		return apprepo.ErrTokenNotFound
	}
	cp := *token
	r.byValue[token.Token] = &cp
	return nil
}

func (r *passwordResetTokenRepo) RevokeTokensByUserID(_ context.Context, userID string) error {
	r.revokeAllCalls = append(r.revokeAllCalls, userID)
	for _, token := range r.byValue {
		if token.UserID == userID {
			token.IsRevoked = true
			token.UpdatedAt = time.Now()
		}
	}
	return nil
}

func passwordResetUser(t *testing.T) *types.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password1"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}
	return &types.User{
		ID:           "user-1",
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: string(hash),
		TenantID:     1,
		IsActive:     true,
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now().Add(-time.Hour),
	}
}

func TestRequestPasswordResetCreatesSingleUseTokenForExistingUser(t *testing.T) {
	ctx := context.Background()
	user := passwordResetUser(t)
	tokenRepo := newPasswordResetTokenRepo()
	svc := NewUserService(nil, newPasswordResetUserRepo(user), tokenRepo, nil, nil)

	result, err := svc.RequestPasswordReset(ctx, " Alice@Example.com ")
	if err != nil {
		t.Fatalf("RequestPasswordReset returned error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("RequestPasswordReset result = %#v, want success", result)
	}
	if result.ResetToken == "" {
		t.Fatal("RequestPasswordReset returned empty reset token for existing user")
	}
	if len(tokenRepo.created) != 1 {
		t.Fatalf("created tokens = %d, want 1", len(tokenRepo.created))
	}
	token := tokenRepo.created[0]
	if token.UserID != user.ID {
		t.Fatalf("token.UserID = %q, want %q", token.UserID, user.ID)
	}
	if token.Token != result.ResetToken {
		t.Fatalf("stored token does not match returned reset token")
	}
	if token.TokenType != types.AuthTokenTypePasswordReset {
		t.Fatalf("token.TokenType = %q, want %q", token.TokenType, types.AuthTokenTypePasswordReset)
	}
	if token.IsRevoked {
		t.Fatal("new reset token must not be revoked")
	}
	if time.Until(token.ExpiresAt) <= 20*time.Minute || time.Until(token.ExpiresAt) > 31*time.Minute {
		t.Fatalf("token.ExpiresAt = %s, want roughly 30 minutes from now", token.ExpiresAt)
	}
}

func TestRequestPasswordResetDoesNotRevealUnknownEmail(t *testing.T) {
	ctx := context.Background()
	tokenRepo := newPasswordResetTokenRepo()
	svc := NewUserService(nil, newPasswordResetUserRepo(), tokenRepo, nil, nil)

	result, err := svc.RequestPasswordReset(ctx, "missing@example.com")
	if err != nil {
		t.Fatalf("RequestPasswordReset returned error for unknown email: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("RequestPasswordReset result = %#v, want generic success", result)
	}
	if result.ResetToken != "" {
		t.Fatalf("unknown email must not return a token, got %q", result.ResetToken)
	}
	if len(tokenRepo.created) != 0 {
		t.Fatalf("unknown email created %d tokens, want 0", len(tokenRepo.created))
	}
}

func TestResetPasswordRejectsInvalidOrExpiredToken(t *testing.T) {
	ctx := context.Background()
	user := passwordResetUser(t)
	expired := &types.AuthToken{
		ID:        "expired-token-id",
		UserID:    user.ID,
		Token:     "expired-token",
		TokenType: types.AuthTokenTypePasswordReset,
		ExpiresAt: time.Now().Add(-time.Minute),
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}
	tokenRepo := newPasswordResetTokenRepo(expired)
	userRepo := newPasswordResetUserRepo(user)
	svc := NewUserService(nil, userRepo, tokenRepo, nil, nil)

	if err := svc.ResetPassword(ctx, "missing-token", "123456"); err == nil {
		t.Fatal("ResetPassword accepted an unknown token")
	}
	if err := svc.ResetPassword(ctx, "expired-token", "123456"); err == nil {
		t.Fatal("ResetPassword accepted an expired token")
	}
	stored, err := tokenRepo.GetTokenByValue(ctx, "expired-token")
	if err != nil {
		t.Fatalf("load expired token: %v", err)
	}
	if stored.IsRevoked {
		t.Fatal("expired token should not be mutated by a rejected reset")
	}
	unchanged, err := userRepo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load unchanged user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(unchanged.PasswordHash), []byte("123456")); err == nil {
		t.Fatal("password unexpectedly changed")
	}
}

func TestResetPasswordUpdatesPasswordAndRevokesTokens(t *testing.T) {
	ctx := context.Background()
	user := passwordResetUser(t)
	resetToken := &types.AuthToken{
		ID:        "reset-token-id",
		UserID:    user.ID,
		Token:     "valid-reset-token",
		TokenType: types.AuthTokenTypePasswordReset,
		ExpiresAt: time.Now().Add(30 * time.Minute),
		CreatedAt: time.Now().Add(-time.Minute),
		UpdatedAt: time.Now().Add(-time.Minute),
	}
	accessToken := &types.AuthToken{
		ID:        "access-token-id",
		UserID:    user.ID,
		Token:     "old-access-token",
		TokenType: "access_token",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	tokenRepo := newPasswordResetTokenRepo(resetToken, accessToken)
	userRepo := newPasswordResetUserRepo(user)
	svc := NewUserService(nil, userRepo, tokenRepo, nil, nil)

	if err := svc.ResetPassword(ctx, "valid-reset-token", "123456"); err != nil {
		t.Fatalf("ResetPassword returned error: %v", err)
	}
	updated, err := userRepo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load updated user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("123456")); err != nil {
		t.Fatalf("new password does not match updated hash: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("old-password1")); err == nil {
		t.Fatal("old password still matches after reset")
	}
	if len(tokenRepo.revokeAllCalls) != 1 || tokenRepo.revokeAllCalls[0] != user.ID {
		t.Fatalf("RevokeTokensByUserID calls = %v, want [%s]", tokenRepo.revokeAllCalls, user.ID)
	}
	storedReset, err := tokenRepo.GetTokenByValue(ctx, "valid-reset-token")
	if err != nil {
		t.Fatalf("load stored reset token: %v", err)
	}
	if !storedReset.IsRevoked {
		t.Fatal("reset token must be revoked after successful use")
	}
	storedAccess, err := tokenRepo.GetTokenByValue(ctx, "old-access-token")
	if err != nil {
		t.Fatalf("load stored access token: %v", err)
	}
	if !storedAccess.IsRevoked {
		t.Fatal("existing login tokens must be revoked after password reset")
	}
	if err := svc.ResetPassword(ctx, "valid-reset-token", "654321"); err == nil {
		t.Fatal("ResetPassword accepted a reused reset token")
	}
}

func TestResetPasswordRequiresPasswordPolicy(t *testing.T) {
	ctx := context.Background()
	user := passwordResetUser(t)
	tokenRepo := newPasswordResetTokenRepo(&types.AuthToken{
		ID:        "reset-token-id",
		UserID:    user.ID,
		Token:     "valid-reset-token",
		TokenType: types.AuthTokenTypePasswordReset,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	})
	svc := NewUserService(nil, newPasswordResetUserRepo(user), tokenRepo, nil, nil)

	err := svc.ResetPassword(ctx, "valid-reset-token", "short")
	if err == nil {
		t.Fatal("ResetPassword accepted too-short password")
	}
	if !errors.Is(err, ErrPasswordPolicy) {
		t.Fatalf("ResetPassword error = %v, want ErrPasswordPolicy", err)
	}
}
