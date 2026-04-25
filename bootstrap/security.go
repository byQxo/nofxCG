package bootstrap

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nofx/config"
	"nofx/crypto"
	"nofx/logger"
	"nofx/store"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	SystemConfigWrappedDataKey       = "security.encrypted_data_key"
	SystemConfigAdminKeyHash         = "security.admin_key_hash"
	SystemConfigAuthVersion          = "security.auth_version"
	SystemConfigInstanceOwnerID      = "security.instance_owner_id"
	SystemConfigMigrationBackupReady = "security.migration_backup_at"
)

type SecurityRuntime struct {
	CryptoService        *crypto.CryptoService
	OwnerID              string
	KeysDir              string
	BackupDir            string
	PublicKeyFingerprint string
}

// InitSecurity 初始化离线根密钥、数据密钥和管理员登录密钥。
func InitSecurity(cfg *config.Config, st *store.Store) (*SecurityRuntime, error) {
	if err := os.MkdirAll(cfg.KeysDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.BackupDir, 0o700); err != nil {
		return nil, err
	}
	if err := securePathPermissions(cfg.KeysDir, true); err != nil {
		return nil, err
	}

	privateKey, publicPEM, createdRootKey, err := ensureRootKeyPair(cfg.KeysDir)
	if err != nil {
		return nil, err
	}

	dataKey, createdDataKey, importedLegacyKey, err := ensureDataKey(st, privateKey)
	if err != nil {
		return nil, err
	}

	cryptoService, err := crypto.NewCryptoServiceFromKeys(privateKey, dataKey)
	if err != nil {
		return nil, err
	}
	crypto.SetGlobalCryptoService(cryptoService)

	ownerID, err := ensureInstanceOwnerID(st)
	if err != nil {
		return nil, err
	}
	if _, err := ensureAuthVersion(st); err != nil {
		return nil, err
	}
	adminKey, createdAdminKey, err := ensureAdminKey(st, cryptoService)
	if err != nil {
		return nil, err
	}
	if err := maybeCreateMigrationBackup(cfg, st); err != nil {
		return nil, err
	}

	fingerprint, err := cryptoService.PublicKeyFingerprint()
	if err != nil {
		return nil, err
	}

	logger.Infof("安全初始化完成，密钥目录: %s", cfg.KeysDir)
	logger.Infof("根公钥指纹: %s", MaskValue(fingerprint))
	logger.Warnf("请立即备份根密钥目录 %s。根密钥丢失后，所有已加密配置将无法恢复。", cfg.KeysDir)
	if createdRootKey {
		logger.Infof("已生成新的本地 RSA-4096 根密钥对。")
	}
	if createdDataKey {
		if importedLegacyKey {
			logger.Warn("已导入旧 DATA_ENCRYPTION_KEY 作为现有数据密钥，请确认迁移完成后删除旧 .env 中的明文密钥。")
		} else {
			logger.Infof("已生成新的 AES-256 数据密钥。")
		}
	}
	if createdAdminKey {
		logger.Warn("========== 本地管理员登录密钥（仅首次显示） ==========")
		logger.Warn(adminKey)
		logger.Warn("请保存该密钥，并妥善备份上方提示的根密钥目录。")
	}
	if IsDockerEnvironment() {
		logger.Warn("检测到容器环境，请确认 ./config/keys 与 ./backup 已挂载到持久化卷，否则删除容器后密钥可能丢失。")
	}
	checkLegacyEnvWarning()

	_ = publicPEM

	return &SecurityRuntime{
		CryptoService:        cryptoService,
		OwnerID:              ownerID,
		KeysDir:              cfg.KeysDir,
		BackupDir:            cfg.BackupDir,
		PublicKeyFingerprint: fingerprint,
	}, nil
}

// ResetAdminKey 重新生成管理员密钥并提升认证版本，强制所有会话失效。
func ResetAdminKey(st *store.Store, cs *crypto.CryptoService) (string, error) {
	adminKey, adminHash, err := generateAdminKeyAndHash()
	if err != nil {
		return "", err
	}
	defer crypto.WipeBytes([]byte(adminHash))
	encryptedHash, err := cs.EncryptForStorage(adminHash)
	if err != nil {
		return "", err
	}
	if err := st.SetSystemConfig(SystemConfigAdminKeyHash, encryptedHash); err != nil {
		return "", err
	}
	if _, err := IncrementAuthVersion(st); err != nil {
		return "", err
	}
	return adminKey, nil
}

