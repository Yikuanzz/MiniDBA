package sqlrun

import (
	"fmt"
	"regexp"
	"strings"
)

var tableNameRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// ValidateTableName 允许浏览/元数据的安全表名。
func ValidateTableName(name string) error {
	if !tableNameRe.MatchString(name) {
		return fmt.Errorf("非法表名")
	}
	return nil
}

// CheckBlacklist 危险语句（始终拒绝）。
func CheckBlacklist(sql string) error {
	s := strings.ToLower(strings.TrimSpace(sql))
	if strings.Contains(s, "drop database") {
		return fmt.Errorf("禁止: DROP DATABASE")
	}
	if strings.Contains(s, "truncate") {
		return fmt.Errorf("禁止: TRUNCATE")
	}
	return nil
}

var writeKeywords = []string{
	"insert ", "update ", "delete ", "replace ",
	"create ", "alter ", "drop ", "rename ", "truncate ",
	"grant ", "revoke ", "call ",
}

// CheckReadonly 在只读模式下拒绝写与 DDL。
func CheckReadonly(sql string) error {
	s := " " + strings.ToLower(strings.TrimSpace(sql)) + " "
	for _, kw := range writeKeywords {
		if strings.Contains(s, kw) {
			return fmt.Errorf("只读模式禁止该语句")
		}
	}
	return nil
}

// IsQueryPath 使用 Query 而非 Exec 的前缀（粗粒度）。
func IsQueryPath(sql string) bool {
	s := strings.TrimSpace(sql)
	if s == "" {
		return false
	}
	u := strings.ToUpper(s)
	return strings.HasPrefix(u, "SELECT") ||
		strings.HasPrefix(u, "WITH") ||
		strings.HasPrefix(u, "SHOW") ||
		strings.HasPrefix(u, "DESCRIBE") ||
		strings.HasPrefix(u, "DESC ") ||
		strings.HasPrefix(u, "EXPLAIN")
}
