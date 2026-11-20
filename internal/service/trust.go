package service

import (
	"errors"
	"regexp"
	"strings"

	"elastic-ui/internal/kibana"
)

var trustHash = regexp.MustCompile(`(?i)^[a-f0-9]{64}$`)
var trustPath = regexp.MustCompile(`(?i)^([a-z]:[\\/]|\\\\[^\\/]+[\\/][^\\/]+[\\/])`)

func validateTrustInput(kind, value string) error {
	if kind == "hash" {
		if !trustHash.MatchString(value) {
			return errors.New("SHA256 必须是 64 位十六进制字符")
		}
		return nil
	}
	if kind != "path" && kind != "wildcard" {
		return errors.New("不支持的信任条件类型")
	}
	if !trustPath.MatchString(value) {
		return errors.New("请输入 Windows 绝对路径或 UNC 路径")
	}
	for _, character := range value {
		if character < 32 || strings.ContainsRune(`"<>|`, character) {
			return errors.New("路径含不支持的字符")
		}
	}
	if strings.Contains(value[2:], ":") {
		return errors.New("路径含不支持的字符")
	}
	hasWildcard := strings.ContainsAny(value, "*?")
	if kind == "wildcard" && !hasWildcard {
		return errors.New("路径通配至少需要一个 * 或 ?")
	}
	if kind == "path" && hasWildcard {
		return errors.New("含通配符的路径请选择路径通配类型")
	}
	if kind == "path" && strings.ContainsAny(value[len(value)-1:], `\/`) {
		return errors.New("请指定可执行文件，不支持目录排除")
	}
	return nil
}

func trustRuleKey(kind, value string) string {
	return kind + ":" + strings.ToLower(strings.TrimSpace(value))
}

func trustAppliesToPolicy(item kibana.ExceptionListItem, policyID string) bool {
	for _, tag := range item.Tags {
		if tag == "policy:all" || tag == kibana.PolicyTag(policyID) {
			return true
		}
	}
	return false
}

// Only equivalent single-condition Windows rules can be skipped as duplicates.
// A path plus a signer/hash condition is not the same trust as a path alone.
func trustedAppMatchKey(item kibana.ExceptionListItem) string {
	if len(item.Entries) != 1 {
		return ""
	}
	windows := false
	for _, os := range item.OSTypes {
		windows = windows || os == "windows"
	}
	if !windows {
		return ""
	}
	entry := item.Entries[0]
	value, ok := entry.Value.(string)
	if !ok || entry.Operator != "included" {
		return ""
	}
	if entry.Field == "process.hash.sha256" && entry.Type == "match" {
		return trustRuleKey("hash", value)
	}
	if entry.Field != "process.executable.caseless" {
		return ""
	}
	switch entry.Type {
	case "match":
		return trustRuleKey("path", value)
	case "wildcard":
		return trustRuleKey("wildcard", value)
	default:
		return ""
	}
}
