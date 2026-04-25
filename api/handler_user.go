package api

import (
	"net/http"
	"strings"

	"nofx/auth"
	"nofx/bootstrap"

	"github.com/gin-gonic/gin"
)

type adminKeyLoginRequest struct {
	AdminKey string `json:"admin_key" binding:"required"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// handleAdminKeyLogin 使用本地管理员密钥登录。
// 认证成功后返回新的 access_token 和一次性 refresh_token。
func (s *Server) handleAdminKeyLogin(c *gin.Context) {
	var req adminKeyLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bootstrap.WriteJSONError(c, http.StatusBadRequest, "管理员密钥错误或已被锁定")
		return
	}

	if err := auth.VerifyAdminKey(strings.TrimSpace(req.AdminKey)); err != nil {
		bootstrap.WriteJSONError(c, http.StatusUnauthorized, auth.ErrAdminKeyRejected.Error())
		return
	}

	pair, err := auth.CreateSession()
	if err != nil {
		SafeInternalError(c, "创建登录会话失败", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":       pair.AccessToken,
		"refresh_token":      pair.RefreshToken,
		"token_type":         "Bearer",
		"expires_in":         2 * 60 * 60,
		"refresh_expires_in": 7 * 24 * 60 * 60,
	})
}

// handleRefreshToken 刷新 access token，并轮换 refresh token。
func (s *Server) handleRefreshToken(c *gin.Context) {
	var req refreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bootstrap.WriteJSONError(c, http.StatusBadRequest, "refresh_token 必填")
		return
	}

	pair, err := auth.RefreshSession(strings.TrimSpace(req.RefreshToken))
	if err != nil {
		bootstrap.WriteJSONError(c, http.StatusUnauthorized, "未登录或登录已失效")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":       pair.AccessToken,
		"refresh_token":      pair.RefreshToken,
		"token_type":         "Bearer",
		"expires_in":         2 * 60 * 60,
		"refresh_expires_in": 7 * 24 * 60 * 60,
	})
}

// handleLogout 吊销当前会话。
func (s *Server) handleLogout(c *gin.Context) {
	sessionID := c.GetString("session_id")
	if sessionID != "" {
		_ = auth.LogoutSession(sessionID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

// handleAuthStatus 返回当前登录状态。
// 该接口不要求鉴权，但会在带上 Authorization 时校验当前令牌。
func (s *Server) handleAuthStatus(c *gin.Context) {
	tokenString, ok := extractBearerToken(c.GetHeader("Authorization"))
	if !ok {
		c.JSON(http.StatusOK, gin.H{"is_logged_in": false})
		return
	}
	if _, err := auth.ValidateAccessToken(tokenString); err != nil {
		c.JSON(http.StatusOK, gin.H{"is_logged_in": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_logged_in": true})
}

func extractBearerToken(authHeader string) (string, bool) {
	if strings.TrimSpace(authHeader) == "" {
		return "", false
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", false
	}
	return parts[1], true
}

// Deprecated: 旧注册接口仅保留占位以便快速回滚，默认不再注册到路由。
func (s *Server) handleRegister(c *gin.Context) {
	bootstrap.WriteJSONError(c, http.StatusGone, "本地离线模式已移除注册流程，请使用管理员密钥登录")
}

// Deprecated: 旧密码登录接口仅保留占位以便快速回滚，默认不再注册到路由。
func (s *Server) handleLogin(c *gin.Context) {
	bootstrap.WriteJSONError(c, http.StatusGone, "本地离线模式已移除账号密码登录，请使用管理员密钥登录")
}

// Deprecated: 旧修改密码接口仅保留占位以便快速回滚，默认不再注册到路由。
func (s *Server) handleChangePassword(c *gin.Context) {
	bootstrap.WriteJSONError(c, http.StatusGone, "本地离线模式已移除密码修改流程")
}

// Deprecated: 旧重置密码接口仅保留占位以便快速回滚，默认不再注册到路由。
func (s *Server) handleResetPassword(c *gin.Context) {
	bootstrap.WriteJSONError(c, http.StatusGone, "本地离线模式已移除密码重置流程")
}

// Deprecated: 旧账户重置接口仅保留占位以便快速回滚，默认不再注册到路由。
func (s *Server) handleResetAccount(c *gin.Context) {
	bootstrap.WriteJSONError(c, http.StatusGone, "本地离线模式已移除账户重置流程，请使用 reset-admin-key")
}