type aiModelSnapshot struct {
	ID              string
	APIKey          string
	CustomAPIURL    string
	CustomModelName string
	Enabled         bool
}

type exchangeSnapshot struct {
	ID                      string
	Enabled                 bool
	Testnet                 bool
	HyperliquidWalletAddr   string
	HyperliquidUnifiedAcct  bool
	AsterUser               string
	AsterSigner             string
	LighterWalletAddr       string
	LighterAPIKeyIndex      int
	APIKey                  string
	SecretKey               string
	Passphrase              string
	AsterPrivateKey         string
	LighterPrivateKey       string
	LighterAPIKeyPrivateKey string
}

type strategySnapshot struct {
	ID       string
	UserID   string
	Config   *store.StrategyConfig
	IsPublic bool
}

type telegramSnapshot struct {
	Exists    bool
	BotToken  string
	TelegramID uint
}

// EnsureAuthVersion 返回当前认证版本。
func EnsureAuthVersion(st *store.Store) (int64, error) {
	return ensureAuthVersion(st)
}

// IncrementAuthVersion 递增认证版本，用于统一吊销所有令牌。
func IncrementAuthVersion(st *store.Store) (int64, error) {
	current, err := ensureAuthVersion(st)
	if err != nil {
		return 0, err
	}
	current++
	if err := st.SetSystemConfig(SystemConfigAuthVersion, fmt.Sprintf("%d", current)); err != nil {
		return 0, err
	}
	return current, nil
}

