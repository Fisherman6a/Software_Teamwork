package tools

import (
	"strings"
	"testing"
)

func TestGenerateResultSummaryPreservesSanitizedToolFailure(t *testing.T) {
	summary := GenerateResultSummary(ToolSearchKnowledge, `{"error":{"code":"retrieval_failed","message":"knowledge retrieval service failed"}}`)

	if summary["error"] != "retrieval_failed" || summary["message"] != "knowledge retrieval service failed" || summary["sanitized"] != true {
		t.Fatalf("summary=%#v", summary)
	}
	if strings.Contains(strings.TrimSpace(summary["message"].(string)), "internal") {
		t.Fatalf("summary leaked raw marker: %#v", summary)
	}
}

func TestGenerateResultSummaryRedactsExternalToolFailureMessage(t *testing.T) {
	summary := GenerateResultSummary("external.lookup", `{"error":{"code":"upstream_failed","message":"dial tcp internal.service.local with token secret"}}`)

	if summary["error"] != "upstream_failed" || summary["message"] != "tool execution failed" || summary["sanitized"] != true {
		t.Fatalf("summary=%#v", summary)
	}
	if strings.Contains(strings.TrimSpace(summary["message"].(string)), "internal") || strings.Contains(strings.TrimSpace(summary["message"].(string)), "secret") {
		t.Fatalf("summary leaked raw external failure: %#v", summary)
	}
}
