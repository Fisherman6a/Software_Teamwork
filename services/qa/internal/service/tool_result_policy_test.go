package service

import (
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
)

func TestDocumentAsyncReportStopPolicySuppressesDocumentTools(t *testing.T) {
	decision := DocumentAsyncReportStopPolicy(agent.ToolObservation{
		Type:     agent.EventToolCompleted,
		ToolName: "document__get_generation_status",
		Result:   `{"status":"succeeded","job":{"id":"job-1","reportId":"rpt-1","jobType":"outline_generation","status":"running"}}`,
	})

	if !containsPolicyString(decision.SuppressToolNames, "document__get_generation_status") {
		t.Fatalf("expected prefixed get_generation_status to be suppressed: %+v", decision)
	}
	if !containsPolicyString(decision.SuppressToolPrefixes, "document__") {
		t.Fatalf("expected document prefix to be suppressed: %+v", decision)
	}
	if !strings.Contains(decision.AppendSystemMessage, "Do not call Document report tools again") {
		t.Fatalf("unexpected directive: %q", decision.AppendSystemMessage)
	}
}

func TestDocumentAsyncReportStopPolicyIgnoresSucceededArtifact(t *testing.T) {
	decision := DocumentAsyncReportStopPolicy(agent.ToolObservation{
		Type:     agent.EventToolCompleted,
		ToolName: "document__get_generation_status",
		Result:   `{"status":"succeeded","job":{"id":"job-1","reportId":"rpt-1","jobType":"outline_generation","status":"succeeded"}}`,
	})

	if len(decision.SuppressToolNames) != 0 || decision.AppendSystemMessage != "" {
		t.Fatalf("succeeded artifact should not stop document tools: %+v", decision)
	}
}

func containsPolicyString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
