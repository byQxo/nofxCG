package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"nofx/bootstrap"
	"nofx/store"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
	accessTokenTTL   = 2 * time.Hour
	refreshTokenTTL  = 7 * 24 * time.Hour
	issuerName       = "nofx-local"
)

var runtimeConfig struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	store      *store.Store
	ownerID    string
}

// Claims 是新的 RS256 JWT 载荷。
// 这里只保留最小会话信息，避免把邮箱等上下文写入令牌。
type Claims struct {
	SessionID   string `json:"sid"`
	TokenType   string `json:"typ"`
	AuthVersion int64  `json:"auth_version"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	SessionID        string
	UserID           string
}

// Init 使用启动阶段准备好的根密钥和 Store 初始化认证运行时。
func Init(st *store.Store, ownerID string, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) error {
	if st == nil {
		return errors.New("store is required")
	}
	if privateKey == nil || publicKey == nil {
		return errors.New("rsa key pair is required")
	}
	runtimeConfig.store = st
	runtimeConfig.ownerID = ownerID
	runtimeConfig.privateKey = privateKey
	runtimeConfig.publicKey = publicKey
	return nil
}

// OwnerID 返回固定实例拥有者 ID，供鉴权中间件注入上下文。
func OwnerID() string {
	return runtimeConfig.ownerID
}

// HashPassword 保留旧工具函数供兼容路径复用。
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 保留旧工具函数供兼容路径复用。
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// VerifyAdminKey 校验本地管理员密钥，并应用持久化限流。
func VerifyAdminKey(adminKey string) error {
	if runtimeConfig.store == nil {
		return errors.New("auth runtime is not initialized")
	}

	allowed, err := canAttemptAdminLogin(runtimeConfig.store)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrAdminKeyRejected
	}

	hash, err := bootstrap.ReadAdminHash(runtimeConfig.store, currentCryptoService())
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(adminKey)) != nil {
		if _, rateErr := recordAdminLoginFailure(runtimeConfig.store); rateErr != nil {
			return rateErr
		}
		return ErrAdminKeyRejected
	}
	if err := clearAdminLoginFailures(runtimeConfig.store); err != nil {
		return err
	}
	return nil
}

// CreateSession 在管理员密钥校验成功后签发新的 access/refresh token。
func CreateSession() (*TokenPair, error) {
	if runtimeConfig.store == nil {
		return nil, errors.New("auth runtime is not initialized")
	}
	authVersion, err := bootstrap.EnsureAuthVersion(runtimeConfig.store)
	if err != nil {
		return nil, err
	}
	return issueTokenPair(runtimeConfig.ownerID, authVersion)
}

// RefreshSession 校验并轮换一次性 refresh token。
func RefreshSession(refreshToken string) (*TokenPair, error) {
	if runtimeConfig.store == nil {
		return nil, errors.New("auth runtime is not initialized")
	}
	claims, err := ValidateToken(refreshToken, TokenTypeRefresh)
	if err != nil {
		return nil, err
	}
	authVersion, err := bootstrap.EnsureAuthVersion(runtimeConfig.store)
	if err != nil {
		return nil, err
	}
	if claims.AuthVersion != authVersion {
		return nil, ErrInvalidToken
	}
	return rotateRefreshToken(claims, refreshToken)
}

// LogoutSession 吊销当前会话。
func LogoutSession(sessionID string) error {
	if runtimeConfig.store == nil || sessionID == "" {
		return nil
	}
	return runtimeConfig.store.AuthSession().Revoke(sessionID, time.Now().UTC())
}

// ValidateToken 验证 RS256 JWT 并检查 token 类型。
func ValidateToken(tokenString string, expectedType string) (*Claims, error) {
	if runtimeConfig.publicKey == nil {
		return nil, errors.New("public key is not configured")
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return runtimeConfig.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != expectedType {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ValidateAccessToken 会额外检查会话是否被吊销以及 auth_version 是否一致。
func ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := ValidateToken(tokenString, TokenTypeAccess)
	if err != nil {
		return nil, err
	}
	authVersion, err := bootstrap.EnsureAuthVersion(runtimeConfig.store)
	if err != nil {
		return nil, err
	}
	if claims.AuthVersion != authVersion {
		return nil, ErrInvalidToken
	}
	session, err := runtimeConfig.store.AuthSession().Get(claims.SessionID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if session.RevokedAt != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
