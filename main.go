package main

import (
	"nofx/api"
	"nofx/auth"
	"nofx/bootstrap"
	"nofx/config"
	"nofx/logger"
	"nofx/manager"
	_ "nofx/mcp/payment"
	_ "nofx/mcp/provider"
	"nofx/store"
	"nofx/telegram"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger.Init(nil)
	defer logger.Shutdown()

	config.Init()
	cfg := config.Get()

	command, arg, hasCommand := parseCommand(os.Args[1:])
	if hasCommand && command == "restore-backup" {
		if arg == "" {
			logger.Fatal("restore-backup 需要提供备份目录名，例如: ./nofx restore-backup 20260424_120000_migration")
		}
		if err := bootstrap.RestoreBackup(cfg, arg); err != nil {
			logger.Fatalf("恢复备份失败: %v", err)
		}
		logger.Infof("备份恢复完成: %s", arg)
		return
	}

	if !hasCommand && len(os.Args) > 1 {
		cfg.DBPath = os.Args[1]
	}
	ensureDataDir(cfg)

	st, err := newStore(cfg)
	if err != nil {
		logger.Fatalf("初始化数据库失败: %v", err)
	}
	defer st.Close()

	securityRuntime, err := bootstrap.InitSecurity(cfg, st)
	if err != nil {
		logger.Fatalf("安全初始化失败: %v", err)
	}

	if hasCommand {
		handleCommand(command, arg, cfg, st, securityRuntime)
		return
	}

	if err := auth.Init(
		st,
		securityRuntime.OwnerID,
		securityRuntime.CryptoService.PrivateKey(),
		securityRuntime.CryptoService.PublicKey(),
	); err != nil {
		logger.Fatalf("认证运行时初始化失败: %v", err)
	}

	traderManager := manager.NewTraderManager()
	if err := traderManager.LoadUserTradersFromStore(st, securityRuntime.OwnerID); err != nil {
		logger.Fatalf("加载交易员配置失败: %v", err)
	}

	server := api.NewServer(traderManager, st, securityRuntime.CryptoService, cfg.APIServerPort)
	telegramReloadCh := make(chan struct{}, 1)
	server.SetTelegramReloadCh(telegramReloadCh)

	go startSessionCleanup(st)
	go func() {
		if err := server.Start(); err != nil {
			logger.Fatalf("启动 API 服务失败: %v", err)
		}
	}()
	go telegram.Start(cfg, st, telegramReloadCh)

	logger.Info("系统已启动，等待交易与管理请求。")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("收到退出信号，正在关闭系统。")
	traderManager.StopAll()
}

func parseCommand(args []string) (command string, arg string, ok bool) {
	if len(args) == 0 {
		return "", "", false
	}
	switch args[0] {
	case "reset-admin-key", "reset-root-key", "restore-backup":
		if len(args) > 1 {
			arg = args[1]
		}
		return args[0], arg, true
	default:
		return "", "", false
	}
}

func handleCommand(command, arg string, cfg *config.Config, st *store.Store, securityRuntime *bootstrap.SecurityRuntime) {
	_ = arg
	switch command {
	case "reset-admin-key":
		adminKey, err := bootstrap.ResetAdminKey(st, securityRuntime.CryptoService)
		if err != nil {
			logger.Fatalf("重置管理员密钥失败: %v", err)
		}
		logger.Warn("========== 新的管理员登录密钥 ==========")
		logger.Warn(adminKey)
	case "reset-root-key":
		backupDir, err := bootstrap.ResetRootKey(cfg, st, securityRuntime)
		if err != nil {
			logger.Fatalf("重置根密钥失败: %v", err)
		}
		logger.Warn("========== 根密钥轮换完成 ==========")
		logger.Warnf("请尽快备份新的根密钥目录: %s", cfg.KeysDir)
		logger.Warnf("如需回滚，可执行: ./nofx restore-backup %s", filepath.Base(backupDir))
	default:
		logger.Fatalf("未知命令: %s", command)
	}
}

func ensureDataDir(cfg *config.Config) {
	if cfg.DBType != "sqlite" {
		return
	}
	if dir := filepath.Dir(cfg.DBPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logger.Warnf("创建数据目录失败: %v", err)
		}
	}
}

func newStore(cfg *config.Config) (*store.Store, error) {
	dbType := store.DBTypeSQLite
	if cfg.DBType == "postgres" {
		dbType = store.DBTypePostgres
	}
	return store.NewWithConfig(store.DBConfig{
		Type:     dbType,
		Path:     cfg.DBPath,
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	})
}

// startSessionCleanup 定期清理已过期或已吊销的持久化会话，防止表无限膨胀。
func startSessionCleanup(st *store.Store) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	_ = st.AuthSession().DeleteExpired(time.Now().UTC())
	for range ticker.C {
		if err := st.AuthSession().DeleteExpired(time.Now().UTC()); err != nil {
			logger.Warnf("清理过期会话失败: %v", err)
		}
	}
}
