package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type walletValidateRequest struct {
	PrivateKey string `json:"private_key"`
}

type walletValidateResponse struct {
	Valid         bool   `json:"valid"`
	Address       string `json:"address,omitempty"`
	BalanceUSDC   string `json:"balance_usdc,omitempty"`
	Claw402Status string `json:"claw402_status"` // "ok", "unreachable", "error"
	Error         string `json:"error,omitempty"`
}

func (s *Server) handleWalletValidate(c *gin.Context) {
	// 中文说明：
	// 自动钱包校验接口已经退出离线模式主流程。即使未来误接回路由，
	// 这里也只返回停用提示，避免服务端再次接收或处理钱包私钥。
	c.JSON(http.StatusGone, walletValidateResponse{
		Valid:         false,
		Claw402Status: "deprecated",
		Error:         "离线模式已停用自动钱包校验接口",
	})
}

type walletGenerateResponse struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
}

func (s *Server) handleWalletGenerate(c *gin.Context) {
	// 中文说明：
	// 旧的自动生成钱包接口仅保留为空壳兼容层，明确禁止再次生成和回传私钥。
	c.JSON(http.StatusGone, gin.H{
		"error": "离线模式已停用自动钱包生成接口",
	})
}

// Deprecated: 保留旧的外部健康检查辅助函数，仅用于未来回滚参考。
func checkClaw402Health() string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://claw402.ai/health")
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "ok"
	}
	return "error"
}