// ResetRootKey 重新生成根密钥对与数据密钥，并批量重加密现有敏感数据。
// 该流程会先创建备份目录，失败时可通过 restore-backup 回滚。
func ResetRootKey(cfg *config.Config, st *store.Store, runtime *SecurityRuntime) (string, error) {
	if runtime == nil || runtime.CryptoService == nil {
		return "", errors.New("security runtime is not initialized")
	}

	backupDir, err := CreateBackup(cfg, "root-key-reset")
	if err != nil {
		return "", err
	}

	models, err := collectAIModelSnapshots(st, runtime.OwnerID)
	if err != nil {
		return backupDir, err
	}
	exchanges, err := collectExchangeSnapshots(st, runtime.OwnerID)
	if err != nil {
		return backupDir, err
	}
	strategies, err := collectStrategySnapshots(st, runtime.OwnerID)
	if err != nil {
		return backupDir, err
	}
	telegramConfig, err := collectTelegramSnapshot(st)
	if err != nil {
		return backupDir, err
	}
	adminHash, err := ReadAdminHash(st, runtime.CryptoService)
	if err != nil {
		return backupDir, err
	}

	newPrivateKey, newPublicPEM, err := generateFreshRootKeyPair()
	if err != nil {
		return backupDir, err
	}
	newDataKey := make([]byte, 32)
	if _, err := rand.Read(newDataKey); err != nil {
		return backupDir, err
	}

	newCryptoService, err := crypto.NewCryptoServiceFromKeys(newPrivateKey, newDataKey)
	if err != nil {
		return backupDir, err
	}
	wrappedDataKey, err := newCryptoService.WrapDataKey(newDataKey)
	if err != nil {
		return backupDir, err
	}
	encryptedAdminHash, err := newCryptoService.EncryptForStorage(adminHash)
	if err != nil {
		return backupDir, err
	}
	newFingerprint, err := newCryptoService.PublicKeyFingerprint()
	if err != nil {
		return backupDir, err
	}

	privateKeyPath := filepath.Join(cfg.KeysDir, "root_private.pem")
	publicKeyPath := filepath.Join(cfg.KeysDir, "root_public.pem")
	tempPrivatePath := privateKeyPath + ".new"
	tempPublicPath := publicKeyPath + ".new"

	newPrivatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(newPrivateKey),
	})
	if err := writePEMFile(tempPrivatePath, newPrivatePEM); err != nil {
		return backupDir, err
	}
	if err := writePEMFile(tempPublicPath, []byte(newPublicPEM)); err != nil {
		_ = os.Remove(tempPrivatePath)
		return backupDir, err
	}

	oldGlobal := crypto.GetGlobalCryptoService()
	crypto.SetGlobalCryptoService(newCryptoService)
	defer func() {
		if runtime.CryptoService != newCryptoService {
			crypto.SetGlobalCryptoService(oldGlobal)
		}
	}()

	currentVersion, err := ensureAuthVersion(st)
	if err != nil {
		_ = os.Remove(tempPrivatePath)
		_ = os.Remove(tempPublicPath)
		return backupDir, err
	}

	if err := st.Transaction(func(tx *gorm.DB) error {
		nextVersion := currentVersion + 1
		if err := tx.Exec(`
			INSERT INTO system_config (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value
		`, SystemConfigWrappedDataKey, wrappedDataKey).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO system_config (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value
		`, SystemConfigAdminKeyHash, encryptedAdminHash).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO system_config (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value
		`, SystemConfigAuthVersion, fmt.Sprintf("%d", nextVersion)).Error; err != nil {
			return err
		}

		for _, model := range models {
			if err := tx.Model(&store.AIModel{}).
				Where("id = ? AND user_id = ?", model.ID, runtime.OwnerID).
				Updates(map[string]interface{}{
					"api_key":           crypto.EncryptedString(model.APIKey),
					"custom_api_url":    model.CustomAPIURL,
					"custom_model_name": model.CustomModelName,
					"enabled":           model.Enabled,
					"updated_at":        time.Now().UTC(),
				}).Error; err != nil {
				return err
			}
		}

		for _, exchange := range exchanges {
			if err := tx.Model(&store.Exchange{}).
				Where("id = ? AND user_id = ?", exchange.ID, runtime.OwnerID).
				Updates(map[string]interface{}{
					"enabled":                     exchange.Enabled,
					"testnet":                     exchange.Testnet,
					"hyperliquid_wallet_addr":     exchange.HyperliquidWalletAddr,
					"hyperliquid_unified_account": exchange.HyperliquidUnifiedAcct,
					"aster_user":                  exchange.AsterUser,
					"aster_signer":                exchange.AsterSigner,
					"lighter_wallet_addr":         exchange.LighterWalletAddr,
					"lighter_api_key_index":       exchange.LighterAPIKeyIndex,
					"api_key":                     crypto.EncryptedString(exchange.APIKey),
					"secret_key":                  crypto.EncryptedString(exchange.SecretKey),
					"passphrase":                  crypto.EncryptedString(exchange.Passphrase),
					"aster_private_key":           crypto.EncryptedString(exchange.AsterPrivateKey),
					"lighter_private_key":         crypto.EncryptedString(exchange.LighterPrivateKey),
					"lighter_api_key_private_key": crypto.EncryptedString(exchange.LighterAPIKeyPrivateKey),
					"updated_at":                  time.Now().UTC(),
				}).Error; err != nil {
				return err
			}
		}

		for _, strategyItem := range strategies {
			updatedStrategy := &store.Strategy{
				ID:            strategyItem.ID,
				UserID:        strategyItem.UserID,
				IsPublic:      strategyItem.IsPublic,
				ConfigVisible: false,
			}
			if strategyItem.Config != nil {
				if err := updatedStrategy.SetConfig(strategyItem.Config); err != nil {
					return err
				}
			}
			if err := tx.Model(&store.Strategy{}).
				Where("id = ? AND user_id = ?", strategyItem.ID, strategyItem.UserID).
				Updates(map[string]interface{}{
					"config":     updatedStrategy.Config,
					"updated_at": time.Now().UTC(),
				}).Error; err != nil {
				return err
			}
		}

		if telegramConfig.Exists {
			if err := tx.Model(&store.TelegramConfig{}).
				Where("id = ?", telegramConfig.TelegramID).
				Updates(map[string]interface{}{
					"bot_token":  crypto.EncryptedString(telegramConfig.BotToken),
					"updated_at": time.Now().UTC(),
				}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("1 = 1").Delete(&store.AuthSession{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		_ = os.Remove(tempPrivatePath)
		_ = os.Remove(tempPublicPath)
		return backupDir, fmt.Errorf("root key reset failed, please use restore-backup to recover: %w", err)
	}

	if err := replaceFile(tempPrivatePath, privateKeyPath); err != nil {
		return backupDir, fmt.Errorf("failed to replace root private key, please restore backup %s manually: %w", filepath.Base(backupDir), err)
	}
	if err := replaceFile(tempPublicPath, publicKeyPath); err != nil {
		return backupDir, fmt.Errorf("failed to replace root public key, please restore backup %s manually: %w", filepath.Base(backupDir), err)
	}

	runtime.CryptoService = newCryptoService
	runtime.PublicKeyFingerprint = newFingerprint
	crypto.SetGlobalCryptoService(newCryptoService)
	logger.Warnf("根密钥轮换完成，新公钥指纹: %s", MaskValue(newFingerprint))
	logger.Warnf("轮换前备份已写入: %s", backupDir)
	return backupDir, nil
}

func ensureRootKeyPair(keysDir string) (*rsa.PrivateKey, string, bool, error) {
	privateKeyPath := filepath.Join(keysDir, "root_private.pem")
	publicKeyPath := filepath.Join(keysDir, "root_public.pem")

	if _, err := os.Stat(privateKeyPath); err == nil {
		privatePEM, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return nil, "", false, err
		}
		privateKey, err := crypto.ParseRSAPrivateKeyFromPEM(privatePEM)
		if err != nil {
			return nil, "", false, err
		}
		publicPEM, _ := os.ReadFile(publicKeyPath)
		return privateKey, string(publicPEM), false, nil
	}

	privatePEM := os.Getenv(crypto.EnvRSAPrivateKey)
	var privateKey *rsa.PrivateKey
	var err error
	if strings.TrimSpace(privatePEM) != "" {
		privateKey, err = crypto.ParseRSAPrivateKeyFromPEM([]byte(strings.ReplaceAll(privatePEM, "\\n", "\n")))
		if err != nil {
			return nil, "", false, err
		}
	} else {
		privatePEMString, publicPEMString, err := crypto.GenerateKeyPair()
		if err != nil {
			return nil, "", false, err
		}
		privateKey, err = crypto.ParseRSAPrivateKeyFromPEM([]byte(privatePEMString))
		if err != nil {
			return nil, "", false, err
		}
		if err := writePEMFile(privateKeyPath, []byte(privatePEMString)); err != nil {
			return nil, "", false, err
		}
		if err := writePEMFile(publicKeyPath, []byte(publicPEMString)); err != nil {
			return nil, "", false, err
		}
		return privateKey, publicPEMString, true, nil
	}

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", false, err
	}
	publicPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	privatePEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err := writePEMFile(privateKeyPath, privatePEMBytes); err != nil {
		return nil, "", false, err
	}
	if err := writePEMFile(publicKeyPath, publicPEMBytes); err != nil {
		return nil, "", false, err
	}
	return privateKey, string(publicPEMBytes), true, nil
}

func generateFreshRootKeyPair() (*rsa.PrivateKey, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, "", err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", err
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	return privateKey, string(publicPEM), nil
}

func ensureDataKey(st *store.Store, privateKey *rsa.PrivateKey) ([]byte, bool, bool, error) {
	wrapped, err := st.GetSystemConfig(SystemConfigWrappedDataKey)
	if err != nil {
		return nil, false, false, err
	}

	tempService, err := crypto.NewCryptoServiceFromKeys(privateKey, make([]byte, 32))
	if err != nil {
		return nil, false, false, err
	}

	if wrapped != "" {
		key, err := tempService.UnwrapDataKey(wrapped)
		return key, false, false, err
	}

	var dataKey []byte
	importedLegacyKey := false
	if legacy := strings.TrimSpace(os.Getenv(crypto.EnvDataEncryptionKey)); legacy != "" {
		dataKey, err = crypto.LoadDataKeyFromString(legacy)
		if err != nil {
			return nil, false, false, err
		}
		importedLegacyKey = true
	} else {
		dataKey = make([]byte, 32)
		if _, err := rand.Read(dataKey); err != nil {
			return nil, false, false, err
		}
	}

	wrapped, err = tempService.WrapDataKey(dataKey)
	if err != nil {
		return nil, false, false, err
	}
	if err := st.SetSystemConfig(SystemConfigWrappedDataKey, wrapped); err != nil {
		return nil, false, false, err
	}
	return dataKey, true, importedLegacyKey, nil
}

func ensureInstanceOwnerID(st *store.Store) (string, error) {
	ownerID, err := st.GetSystemConfig(SystemConfigInstanceOwnerID)
	if err != nil {
		return "", err
	}
	if ownerID != "" {
		return ownerID, nil
	}
	users, err := st.User().GetAll()
	if err != nil {
		return "", err
	}
	ownerID = "default"
	if len(users) > 0 && strings.TrimSpace(users[0].ID) != "" {
		ownerID = users[0].ID
	}
	if err := st.SetSystemConfig(SystemConfigInstanceOwnerID, ownerID); err != nil {
		return "", err
	}
	return ownerID, nil
}

func ensureAuthVersion(st *store.Store) (int64, error) {
	value, err := st.GetSystemConfig(SystemConfigAuthVersion)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(value) == "" {
		if err := st.SetSystemConfig(SystemConfigAuthVersion, "1"); err != nil {
			return 0, err
		}
		return 1, nil
	}
	var current int64
	if _, err := fmt.Sscanf(value, "%d", &current); err != nil || current <= 0 {
		current = 1
		if err := st.SetSystemConfig(SystemConfigAuthVersion, "1"); err != nil {
			return 0, err
		}
	}
	return current, nil
}

func ensureAdminKey(st *store.Store, cs *crypto.CryptoService) (string, bool, error) {
	current, err := st.GetSystemConfig(SystemConfigAdminKeyHash)
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(current) != "" {
		return "", false, nil
	}
	adminKey, adminHash, err := generateAdminKeyAndHash()
	if err != nil {
		return "", false, err
	}
	encryptedHash, err := cs.EncryptForStorage(adminHash)
	if err != nil {
		return "", false, err
	}
	if err := st.SetSystemConfig(SystemConfigAdminKeyHash, encryptedHash); err != nil {
		return "", false, err
	}
	return adminKey, true, nil
}

func generateAdminKeyAndHash() (string, string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", err
	}
	adminKey := base64.RawStdEncoding.EncodeToString(randomBytes)
	crypto.WipeBytes(randomBytes)
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(adminKey), 12)
	if err != nil {
		return "", "", err
	}
	return adminKey, string(hashBytes), nil
}

