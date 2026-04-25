package api

import (
	"fmt"
	"net/http"

	"nofx/logger"

	"github.com/gin-gonic/gin"
)

type ExchangeConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Enabled   bool   `json:"enabled"`
	APIKey    string `json:"apiKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
	Testnet   bool   `json:"testnet,omitempty"`
}

// SafeExchangeConfig 返回给前端的安全配置结构。
// 敏感密钥不会回传，只暴露是否已配置和必要的公开字段。
type SafeExchangeConfig struct {
	ID                    string `json:"id"`
	ExchangeType          string `json:"exchange_type"`
	AccountName           string `json:"account_name"`
	Name                  string `json:"name"`
	Type                  string `json:"type"`
	Enabled               bool   `json:"enabled"`
	Configured            bool   `json:"configured"`
	Testnet               bool   `json:"testnet,omitempty"`
	HyperliquidWalletAddr string `json:"hyperliquidWalletAddr"`
	AsterUser             string `json:"asterUser"`
	AsterSigner           string `json:"asterSigner"`
	LighterWalletAddr     string `json:"lighterWalletAddr"`
}

type UpdateExchangeConfigRequest struct {
	Exchanges map[string]struct {
		Enabled                 bool   `json:"enabled"`
		APIKey                  string `json:"api_key"`
		SecretKey               string `json:"secret_key"`
		Passphrase              string `json:"passphrase"`
		Testnet                 bool   `json:"testnet"`
		HyperliquidWalletAddr   string `json:"hyperliquid_wallet_addr"`
		HyperliquidUnifiedAcct  bool   `json:"hyperliquid_unified_account"`
		AsterUser               string `json:"aster_user"`
		AsterSigner             string `json:"aster_signer"`
		AsterPrivateKey         string `json:"aster_private_key"`
		LighterWalletAddr       string `json:"lighter_wallet_addr"`
		LighterPrivateKey       string `json:"lighter_private_key"`
		LighterAPIKeyPrivateKey string `json:"lighter_api_key_private_key"`
		LighterAPIKeyIndex      int    `json:"lighter_api_key_index"`
	} `json:"exchanges"`
}

type CreateExchangeRequest struct {
	ExchangeType            string `json:"exchange_type" binding:"required"`
	AccountName             string `json:"account_name"`
	Enabled                 bool   `json:"enabled"`
	APIKey                  string `json:"api_key"`
	SecretKey               string `json:"secret_key"`
	Passphrase              string `json:"passphrase"`
	Testnet                 bool   `json:"testnet"`
	HyperliquidWalletAddr   string `json:"hyperliquid_wallet_addr"`
	HyperliquidUnifiedAcct  bool   `json:"hyperliquid_unified_account"`
	AsterUser               string `json:"aster_user"`
	AsterSigner             string `json:"aster_signer"`
	AsterPrivateKey         string `json:"aster_private_key"`
	LighterWalletAddr       string `json:"lighter_wallet_addr"`
	LighterPrivateKey       string `json:"lighter_private_key"`
	LighterAPIKeyPrivateKey string `json:"lighter_api_key_private_key"`
	LighterAPIKeyIndex      int    `json:"lighter_api_key_index"`
}

func (s *Server) handleGetExchangeConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	exchanges, err := s.store.Exchange().List(userID)
	if err != nil {
		SafeInternalError(c, "Failed to get exchange configs", err)
		return
	}

	safeExchanges := make([]SafeExchangeConfig, 0, len(exchanges))
	for _, exchange := range exchanges {
		configured := exchange.APIKey.String() != "" ||
			exchange.SecretKey.String() != "" ||
			exchange.Passphrase.String() != "" ||
			exchange.AsterPrivateKey.String() != "" ||
			exchange.LighterPrivateKey.String() != "" ||
			exchange.LighterAPIKeyPrivateKey.String() != ""

		safeExchanges = append(safeExchanges, SafeExchangeConfig{
			ID:                    exchange.ID,
			ExchangeType:          exchange.ExchangeType,
			AccountName:           exchange.AccountName,
			Name:                  exchange.Name,
			Type:                  exchange.Type,
			Enabled:               exchange.Enabled,
			Configured:            configured,
			Testnet:               exchange.Testnet,
			HyperliquidWalletAddr: exchange.HyperliquidWalletAddr,
			AsterUser:             exchange.AsterUser,
			AsterSigner:           exchange.AsterSigner,
			LighterWalletAddr:     exchange.LighterWalletAddr,
		})
	}

	c.JSON(http.StatusOK, safeExchanges)
}

func (s *Server) handleUpdateExchangeConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	var req UpdateExchangeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	tradersToReload := make(map[string]bool)
	for exchangeID, exchangeData := range req.Exchanges {
		traders, _ := s.store.Trader().ListByExchangeID(userID, exchangeID)
		for _, trader := range traders {
			tradersToReload[trader.ID] = true
		}

		err := s.store.Exchange().Update(
			userID,
			exchangeID,
			exchangeData.Enabled,
			exchangeData.APIKey,
			exchangeData.SecretKey,
			exchangeData.Passphrase,
			exchangeData.Testnet,
			exchangeData.HyperliquidWalletAddr,
			exchangeData.HyperliquidUnifiedAcct,
			exchangeData.AsterUser,
			exchangeData.AsterSigner,
			exchangeData.AsterPrivateKey,
			exchangeData.LighterWalletAddr,
			exchangeData.LighterPrivateKey,
			exchangeData.LighterAPIKeyPrivateKey,
			exchangeData.LighterAPIKeyIndex,
		)
		if err != nil {
			SafeInternalError(c, fmt.Sprintf("Update exchange %s", exchangeID), err)
			return
		}
	}

	s.exchangeAccountStateCache.Invalidate(userID)
	for traderID := range tradersToReload {
		s.traderManager.RemoveTrader(traderID)
	}
	if err := s.traderManager.LoadUserTradersFromStore(s.store, userID); err != nil {
		logger.Warnf("Reload traders after exchange update failed: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Exchange configuration updated"})
}

func (s *Server) handleCreateExchange(c *gin.Context) {
	userID := c.GetString("user_id")
	var req CreateExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	validTypes := map[string]bool{
		"binance": true, "bybit": true, "okx": true, "bitget": true,
		"hyperliquid": true, "aster": true, "lighter": true, "gate": true, "kucoin": true, "indodax": true,
	}
	if !validTypes[req.ExchangeType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid exchange type: %s", req.ExchangeType)})
		return
	}

	id, err := s.store.Exchange().Create(
		userID,
		req.ExchangeType,
		req.AccountName,
		req.Enabled,
		req.APIKey,
		req.SecretKey,
		req.Passphrase,
		req.Testnet,
		req.HyperliquidWalletAddr,
		req.HyperliquidUnifiedAcct,
		req.AsterUser,
		req.AsterSigner,
		req.AsterPrivateKey,
		req.LighterWalletAddr,
		req.LighterPrivateKey,
		req.LighterAPIKeyPrivateKey,
		req.LighterAPIKeyIndex,
	)
	if err != nil {
		SafeInternalError(c, "Failed to create exchange account", err)
		return
	}

	s.exchangeAccountStateCache.Invalidate(userID)
	c.JSON(http.StatusOK, gin.H{"message": "Exchange account created", "id": id})
}

func (s *Server) handleDeleteExchange(c *gin.Context) {
	userID := c.GetString("user_id")
	exchangeID := c.Param("id")
	if exchangeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exchange ID is required"})
		return
	}

	traders, err := s.store.Trader().List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check traders"})
		return
	}
	for _, trader := range traders {
		if trader.ExchangeID == exchangeID {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":       "Cannot delete exchange account that is in use by traders",
				"trader_id":   trader.ID,
				"trader_name": trader.Name,
			})
			return
		}
	}

	if err := s.store.Exchange().Delete(userID, exchangeID); err != nil {
		SafeInternalError(c, "Failed to delete exchange account", err)
		return
	}

	s.exchangeAccountStateCache.Invalidate(userID)
	c.JSON(http.StatusOK, gin.H{"message": "Exchange account deleted"})
}

func (s *Server) handleGetSupportedExchanges(c *gin.Context) {
	supportedExchanges := []SafeExchangeConfig{
		{ExchangeType: "binance", Name: "Binance Futures", Type: "cex"},
		{ExchangeType: "bybit", Name: "Bybit Futures", Type: "cex"},
		{ExchangeType: "okx", Name: "OKX Futures", Type: "cex"},
		{ExchangeType: "gate", Name: "Gate.io Futures", Type: "cex"},
		{ExchangeType: "kucoin", Name: "KuCoin Futures", Type: "cex"},
		{ExchangeType: "hyperliquid", Name: "Hyperliquid", Type: "dex"},
		{ExchangeType: "aster", Name: "Aster DEX", Type: "dex"},
		{ExchangeType: "lighter", Name: "LIGHTER DEX", Type: "dex"},
		{ExchangeType: "alpaca", Name: "Alpaca (US Stocks)", Type: "stock"},
		{ExchangeType: "forex", Name: "Forex (TwelveData)", Type: "forex"},
		{ExchangeType: "metals", Name: "Metals (TwelveData)", Type: "metals"},
	}

	c.JSON(http.StatusOK, supportedExchanges)
}
