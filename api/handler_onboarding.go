package api

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"nofx/logger"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

// Deprecated: 旧的新手自动钱包流程已退出主路径。
// 保留以下结构体与辅助函数，仅用于回滚旧逻辑或迁移历史数据时参考。
type beginnerOnboardingResponse struct {
	Address           string `json:"address"`
	PrivateKey        string `json:"private_key"`
	Chain             string `json:"chain"`
	Asset             string `json:"asset"`
	Provider          string `json:"provider"`
	DefaultModel      string `json:"default_model"`
	ConfiguredModelID string `json:"configured_model_id"`
	BalanceUSDC       string `json:"balance_usdc"`
	EnvSaved          bool   `json:"env_saved"`
	EnvPath           string `json:"env_path,omitempty"`
	ReusedExisting    bool   `json:"reused_existing"`
	EnvWarning        string `json:"env_warning,omitempty"`
}

type currentBeginnerWalletResponse struct {
	Found         bool   `json:"found"`
	Address       string `json:"address,omitempty"`
	BalanceUSDC   string `json:"balance_usdc,omitempty"`
	Source        string `json:"source,omitempty"`
	Claw402Status string `json:"claw402_status"`
}

func (s *Server) handleBeginnerOnboarding(c *gin.Context) {
	// 中文说明：
	// 旧的新手自动钱包流程已经退出离线模式主路径。即使未来有人误将该路由重新接回，
	// 这里也必须直接拒绝，避免再次生成、保存或回传钱包私钥。
	c.JSON(http.StatusGone, gin.H{
		"error": "离线模式已停用新手自动钱包流程",
	})
}

func (s *Server) handleCurrentBeginnerWallet(c *gin.Context) {
	// 中文说明：
	// 该接口原本会暴露自动钱包状态。离线认证改造后，前端已经不再依赖它，
	// 因此这里统一返回停用响应，防止后续误接入后重新泄露敏感钱包信息。
	c.JSON(http.StatusGone, gin.H{
		"error": "离线模式已停用新手自动钱包流程",
	})
}

func (s *Server) resolveBeginnerWallet(userID string) (privateKey string, address string, configuredModelID string, reused bool, err error) {
	// 1. Check if current user already has a claw402 wallet
	models, err := s.store.AIModel().List(userID)
	if err != nil {
		return "", "", "", false, err
	}

	for _, model := range models {
		if model == nil || model.Provider != "claw402" {
			continue
		}
		existingKey := strings.TrimSpace(model.APIKey.String())
		if existingKey == "" {
			continue
		}

		addr, addrErr := walletAddressFromPrivateKey(existingKey)
		if addrErr != nil {
			logger.Warnf("Existing claw402 key for user %s is invalid, regenerating: %v", userID, addrErr)
			break
		}

		return existingKey, addr, model.ID, true, nil
	}

	// 2. Check for orphan claw402 wallet from a previous account (e.g. after account reset).
	//    Adopt it to preserve funds.
	orphan, orphanErr := s.store.AIModel().FindOrphanClaw402()
	if orphanErr == nil && orphan != nil {
		existingKey := strings.TrimSpace(orphan.APIKey.String())
		if existingKey != "" {
			addr, addrErr := walletAddressFromPrivateKey(existingKey)
			if addrErr == nil {
				if adoptErr := s.store.AIModel().AdoptModel(orphan.ID, userID); adoptErr != nil {
					logger.Warnf("Failed to adopt orphan claw402 wallet for user %s: %v", userID, adoptErr)
				} else {
					logger.Infof("Adopted orphan claw402 wallet %s for new user %s (address: %s)", orphan.ID, userID, addr)
					return existingKey, addr, orphan.ID, true, nil
				}
			}
		}
	}

	// 3. No existing wallet found — generate a new one
	privateKeyObj, genErr := gethcrypto.GenerateKey()
	if genErr != nil {
		return "", "", "", false, genErr
	}

	addr := gethcrypto.PubkeyToAddress(privateKeyObj.PublicKey)
	keyHex := "0x" + hex.EncodeToString(gethcrypto.FromECDSA(privateKeyObj))
	return keyHex, addr.Hex(), "", false, nil
}

func (s *Server) findConfiguredClaw402ModelID(userID string) (string, error) {
	models, err := s.store.AIModel().List(userID)
	if err != nil {
		return "", err
	}

	for _, model := range models {
		if model != nil && model.Provider == "claw402" {
			return model.ID, nil
		}
	}

	return "", fmt.Errorf("claw402 model not found")
}

func walletAddressFromPrivateKey(privateKey string) (string, error) {
	key := strings.TrimSpace(privateKey)
	if !strings.HasPrefix(key, "0x") {
		return "", fmt.Errorf("private key must start with 0x")
	}
	if len(key) != 66 {
		return "", fmt.Errorf("private key must be 66 characters")
	}

	privateKeyObj, err := gethcrypto.HexToECDSA(strings.TrimPrefix(key, "0x"))
	if err != nil {
		return "", err
	}

	return gethcrypto.PubkeyToAddress(privateKeyObj.PublicKey).Hex(), nil
}

func persistBeginnerWalletEnv(privateKey string, address string) (bool, string, error) {
	paths := uniqueEnvPaths([]string{
		".env",
		filepath.Join(".", ".env"),
		"/app/.env",
	})

	var lastErr error
	for _, path := range paths {
		if path == "" {
			continue
		}

		if err := upsertEnvFile(path, map[string]string{
			"CLAW402_WALLET_KEY":     privateKey,
			"CLAW402_WALLET_ADDRESS": address,
			"CLAW402_DEFAULT_MODEL":  "glm-5",
		}); err != nil {
			lastErr = err
			continue
		}

		return true, path, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no writable .env path found")
	}
	return false, "", lastErr
}

func uniqueEnvPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	return result
}

func upsertEnvFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	existingLines := make([]string, 0)
	if file, err := os.Open(path); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			existingLines = append(existingLines, scanner.Text())
		}
		file.Close()
		if err := scanner.Err(); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	remaining := make(map[string]string, len(values))
	for key, value := range values {
		remaining[key] = value
	}

	updatedLines := make([]string, 0, len(existingLines)+len(values))
	for _, line := range existingLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || !strings.Contains(line, "=") {
			updatedLines = append(updatedLines, line)
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value, ok := remaining[key]
		if !ok {
			updatedLines = append(updatedLines, line)
			continue
		}

		updatedLines = append(updatedLines, fmt.Sprintf("%s=%s", key, value))
		delete(remaining, key)
	}

	for key, value := range remaining {
		updatedLines = append(updatedLines, fmt.Sprintf("%s=%s", key, value))
	}

	content := strings.Join(updatedLines, "\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return err
	}

	return nil
}