func maybeCreateMigrationBackup(cfg *config.Config, st *store.Store) error {
	marker, err := st.GetSystemConfig(SystemConfigMigrationBackupReady)
	if err != nil {
		return err
	}
	if marker != "" {
		return nil
	}

	needsBackup := false
	if strings.TrimSpace(os.Getenv(crypto.EnvDataEncryptionKey)) != "" || strings.TrimSpace(os.Getenv(crypto.EnvRSAPrivateKey)) != "" {
		needsBackup = true
	}
	for _, candidate := range []string{".env", "config.json", "configbak.json"} {
		if _, err := os.Stat(candidate); err == nil {
			needsBackup = true
			break
		}
	}
	if !needsBackup {
		return nil
	}

	backupDir, err := CreateBackup(cfg, "migration")
	if err != nil {
		return err
	}
	if err := st.SetSystemConfig(SystemConfigMigrationBackupReady, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	logger.Warnf("已创建迁移前备份: %s", backupDir)
	return nil
}

func checkLegacyEnvWarning() {
	raw, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	legacyKeys := []string{
		crypto.EnvDataEncryptionKey,
		crypto.EnvRSAPrivateKey,
		"JWT_SECRET",
		"CLAW402_WALLET_KEY",
	}
	found := make([]string, 0, len(legacyKeys))
	for _, key := range legacyKeys {
		if strings.Contains(string(raw), key+"=") {
			found = append(found, key)
		}
	}
	if len(found) == 0 {
		return
	}
	logger.Warnf("检测到旧 .env 中仍包含明文敏感项: %s。请在确认迁移完成后安全删除或清理。", strings.Join(found, ", "))
}

func writePEMFile(path string, content []byte) error {
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return err
	}
	return securePathPermissions(path, false)
}

