package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"nofx/crypto"
	"nofx/store"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken     = errors.New("invalid or expired token")
	ErrAdminKeyRejected = errors.New("管理员密钥错误或已被锁定")
)

func issueTokenPair(userID string, authVersion int64) (*TokenPair, error) {
	sessionID, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	accessExp := now.Add(accessTokenTTL)
	refreshExp := now.Add(refreshTokenTTL)

	accessToken, err := signToken(userID, sessionID, TokenTypeAccess, authVersion, accessExp)
	if err != nil {
		return nil, err
	}
	refreshToken, err := signToken(userID, sessionID, TokenTypeRefresh, authVersion, refreshExp)
	if err != nil {
		return nil, err
	}

	session := &store.AuthSession{
		SessionID:        sessionID,
		UserID:           userID,
		RefreshTokenHash: hashRefreshToken(refreshToken),
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}
	if err := runtimeConfig.store.AuthSession().Upsert(session); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		SessionID:        sessionID,
		UserID:           userID,
	}, nil
}

func rotateRefreshToken(claims *Claims, refreshToken string) (*TokenPair, error) {
	session, err := runtimeConfig.store.AuthSession().Get(claims.SessionID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if session.RevokedAt != nil || time.Now().UTC().After(session.RefreshExpiresAt) {
		return nil, ErrInvalidToken
	}
	if session.RefreshTokenHash != hashRefreshToken(refreshToken) {
		_ = runtimeConfig.store.AuthSession().Revoke(session.SessionID, time.Now().UTC())
		return nil, ErrInvalidToken
	}

	now := time.Now().UTC()
	accessExp := now.Add(accessTokenTTL)
	refreshExp := now.Add(refreshTokenTTL)
	accessToken, err := signToken(claims.Subject, claims.SessionID, TokenTypeAccess, claims.AuthVersion, accessExp)
	if err != nil {
		return nil, err
	}
	newRefreshToken, err := signToken(claims.Subject, claims.SessionID, TokenTypeRefresh, claims.AuthVersion, refreshExp)
	if err != nil {
		return nil, err
	}

	session.RefreshTokenHash = hashRefreshToken(newRefreshToken)
	session.AccessExpiresAt = accessExp
	session.RefreshExpiresAt = refreshExp
	session.RevokedAt = nil
	session.UpdatedAt = time.Now().UTC()
	if err := runtimeConfig.store.AuthSession().Upsert(session); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     newRefreshToken,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		SessionID:        session.SessionID,
		UserID:           session.UserID,
	}, nil
}

func signToken(userID, sessionID, tokenType string, authVersion int64, exp time.Time) (string, error) {
	claims := &Claims{
		SessionID:   sessionID,
		TokenType:   tokenType,
		AuthVersion: authVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    issuerName,
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			NotBefore: jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(runtimeConfig.privateKey)
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	defer crypto.WipeBytes(buf)
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func currentCryptoService() *crypto.CryptoService {
	return crypto.GetGlobalCryptoService()
}
