package main

import (
	_ "embed"
	"fmt"

	arbiter "m31labs.dev/arbiter"
	"m31labs.dev/arbiter/govern"
	"m31labs.dev/gosx3d-studio/internal/studio"
)

const webMCPOperationStrategy = "DecideWebMCPOperation"

//go:embed internal/studio/rules/webmcp-operations.arb
var webMCPOperationPolicySource []byte

// webMCPOperationDecision is both the authority result and its evidence. The
// trace makes an allow or denial inspectable without moving operation-shape
// validation out of Go's typed request boundary.
type webMCPOperationDecision struct {
	Kind      studio.OperationKind `json:"kind"`
	Allowed   bool                 `json:"allowed"`
	Selected  string               `json:"selected"`
	Reason    string               `json:"reason"`
	Arbitrace govern.Arbitrace     `json:"arbitrace"`
}

type webMCPOperationPolicy struct {
	program *arbiter.Program
}

func compileWebMCPOperationPolicy() (*webMCPOperationPolicy, error) {
	program, err := arbiter.Compile(webMCPOperationPolicySource)
	if err != nil {
		return nil, err
	}
	if program.Strategies == nil ||
		!program.Strategies.Has(webMCPOperationStrategy) ||
		!program.Strategies.HasCandidate(webMCPOperationStrategy, "Allow") ||
		!program.Strategies.HasCandidate(webMCPOperationStrategy, "Deny") {
		return nil, fmt.Errorf("WebMCP operation strategy is incomplete")
	}
	return &webMCPOperationPolicy{program: program}, nil
}

func (policy *webMCPOperationPolicy) Decide(kind studio.OperationKind) (webMCPOperationDecision, error) {
	if policy == nil || policy.program == nil || policy.program.Strategies == nil {
		return webMCPOperationDecision{}, fmt.Errorf("WebMCP operation policy is unavailable")
	}
	result, err := policy.program.Strategies.Evaluate(webMCPOperationStrategy, map[string]any{
		"operation": map[string]any{"kind": string(kind)},
	})
	if err != nil {
		return webMCPOperationDecision{}, err
	}
	allowed, allowedOK := result.Params["allowed"].(bool)
	reason, reasonOK := result.Params["reason"].(string)
	if !allowedOK || !reasonOK || reason == "" {
		return webMCPOperationDecision{}, fmt.Errorf("WebMCP operation strategy returned invalid decision parameters")
	}
	if result.Selected != "Allow" && result.Selected != "Deny" {
		return webMCPOperationDecision{}, fmt.Errorf("WebMCP operation strategy selected unexpected candidate %q", result.Selected)
	}
	if allowed != (result.Selected == "Allow") {
		return webMCPOperationDecision{}, fmt.Errorf("WebMCP operation strategy returned an inconsistent decision")
	}
	return webMCPOperationDecision{
		Kind:      kind,
		Allowed:   allowed,
		Selected:  result.Selected,
		Reason:    reason,
		Arbitrace: result.Arbitrace,
	}, nil
}