func replaceFile(src, dst string) error {
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	return securePathPermissions(dst, false)
}

func collectAIModelSnapshots(st *store.Store, ownerID string) ([]aiModelSnapshot, error) {
	models, err := st.AIModel().List(ownerID)
	if err != nil {
		return nil, err
	}
	result := make([]aiModelSnapshot, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		result = append(result, aiModelSnapshot{
			ID:              model.ID,
			APIKey:          model.APIKey.String(),
			CustomAPIURL:    model.CustomAPIURL,
			CustomModelName: model.CustomModelName,
			Enabled:         model.Enabled,
		})
	}
	return result, nil
}

func collectExchangeSnapshots(st *store.Store, ownerID string) ([]exchangeSnapshot, error) {
	exchanges, err := st.Exchange().List(ownerID)
	if err != nil {
		return nil, err
	}
	result := make([]exchangeSnapshot, 0, len(exchanges))
	for _, exchange := range exchanges {
		if exchange == nil {
			continue
		}
		result = append(result, exchangeSnapshot{
			ID:                      exchange.ID,
			Enabled:                 exchange.Enabled,
			Testnet:                 exchange.Testnet,
			HyperliquidWalletAddr:   exchange.HyperliquidWalletAddr,
			HyperliquidUnifiedAcct:  exchange.HyperliquidUnifiedAcct,
			AsterUser:               exchange.AsterUser,
			AsterSigner:             exchange.AsterSigner,
			LighterWalletAddr:       exchange.LighterWalletAddr,
			LighterAPIKeyIndex:      exchange.LighterAPIKeyIndex,
			APIKey:                  exchange.APIKey.String(),
			SecretKey:               exchange.SecretKey.String(),
			Passphrase:              exchange.Passphrase.String(),
			AsterPrivateKey:         exchange.AsterPrivateKey.String(),
			LighterPrivateKey:       exchange.LighterPrivateKey.String(),
			LighterAPIKeyPrivateKey: exchange.LighterAPIKeyPrivateKey.String(),
		})
	}
	return result, nil
}

