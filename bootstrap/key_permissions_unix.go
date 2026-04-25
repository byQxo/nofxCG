//go:build !windows

package bootstrap

import (
	"os"
)

// securePathPermissions 在 Unix 系统上收紧密钥目录与文件权限。
func securePathPermissions(path string, isDir bool) error {
	if isDir {
		return os.Chmod(path, 0o700)
	}
	return os.Chmod(path, 0o600)
}
