package main

import (
	"strings"
	"testing"

	"m31labs.dev/arbiter/govern"
	"m31labs.dev/gosx3d-studio/internal/studio"
)

func mustWebMCPPolicy(t *testing.T) *webMCPOperationPolicy {
	t.Helper()
	policy, err := compileWebMCPOperationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestWebMCPOperationPolicyAllowsOnlyTheBrowserSurface(t *testing.T) {
	policy, err := compileWebMCPOperationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[studio.OperationKind]bool{
		studio.OpRenameEntity:   true,
		studio.OpSetTransform:   true,
		studio.OpAssignMaterial: true,
	}
	for _, capability := range studio.ActionCatalog() {
		kind := studio.OperationKind(capability.ID)
		decision, err := policy.Decide(kind)
		if err != nil {
			t.Fatalf("decide %q: %v", kind, err)
		}
		if decision.Allowed != allowed[kind] {
			t.Errorf("decision for %q allowed=%t selected=%q, want allowed=%t", kind, decision.Allowed, decision.Selected, allowed[kind])
		}
	}
	for _, kind := range []studio.OperationKind{"", "future-agent-operation"} {
		decision, err := policy.Decide(kind)
		if err != nil {
			t.Fatalf("decide %q: %v", kind, err)
		}
		if decision.Allowed || decision.Selected != "Deny" {
			t.Errorf("decision for %q = %+v, want Deny", kind, decision)
		}
	}
}

func TestWebMCPOperationPolicyProducesExplainableAllowAndDenyTraces(t *testing.T) {
	policy, err := compileWebMCPOperationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	allow, err := policy.Decide(studio.OpRenameEntity)
	if err != nil {
		t.Fatal(err)
	}
	if allow.Selected != "Allow" || len(allow.Arbitrace.Steps) == 0 {
		t.Fatalf("allow decision has no trace: %+v", allow)
	}
	if last := allow.Arbitrace.Steps[len(allow.Arbitrace.Steps)-1]; last.Disposition != govern.ArbitraceDispositionPassed {
		t.Fatalf("allow trace ends with %+v, want passed", last)
	}

	deny, err := policy.Decide(studio.OpDeleteEntity)
	if err != nil {
		t.Fatal(err)
	}
	if deny.Selected != "Deny" || len(deny.Arbitrace.Steps) < 2 {
		t.Fatalf("deny decision has no fallthrough trace: %+v", deny)
	}
	var blocked, fallback bool
	for _, step := range deny.Arbitrace.Steps {
		blocked = blocked || step.Disposition == govern.ArbitraceDispositionBlocked
		fallback = fallback || (strings.Contains(step.Check, ":fallback") && step.Disposition == govern.ArbitraceDispositionPassed)
	}
	if !blocked || !fallback {
		t.Fatalf("deny trace = %+v, want blocked condition followed by passed fallback", deny.Arbitrace.Steps)
	}
}

func TestWebMCPStageEnforcesPolicyBeforeCanonicalExecution(t *testing.T) {
	workspace, err := studio.NewWorkspace(studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	store := newWebMCPProposalStore(workspace, mustWebMCPPolicy(t))
	before, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Stage(webMCPProposalRequest{
		ExpectedRevision: before.Revision,
		Title:            "Destructive proposal",
		Operations:       []studio.Operation{{Kind: studio.OpDeleteEntity, Target: "board"}},
	}, "session-a")
	if err == nil || !strings.Contains(err.Error(), "denied by WebMCP policy") {
		t.Fatalf("destructive Stage error = %v, want policy denial", err)
	}
	after, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != before.Revision || after.Entities["board"].Name != before.Entities["board"].Name {
		t.Fatal("denied operation changed canonical scene state")
	}
}
