package bootstrap

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"nofx/logger"

	"github.com/gin-gonic/gin"
)

// RequestLogMiddleware 自定义请求日志，避免默认访问日志把敏感头和请求体写入日志。
func RequestLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		c.Next()

		logger.Infof("HTTP %s %s -> %d (%s) ip=%s",
			method,
			path,
			c.Writer.Status(),
			time.Since(start).Round(time.Millisecond),
			clientIP,
		)
	}
}

// MaskValue 统一的敏感日志脱敏函数。
func MaskValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 6 {
		return "***"
	}
	return trimmed[:2] + strings.Repeat("*", len(trimmed)-6) + trimmed[len(trimmed)-4:]
}

// SanitizeErrorMessage 避免错误日志中输出完整密文、令牌或密钥。
func SanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	replacements := []string{
		"Bearer ",
		"ENC:v1:",
		"-----BEGIN",
		"eyJ",
	}
	for _, marker := range replacements {
		if idx := strings.Index(message, marker); idx >= 0 {
			return message[:idx] + "[REDACTED]"
		}
	}
	return message
}

// SanitizeBodyForLog 对少量 JSON 请求做按字段过滤，避免后续调试日志泄露敏感输入。
func SanitizeBodyForLog(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	for _, key := range []string{
		"admin_key",
		"refresh_token",
		"api_key",
		"secret_key",
		"passphrase",
		"aster_private_key",
		"lighter_private_key",
		"lighter_api_key_private_key",
		"bot_token",
	} {
		if value, ok := payload[key]; ok {
			payload[key] = MaskValue(strings.TrimSpace(toString(value)))
		}
	}
	marshaled, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(marshaled)
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

// WriteJSONError 统一返回错误信息，避免不同鉴权分支暴露内部状态。
func WriteJSONError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

// IsDockerEnvironment 使用常见容器标记做 best-effort 检测。
func IsDockerEnvironment() bool {
	if _, err := http.Dir("/").Open(".dockerenv"); err == nil {
		return true
	}
	return false
}
