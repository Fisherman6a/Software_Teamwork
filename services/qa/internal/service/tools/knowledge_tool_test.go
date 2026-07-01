package tools

import (
	"context"
	"testing"
)

type listToolsRetrieverStub struct{}

func (listToolsRetrieverStub) Retrieve(context.Context, string, RetrievalTestInput) ([]RetrievalTestResult, error) {
	return nil, nil
}

func TestKnowledgeToolClientListsCitationSourceTool(t *testing.T) {
	client, err := NewKnowledgeToolClient(KnowledgeToolConfig{RetrievalClient: listToolsRetrieverStub{}})
	if err != nil {
		t.Fatal(err)
	}

	definitions, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, definition := range definitions {
		names[definition.Function.Name] = true
	}

	if !names[ToolSearchKnowledge] || !names[ToolGetCitationSource] {
		t.Fatalf("tool names=%#v, want search and citation source tools", names)
	}
}
