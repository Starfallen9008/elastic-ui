//go:build windows

package shell

import (
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

func startupTaskName() (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", err
	}
	return "ElasticDefendConsole-" + user.User.Sid.String(), nil
}

func taskCommand(args ...string) ([]byte, error) {
	command := exec.Command("schtasks.exe", args...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return command.CombinedOutput()
}

func StartupEnabled() (bool, error) {
	name, err := startupTaskName()
	if err != nil {
		return false, err
	}
	output, err := taskCommand("/Query", "/TN", name, "/XML")
	if err != nil {
		// Query failure can mean a missing task. Use the task file to distinguish it
		// from an inaccessible scheduler rather than showing an incorrect state.
		_, statErr := os.Stat(filepath.Join(os.Getenv("WINDIR"), "System32", "Tasks", name))
		if errors.Is(statErr, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("读取开机自启：%s", strings.TrimSpace(string(output)))
	}
	text := string(output)
	if len(output) > 1 && (output[0] == 0xff || output[1] == 0) {
		words := make([]uint16, 0, len(output)/2)
		for i := 0; i+1 < len(output); i += 2 {
			words = append(words, binary.LittleEndian.Uint16(output[i:]))
		}
		text = string(utf16.Decode(words))
	}
	var definition struct {
		Settings struct {
			Enabled string `xml:"Enabled"`
		} `xml:"Settings"`
		Actions struct {
			Exec struct {
				Arguments string `xml:"Arguments"`
			} `xml:"Exec"`
		} `xml:"Actions"`
	}
	text = strings.TrimPrefix(text, "\ufeff")
	// schtasks declares UTF-16 even after its output has been decoded.
	if end := strings.Index(text, "?>"); end >= 0 {
		text = text[end+2:]
	}
	if err := xml.Unmarshal([]byte(text), &definition); err != nil {
		return false, fmt.Errorf("读取开机自启：%w", err)
	}
	return definition.Settings.Enabled != "false" && definition.Actions.Exec.Arguments == "--autostart", nil
}

func SetStartupEnabled(enabled bool) error {
	name, err := startupTaskName()
	if err != nil {
		return err
	}
	if !enabled {
		active, err := StartupEnabled()
		if err != nil || !active {
			return err
		}
		output, err := taskCommand("/Delete", "/TN", name, "/F")
		if err != nil {
			return fmt.Errorf("关闭开机自启：%s", strings.TrimSpace(string(output)))
		}
		return nil
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	path, err := os.Executable()
	if err != nil {
		return err
	}
	escape := func(value string) string {
		var text strings.Builder
		_ = xml.EscapeText(&text, []byte(value))
		return text.String()
	}
	sid := user.User.Sid.String()
	definition := `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <Triggers><LogonTrigger><Enabled>true</Enabled><UserId>` + sid + `</UserId></LogonTrigger></Triggers>
  <Principals><Principal id="Author"><UserId>` + sid + `</UserId><LogonType>InteractiveToken</LogonType><RunLevel>HighestAvailable</RunLevel></Principal></Principals>
  <Settings><MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy><DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries><StopIfGoingOnBatteries>false</StopIfGoingOnBatteries><ExecutionTimeLimit>PT0S</ExecutionTimeLimit><Enabled>true</Enabled></Settings>
  <Actions Context="Author"><Exec><Command>` + escape(path) + `</Command><Arguments>--autostart</Arguments><WorkingDirectory>` + escape(filepath.Dir(path)) + `</WorkingDirectory></Exec></Actions>
</Task>`
	file, err := os.CreateTemp("", "elastic-startup-*.xml")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	words := utf16.Encode([]rune(definition))
	encoded := make([]byte, 2+len(words)*2)
	encoded[0], encoded[1] = 0xff, 0xfe
	for i, word := range words {
		binary.LittleEndian.PutUint16(encoded[2+i*2:], word)
	}
	_, writeErr := file.Write(encoded)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	output, err := taskCommand("/Create", "/TN", name, "/XML", file.Name(), "/F")
	if err != nil {
		return fmt.Errorf("设置开机自启：%s", strings.TrimSpace(string(output)))
	}
	return nil
}
