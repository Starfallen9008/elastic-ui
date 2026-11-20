//go:build windows

package update

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"

	"elastic-ui/internal/config"
)

func StartInstall(installer, expected string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
	dir, err := config.Dir()
	if err != nil {
		return err
	}
	failurePath := filepath.Join(dir, "updates", "last-error.txt")
	if err := os.MkdirAll(filepath.Dir(failurePath), 0700); err != nil {
		return err
	}
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$package = %s
$application = %s
$directory = %s
$failure = %s
try {
  Remove-Item -LiteralPath $failure -ErrorAction SilentlyContinue
  if ((Get-FileHash -LiteralPath $package -Algorithm SHA256).Hash -ne %s) { throw 'Update hash mismatch' }
  Wait-Process -Id %d -Timeout 45 -ErrorAction SilentlyContinue
  if (Get-Process -Id %d -ErrorAction SilentlyContinue) { throw 'Application did not exit' }
  $setup = Start-Process -FilePath $package -ArgumentList @('/S', ('/D=' + $directory)) -PassThru -Wait
  if ($setup.ExitCode -ne 0) { throw ('Installer exit code: ' + $setup.ExitCode) }
  $application = Join-Path $directory 'elastic-ui.exe'
} catch {
  $_.Exception.Message | Set-Content -LiteralPath $failure -Encoding UTF8
} finally {
  Start-Process -FilePath $application
}`, quote(installer), quote(executable), quote(filepath.Dir(executable)), quote(failurePath), quote(expected), os.Getpid(), os.Getpid())
	words := utf16.Encode([]rune(script))
	bytes := make([]byte, len(words)*2)
	for i, word := range words {
		bytes[i*2] = byte(word)
		bytes[i*2+1] = byte(word >> 8)
	}
	command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(bytes))
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return command.Start()
}