func collectStrategySnapshots(st *store.Store, ownerID string) ([]strategySnapshot, error) {
	strategies, err := st.Strategy().List(ownerID)
	if err != nil {
		return nil, err
	}
	result := make([]strategySnapshot, 0, len(strategies))
	for _, item := range strategies {
		if item == nil {
			continue
		}
		config, err := item.ParseConfig()
		if err != nil {
			return nil, err
		}
		result = append(result, strategySnapshot{
			ID:       item.ID,
			UserID:   item.UserID,
			Config:   config,
			IsPublic: item.IsPublic,
		})
	}
	return result, nil
}

func collectTelegramSnapshot(st *store.Store) (*telegramSnapshot, error) {
	cfg, err := st.TelegramConfig().Get()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &telegramSnapshot{}, nil
		}
		return nil, err
	}
	if cfg == nil {
		return &telegramSnapshot{}, nil
	}
	return &telegramSnapshot{
		Exists:     true,
		BotToken:   cfg.BotToken.String(),
		TelegramID: cfg.ID,
	}, nil
}

// ReadAdminHash 供认证模块读取管理员密钥哈希。
func ReadAdminHash(st *store.Store, cs *crypto.CryptoService) (string, error) {
	encryptedHash, err := st.GetSystemConfig(SystemConfigAdminKeyHash)
	if err != nil {
		return "", err
	}
	if encryptedHash == "" {
		return "", errors.New("admin key hash is missing")
	}
	return cs.DecryptFromStorage(encryptedHash)
}

// ReadRateLimitState 从 system_config 中读取登录限流状态。
func ReadRateLimitState(st *store.Store, key string, out any) error {
	raw, err := st.GetSystemConfig(key)
	if err != nil || raw == "" {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}

// WriteRateLimitState 把登录限流状态持久化到 system_config。
func WriteRateLimitState(st *store.Store, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return st.SetSystemConfig(key, string(raw))
}
