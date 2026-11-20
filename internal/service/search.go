package service

import "strings"

// searchClause 用文本短语加关键字通配的方式做子串搜索。
// 避免 simple_query_string 把用户输入的 ../、<script> 当成查询语法。
func searchClause(text string, textFields, termFields, wildcardFields []string) map[string]any {
	query := strings.TrimSpace(text)
	escaped := strings.NewReplacer(`\`, `\\`, `*`, `\*`, `?`, `\?`).Replace(query)

	should := make([]any, 0, len(textFields)+len(termFields)+len(wildcardFields))
	if len(textFields) > 0 {
		should = append(should, map[string]any{"multi_match": map[string]any{
			"query":  query,
			"fields": textFields,
			"type":   "phrase",
		}})
	}
	for _, field := range termFields {
		should = append(should, map[string]any{"term": map[string]any{field: query}})
	}
	for _, field := range wildcardFields {
		should = append(should, map[string]any{"wildcard": map[string]any{
			field: map[string]any{
				"value":            "*" + escaped + "*",
				"case_insensitive": true,
			},
		}})
	}
	return map[string]any{"bool": map[string]any{
		"should":               should,
		"minimum_should_match": 1,
	}}
}
