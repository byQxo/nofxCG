//go:build windows

package bootstrap

import (
	"fmt"
	"os/exec"
	"os/user"
)

// securePathPermissions 在 Windows 上通过 icacls 收紧 ACL。
// 目标是仅允许当前用户访问根密钥目录和文件，避免默认继承权限过宽。
func securePathPermissions(path string, isDir bool) error {
	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	grant := fmt.Sprintf("%s:(F)", currentUser.Username)
	if !isDir {
		grant = fmt.Sprintf("%s:(R,W)", currentUser.Username)
	}

	cmd := exec.Command("icacls", path, "/inheritance:r", "/grant:r", grant)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to apply ACL on %s: %w (%s)", path, err, string(output))
	}
	return nil
}
