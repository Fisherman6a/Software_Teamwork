package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationsEnableDocumentReportToolsForUntouchedSystemDefault(t *testing.T) {
	matches, err := filepath.Glob("../../migrations/*document*report*tool*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("missing migration for enabling document report tools in untouched system default QA config")
	}
	var content string
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		content += "\n" + string(data)
	}
	required := []string{
		"-- +goose Up",
		"UPDATE qa_config_versions",
		"created_by_user_id = 'system'",
		"version_no = 1",
		`enabled_tool_names = '["search_knowledge", "search_session_attachments"]'::jsonb`,
		"document__generate_report_from_content",
		"document__generate_report_outline",
		"document__generate_report_text",
		"document__get_generation_status",
		"document__export_report_docx",
		"document__get_report_result",
	}
	for _, token := range required {
		if !strings.Contains(content, token) {
			t.Fatalf("document report tool migration missing %q\n%s", token, content)
		}
	}
	if strings.Contains(content, "created_by_user_id <>") || strings.Contains(content, "created_by_user_id !=") {
		t.Fatalf("document report tool migration must not update user-created configs:\n%s", content)
	}
}

func TestDomainRefusalPromptMigrationCoversLegacyDefaults(t *testing.T) {
	data, err := os.ReadFile("../../migrations/0019_domain_refusal_system_prompt.sql")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	required := []string{
		"-- +goose Up",
		"Answer in the same language as the user''s question.",
		"Answer only questions related to the power industry",
		"politely refuse in the user''s language",
		`"write bubble sort"`,
		"do not call retrieval, attachment, or document tools",
		"leave knowledgeBaseIds empty to search all indexed knowledge bases.\nIf knowledge__search is unavailable",
		"leave knowledgeBaseIds empty to search all indexed knowledge bases.\nAfter knowledge__search or knowledge__get_chunk returns relevant results",
		"WHERE key = 'system_prompt'\n  AND value IN (",
		"WHERE system_prompt IN (",
	}
	for _, token := range required {
		if !strings.Contains(content, token) {
			t.Fatalf("domain-refusal prompt migration missing %q\n%s", token, content)
		}
	}
	if strings.Contains(content, strings.Repeat("?", 5)) {
		t.Fatalf("domain-refusal prompt migration contains placeholder question marks:\n%s", content)
	}
}
