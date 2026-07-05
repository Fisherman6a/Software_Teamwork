package service

import (
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/tools"
)

const documentAsyncReportStopDirective = "A Document report job was accepted or is still running asynchronously. Do not call Document report tools again for this user request, and do not wait for completion inside this answer. Tell the user the report task has started and that the report card will continue showing progress. Do not claim the DOCX is ready unless a succeeded file artifact is already present."

func NewAgentToolResultPolicy(knowledgeMCPAlias string) agent.ToolResultPolicy {
	knowledgePolicy := NewKnowledgeRetrievalStopPolicy(knowledgeMCPAlias)
	return func(observation agent.ToolObservation) agent.ToolResultPolicyDecision {
		return mergeToolResultPolicyDecisions(
			knowledgePolicy(observation),
			DocumentAsyncReportStopPolicy(observation),
		)
	}
}

func AgentToolResultPolicy(observation agent.ToolObservation) agent.ToolResultPolicyDecision {
	return mergeToolResultPolicyDecisions(
		KnowledgeRetrievalStopPolicy(observation),
		DocumentAsyncReportStopPolicy(observation),
	)
}

func DocumentAsyncReportStopPolicy(observation agent.ToolObservation) agent.ToolResultPolicyDecision {
	if observation.Type != agent.EventToolCompleted || !tools.IsDocumentReportTool(observation.ToolName) {
		return agent.ToolResultPolicyDecision{}
	}
	artifact, ok := reportArtifactFromToolObservation(observation)
	if !ok || !isUnfinishedReportArtifact(artifact) {
		return agent.ToolResultPolicyDecision{}
	}
	decision := agent.ToolResultPolicyDecision{
		SuppressToolNames: append([]string{
			tools.ToolGenerateReportOutline,
			tools.ToolGenerateReportFromContent,
			tools.ToolGenerateReportText,
			tools.ToolGetGenerationStatus,
			tools.ToolExportReportDOCX,
			tools.ToolGetReportResult,
		}, tools.DefaultDocumentReportToolNames...),
		AppendSystemMessage: documentAsyncReportStopDirective,
	}
	if prefix := toolNamePrefix(observation.ToolName); prefix != "" {
		decision.SuppressToolPrefixes = append(decision.SuppressToolPrefixes, prefix)
	}
	return decision
}

func asyncReportCompletionMessage(observations map[string]agent.ToolObservation) (string, bool) {
	artifact, ok := latestUnfinishedReportArtifact(observations)
	if !ok {
		return "", false
	}
	if isUnfinishedReportStatus(reportArtifactString(artifact, "fileStatus")) || reportArtifactString(artifact, "jobType") == "report_file_creation" {
		return "DOCX 导出任务已创建，正在后台处理。下方卡片会继续展示导出状态，完成后可以直接下载。", true
	}
	return "报告生成任务已创建，正在后台处理。下方卡片会继续展示生成状态；生成完成后可以查看编辑，或生成并下载 DOCX。", true
}

func latestUnfinishedReportArtifact(observations map[string]agent.ToolObservation) (map[string]any, bool) {
	var selected map[string]any
	selectedIteration := -1
	for _, observation := range observations {
		if observation.Type != agent.EventToolCompleted || observation.Iteration < selectedIteration {
			continue
		}
		artifact, ok := reportArtifactFromToolObservation(observation)
		if !ok || !isUnfinishedReportArtifact(artifact) {
			continue
		}
		selected = artifact
		selectedIteration = observation.Iteration
	}
	return selected, selected != nil
}

func reportArtifactFromToolObservation(observation agent.ToolObservation) (map[string]any, bool) {
	if !tools.IsDocumentReportTool(observation.ToolName) {
		return nil, false
	}
	summary := tools.GenerateResultSummary(observation.ToolName, observation.Result)
	artifact, ok := summary["reportArtifact"].(map[string]any)
	if !ok {
		return nil, false
	}
	return artifact, true
}

func isUnfinishedReportArtifact(artifact map[string]any) bool {
	return isUnfinishedReportStatus(reportArtifactString(artifact, "jobStatus")) ||
		isUnfinishedReportStatus(reportArtifactString(artifact, "fileStatus"))
}

func isUnfinishedReportStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "accepted", "pending", "running":
		return true
	default:
		return false
	}
}

func reportArtifactString(artifact map[string]any, key string) string {
	value, _ := artifact[key].(string)
	return strings.TrimSpace(value)
}

func toolNamePrefix(toolName string) string {
	name := strings.TrimSpace(toolName)
	if before, _, ok := strings.Cut(name, "__"); ok && before != "" {
		return before + "__"
	}
	if before, _, ok := strings.Cut(name, "."); ok && before != "" {
		return before + "."
	}
	return ""
}

func mergeToolResultPolicyDecisions(decisions ...agent.ToolResultPolicyDecision) agent.ToolResultPolicyDecision {
	merged := agent.ToolResultPolicyDecision{}
	for _, decision := range decisions {
		merged.SuppressToolNames = append(merged.SuppressToolNames, decision.SuppressToolNames...)
		merged.SuppressToolPrefixes = append(merged.SuppressToolPrefixes, decision.SuppressToolPrefixes...)
		message := strings.TrimSpace(decision.AppendSystemMessage)
		if message == "" {
			continue
		}
		if merged.AppendSystemMessage == "" {
			merged.AppendSystemMessage = message
		} else {
			merged.AppendSystemMessage += "\n\n" + message
		}
	}
	return merged
}
