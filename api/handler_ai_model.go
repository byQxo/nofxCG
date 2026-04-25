package api

import (
	"fmt"
	"net/http"
	"strings"

	"nofx/logger"
	"nofx/security"
	"nofx/wallet"

	"github.com/gin-gonic/gin"
)

type ModelConfig struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Enabled      bool   `json:"enabled"`
	APIKey       string `json:"apiKey,omitempty"`
	CustomAPIURL string `json:"customApiUrl,omitempty"`
}

// SafeModelConfig 为前端返回的安全模型配置。
// 只暴露模型是否已配置，不回传任何 API Key 明文或密文。
type SafeModelConfig struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Provider        string `json:"provider"`
	Enabled         bool   `json:"enabled"`
	Configured      bool   `json:"configured"`
	CustomAPIURL    string `json:"customApiUrl"`
	CustomModelName string `json:"customModelName"`
	WalletAddress   string `json:"walletAddress,omitempty"`
	BalanceUSDC     string `json:"balanceUsdc,omitempty"`
}

type UpdateModelConfigRequest struct {
	Models map[string]struct {
		Enabled         bool   `json:"enabled"`
		APIKey          string `json:"api_key"`
		CustomAPIURL    string `json:"custom_api_url"`
		CustomModelName string `json:"custom_model_name"`
	} `json:"models"`
}

// handleGetModelConfigs 获取当前用户的 AI 模型配置。
// 返回值只包含“是否已配置”，不会回显敏感密钥。
func (s *Server) handleGetModelConfigs(c *gin.Context) {
	userID := c.GetString("user_id")

	models, err := s.store.AIModel().List(userID)
	if err != nil {
		SafeInternalError(c, "Failed to get AI model configs", err)
		return
	}

	if len(models) == 0 {
		c.JSON(http.StatusOK, []SafeModelConfig{
			{ID: "deepseek", Name: "DeepSeek AI", Provider: "deepseek", Enabled: false, Configured: false},
			{ID: "qwen", Name: "Qwen AI", Provider: "qwen", Enabled: false, Configured: false},
			{ID: "openai", Name: "OpenAI", Provider: "openai", Enabled: false, Configured: false},
			{ID: "claude", Name: "Claude AI", Provider: "claude", Enabled: false, Configured: false},
			{ID: "gemini", Name: "Gemini AI", Provider: "gemini", Enabled: false, Configured: false},
			{ID: "grok", Name: "Grok AI", Provider: "grok", Enabled: false, Configured: false},
			{ID: "kimi", Name: "Kimi AI", Provider: "kimi", Enabled: false, Configured: false},
			{ID: "minimax", Name: "MiniMax AI", Provider: "minimax", Enabled: false, Configured: false},
			{ID: "claw402", Name: "Claw402 (Base USDC)", Provider: "claw402", Enabled: false, Configured: false},
		})
		return
	}

	safeModels := make([]SafeModelConfig, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}

		apiKey := strings.TrimSpace(model.APIKey.String())
		item := SafeModelConfig{
			ID:              model.ID,
			Name:            model.Name,
			Provider:        model.Provider,
			Enabled:         model.Enabled,
			Configured:      apiKey != "",
			CustomAPIURL:    model.CustomAPIURL,
			CustomModelName: model.CustomModelName,
		}

		// claw402 仅回传派生地址和余额，方便用户确认钱包配置状态。
		if model.Provider == "claw402" && apiKey != "" {
			if walletAddress, addrErr := walletAddressFromPrivateKey(apiKey); addrErr == nil {
				item.WalletAddress = walletAddress
				item.BalanceUSDC = wallet.QueryUSDCBalanceStr(walletAddress)
			} else {
				logger.Warnf("Failed to derive claw402 wallet address for model %s: %v", model.ID, addrErr)
			}
		}

		safeModels = append(safeModels, item)
	}

	c.JSON(http.StatusOK, safeModels)
}

// handleUpdateModelConfigs 更新 AI 模型配置。
// 前端提交明文后，后端会立即加密写库，不会将敏感值写回响应。
func (s *Server) handleUpdateModelConfigs(c *gin.Context) {
	userID := c.GetString("user_id")

	var req UpdateModelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		SafeBadRequest(c, "Invalid request format")
		return
	}

	tradersToReload := make(map[string]bool)
	for modelID, modelData := range req.Models {
		if modelData.CustomAPIURL != "" {
			cleanURL := strings.TrimSuffix(modelData.CustomAPIURL, "#")
			if err := security.ValidateURL(cleanURL); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Invalid custom_api_url for model %s: %s", modelID, err.Error()),
				})
				return
			}
		}

		traders, _ := s.store.Trader().ListByAIModelID(userID, modelID)
		for _, trader := range traders {
			tradersToReload[trader.ID] = true
		}

		if err := s.store.AIModel().Update(
			userID,
			modelID,
			modelData.Enabled,
			modelData.APIKey,
			modelData.CustomAPIURL,
			modelData.CustomModelName,
		); err != nil {
			SafeInternalError(c, fmt.Sprintf("Update model %s", modelID), err)
			return
		}
	}

	for traderID := range tradersToReload {
		s.traderManager.RemoveTrader(traderID)
	}

	if err := s.traderManager.LoadUserTradersFromStore(s.store, userID); err != nil {
		logger.Warnf("Failed to reload user traders after AI model update: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model configuration updated"})
}

// handleGetSupportedModels 返回系统支持的 AI 模型列表。
func (s *Server) handleGetSupportedModels(c *gin.Context) {
	c.JSON(http.StatusOK, []map[string]interface{}{
		{"id": "deepseek", "name": "DeepSeek", "provider": "deepseek", "defaultModel": "deepseek-chat"},
		{"id": "qwen", "name": "Qwen", "provider": "qwen", "defaultModel": "qwen3-max"},
		{"id": "openai", "name": "OpenAI", "provider": "openai", "defaultModel": "gpt-5.1"},
		{"id": "claude", "name": "Claude", "provider": "claude", "defaultModel": "claude-opus-4-1"},
		{"id": "gemini", "name": "Google Gemini", "provider": "gemini", "defaultModel": "gemini-2.5-pro"},
		{"id": "grok", "name": "Grok (xAI)", "provider": "grok", "defaultModel": "grok-3-latest"},
		{"id": "kimi", "name": "Kimi (Moonshot)", "provider": "kimi", "defaultModel": "moonshot-v1-8k"},
		{"id": "minimax", "name": "MiniMax", "provider": "minimax", "defaultModel": "MiniMax-M2"},
		{"id": "claw402", "name": "Claw402 (Base USDC)", "provider": "claw402", "defaultModel": "glm-5"},
	})
}
