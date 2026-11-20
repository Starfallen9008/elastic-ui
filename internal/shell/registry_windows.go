//go:build windows

package shell

import (
	"errors"
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	fileVerbKey = `Software\Classes\*\shell\ElasticDefendScan`
	dirVerbKey  = `Software\Classes\Directory\shell\ElasticDefendScan`
	bgVerbKey   = `Software\Classes\Directory\Background\shell\ElasticDefendScan`
)

func exePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return exe
}

func SetEnabled(enabled bool) error {
	if !enabled {
		return unregister()
	}
	exe := exePath()
	if exe == "" {
		return errors.New("鏃犳硶鑾峰彇褰撳墠绋嬪簭璺緞")
	}
	command := `"` + exe + `" --scan "%1"`
	backgroundCommand := `"` + exe + `" --scan "%V"`
	if err := writeVerb(fileVerbKey, exe, command); err != nil {
		return err
	}
	if err := writeVerb(dirVerbKey, exe, command); err != nil {
		return err
	}
	return writeVerb(bgVerbKey, exe, backgroundCommand)
}

func IsEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, fileVerbKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue("")
	return err == nil && value != ""
}

func writeVerb(path, exe, command string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.SetStringValue("", "浣跨敤 Elastic Defend 杩涜鎵弿"); err != nil {
		return err
	}
	if err := key.SetStringValue("Icon", exe); err != nil {
		return err
	}
	if err := key.SetStringValue("MultiSelectModel", "Player"); err != nil {
		return err
	}
	commandKey, _, err := registry.CreateKey(registry.CURRENT_USER, path+`\command`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer commandKey.Close()
	return commandKey.SetStringValue("", command)
}

func unregister() error {
	for _, path := range []string{fileVerbKey, dirVerbKey, bgVerbKey} {
		_ = registry.DeleteKey(registry.CURRENT_USER, path+`\command`)
		_ = registry.DeleteKey(registry.CURRENT_USER, path)
	}
	return nil
}

