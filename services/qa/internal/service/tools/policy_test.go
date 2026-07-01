package tools

import (
	"fmt"
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

func TestGenerateResultSummaryBuildsPendingReportArtifact(t *testing.T) {
	summary := GenerateResultSummary("document__generate_report_text", `{"status":"accepted","job":{"id":"job-1","reportId":"rpt-1","jobType":"content_generation","status":"running","progress":{"percent":42,"internalUrl":"http://internal/doc"}}}`)

	artifact := summary["reportArtifact"].(map[string]any)
	if artifact["artifactType"] != "report_generation" || artifact["jobId"] != "job-1" || artifact["jobStatus"] != "running" {
		t.Fatalf("artifact=%#v", artifact)
	}
	if _, ok := artifact["downloadPath"]; ok {
		t.Fatalf("pending artifact must not expose downloadPath: %#v", artifact)
	}
	preview := artifact["preview"].(map[string]any)
	if preview["progressPercent"] != 42 {
		t.Fatalf("preview=%#v", preview)
	}
	if strings.Contains(strings.ToLower(fmtSprint(summary)), "internalurl") || strings.Contains(strings.ToLower(fmtSprint(summary)), "http://internal") {
		t.Fatalf("summary leaked internal progress fields: %#v", summary)
	}
}

func TestGenerateResultSummaryBuildsSucceededExportArtifact(t *testing.T) {
	summary := GenerateResultSummary("document__export_report_docx", `{"status":"accepted","reportFile":{"id":"rf-1","reportId":"rpt-1","jobId":"job-2","filename":"report.docx","format":"docx","fileSize":4096,"status":"succeeded"}}`)

	artifact := summary["reportArtifact"].(map[string]any)
	if artifact["reportFileId"] != "rf-1" || artifact["fileStatus"] != "succeeded" || artifact["downloadPath"] != "/api/v1/report-files/rf-1/content" {
		t.Fatalf("artifact=%#v", artifact)
	}
	if artifact["detailPath"] != "/api/v1/reports/rpt-1" {
		t.Fatalf("detailPath=%#v", artifact["detailPath"])
	}
}

func TestGenerateResultSummaryBuildsSanitizedDocumentFailure(t *testing.T) {
	summary := GenerateResultSummary("document__get_report_result", `{"status":"failed","error":{"code":"forbidden","message":"token secret at http://internal/document"}}`)

	if summary["error"] != "policy_denied" || summary["message"] != "report access is not allowed" {
		t.Fatalf("summary=%#v", summary)
	}
	artifact := summary["reportArtifact"].(map[string]any)
	if artifact["jobStatus"] != "failed" {
		t.Fatalf("artifact=%#v", artifact)
	}
	if strings.Contains(strings.ToLower(fmtSprint(summary)), "token secret") || strings.Contains(strings.ToLower(fmtSprint(summary)), "http://internal") {
		t.Fatalf("summary leaked raw document failure: %#v", summary)
	}
}

func fmtSprint(value any) string {
	return fmt.Sprintf("%#v", value)
}
