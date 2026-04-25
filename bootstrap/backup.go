package bootstrap

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"nofx/config"
)

// CreateBackup 在密钥轮换、迁移前创建本地快照。
// 这里只复制本地文件系统中的数据库、密钥目录和遗留配置文件，避免操作失败后无法回滚。
func CreateBackup(cfg *config.Config, label string) (string, error) {
	timestamp := time.Now().UTC().Format("20060102_150405")
	if label == "" {
		label = "manual"
	}
	backupDir := filepath.Join(cfg.BackupDir, fmt.Sprintf("%s_%s", timestamp, label))
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", err
	}

	if cfg.DBType == "sqlite" {
		if _, err := os.Stat(cfg.DBPath); err == nil {
			target := filepath.Join(backupDir, "data.db")
			if err := copyFile(cfg.DBPath, target, 0o600); err != nil {
				return "", err
			}
		}
	}

	if _, err := os.Stat(cfg.KeysDir); err == nil {
		if err := copyDir(cfg.KeysDir, filepath.Join(backupDir, "keys")); err != nil {
			return "", err
		}
	}

	for _, candidate := range []string{".env", "config.json", "configbak.json"} {
		if _, err := os.Stat(candidate); err == nil {
			if err := copyFile(candidate, filepath.Join(backupDir, filepath.Base(candidate)), 0o600); err != nil {
				return "", err
			}
		}
	}

	return backupDir, nil
}

// RestoreBackup 从指定时间戳目录恢复数据库、密钥和配置快照。
func RestoreBackup(cfg *config.Config, backupName string) error {
	backupDir := filepath.Join(cfg.BackupDir, backupName)
	info, err := os.Stat(backupDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("backup %s is not a directory", backupName)
	}

	if cfg.DBType == "sqlite" {
		backupDB := filepath.Join(backupDir, "data.db")
		if _, err := os.Stat(backupDB); err == nil {
			if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
				return err
			}
			if err := copyFile(backupDB, cfg.DBPath, 0o600); err != nil {
				return err
			}
		}
	}

	backupKeys := filepath.Join(backupDir, "keys")
	if _, err := os.Stat(backupKeys); err == nil {
		if err := os.RemoveAll(cfg.KeysDir); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := copyDir(backupKeys, cfg.KeysDir); err != nil {
			return err
		}
	}

	for _, candidate := range []string{".env", "config.json", "configbak.json"} {
		src := filepath.Join(backupDir, filepath.Base(candidate))
		if _, err := os.Stat(src); err == nil {
			if err := copyFile(src, candidate, 0o600); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		mode := os.FileMode(0o600)
		if info, err := entry.Info(); err == nil {
			mode = info.Mode().Perm()
		}
		if err := copyFile(srcPath, dstPath, mode); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
