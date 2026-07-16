package studio

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"sync"

	arbiter "m31labs.dev/arbiter"
)

type RetargetMap struct {
	ID     ID        `json:"id"`
	Name   string    `json:"name"`
	Source ID        `json:"source"`
	Target ID        `json:"target"`
	Bones  map[ID]ID `json:"bones"`
	Scale  float64   `json:"scale,omitempty"`
}

type AnimationStateMachine struct {
	ID          ID                    `json:"id"`
	Name        string                `json:"name"`
	Initial     ID                    `json:"initial"`
	Current     ID                    `json:"current"`
	StateTime   float64               `json:"stateTime"`
	Parameters  map[string]float64    `json:"parameters"`
	States      map[ID]AnimationState `json:"states"`
	Transitions []AnimationTransition `json:"transitions,omitempty"`
}

type AnimationState struct {
	ID    ID      `json:"id"`
	Name  string  `json:"name"`
	Clip  ID      `json:"clip"`
	Speed float64 `json:"speed"`
}

type AnimationTransition struct {
	ID        ID      `json:"id"`
	From      ID      `json:"from"`
	To        ID      `json:"to"`
	Parameter string  `json:"parameter"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
	ExitTime  float64 `json:"exitTime,omitempty"`
	Priority  int     `json:"priority,omitempty"`
}

type TransitionTrace struct {
	Transition ID      `json:"transition"`
	Rule       string  `json:"rule,omitempty"`
	Eligible   bool    `json:"eligible"`
	Parameter  string  `json:"parameter"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	StateTime  float64 `json:"stateTime"`
}

type MachineStepResult struct {
	Machine    ID                `json:"machine"`
	From       ID                `json:"from"`
	To         ID                `json:"to"`
	Transition ID                `json:"transition,omitempty"`
	StateTime  float64           `json:"stateTime"`
	Clip       ID                `json:"clip"`
	Sample     AnimationSample   `json:"sample"`
	Trace      []TransitionTrace `json:"trace"`
}

//go:embed rules/animation-transitions.arb
var animationTransitionRules []byte

var transitionProgram struct {
	sync.Once
	program *arbiter.Program
	err     error
}

func validateRetargetMap(value RetargetMap, armatures map[ID]Armature) error {
	if value.ID == "" || strings.TrimSpace(value.Name) == "" || value.Source == "" || value.Target == "" || len(value.Bones) == 0 || !finite(value.Scale) || value.Scale < 0 {
		return fmt.Errorf("id, name, source, target, non-negative scale, and bone mappings are required")
	}
	source, sourceOK := armatures[value.Source]
	target, targetOK := armatures[value.Target]
	if !sourceOK || !targetOK {
		return fmt.Errorf("source or target armature does not exist")
	}
	seenTargets := map[ID]bool{}
	for sourceID, targetID := range value.Bones {
		if _, ok := source.Bones[sourceID]; !ok {
			return fmt.Errorf("source bone %q does not exist", sourceID)
		}
		if _, ok := target.Bones[targetID]; !ok || seenTargets[targetID] {
			return fmt.Errorf("target bone %q is missing or mapped twice", targetID)
		}
		seenTargets[targetID] = true
	}
	return nil
}

func validateAnimationMachine(machine AnimationStateMachine, clips map[ID]AnimationClip) error {
	if machine.ID == "" || strings.TrimSpace(machine.Name) == "" || len(machine.States) == 0 || machine.Initial == "" {
		return fmt.Errorf("id, name, initial state, and states are required")
	}
	if machine.Current == "" {
		machine.Current = machine.Initial
	}
	if _, ok := machine.States[machine.Initial]; !ok {
		return fmt.Errorf("initial state %q does not exist", machine.Initial)
	}
	if _, ok := machine.States[machine.Current]; !ok || !finite(machine.StateTime) || machine.StateTime < 0 {
		return fmt.Errorf("current state or state time is invalid")
	}
	for name, value := range machine.Parameters {
		if strings.TrimSpace(name) == "" || !finite(value) {
			return fmt.Errorf("invalid parameter %q", name)
		}
	}
	for id, state := range machine.States {
		if id == "" || state.ID != id || strings.TrimSpace(state.Name) == "" || !finite(state.Speed) || state.Speed <= 0 {
			return fmt.Errorf("invalid state %q", id)
		}
		if _, ok := clips[state.Clip]; !ok {
			return fmt.Errorf("state %q references missing clip %q", id, state.Clip)
		}
	}
	seen := map[ID]bool{}
	for _, transition := range machine.Transitions {
		if transition.ID == "" || seen[transition.ID] || transition.From == transition.To || !finite(transition.Threshold) || !finite(transition.ExitTime) || transition.ExitTime < 0 {
			return fmt.Errorf("invalid or duplicate transition %q", transition.ID)
		}
		seen[transition.ID] = true
		if _, ok := machine.States[transition.From]; !ok {
			return fmt.Errorf("transition %q missing source state", transition.ID)
		}
		if _, ok := machine.States[transition.To]; !ok {
			return fmt.Errorf("transition %q missing target state", transition.ID)
		}
		if _, ok := machine.Parameters[transition.Parameter]; !ok {
			return fmt.Errorf("transition %q missing parameter %q", transition.ID, transition.Parameter)
		}
		switch transition.Operator {
		case "equal", "not-equal", "greater", "greater-equal", "less", "less-equal":
		default:
			return fmt.Errorf("transition %q has unsupported operator %q", transition.ID, transition.Operator)
		}
	}
	return nil
}

func RetargetAnimationClip(document Document, mapID, clipID, newID ID, name string) (AnimationClip, error) {
	mapping, ok := document.RetargetMaps[mapID]
	if !ok {
		return AnimationClip{}, fmt.Errorf("retarget map %q does not exist", mapID)
	}
	clip, ok := document.Animations[clipID]
	if !ok {
		return AnimationClip{}, fmt.Errorf("animation %q does not exist", clipID)
	}
	if newID == "" || strings.TrimSpace(name) == "" {
		return AnimationClip{}, fmt.Errorf("retargeted clip id and name are required")
	}
	if _, exists := document.Animations[newID]; exists {
		return AnimationClip{}, fmt.Errorf("animation %q already exists", newID)
	}
	source, target := document.Armatures[mapping.Source], document.Armatures[mapping.Target]
	result := AnimationClip{ID: newID, Name: strings.TrimSpace(name), Duration: clip.Duration, Loop: clip.Loop, Tracks: map[ID]TransformTrack{}}
	ids := make([]string, 0, len(clip.Tracks))
	for id := range clip.Tracks {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, rawID := range ids {
		track := clip.Tracks[ID(rawID)]
		if track.Armature != mapping.Source || track.Bone == "" {
			return AnimationClip{}, fmt.Errorf("track %q is not a bone track for source armature %q", track.ID, mapping.Source)
		}
		targetBoneID, ok := mapping.Bones[track.Bone]
		if !ok {
			return AnimationClip{}, fmt.Errorf("source bone %q is not mapped", track.Bone)
		}
		sourceBone, targetBone := source.Bones[track.Bone], target.Bones[targetBoneID]
		scale := mapping.Scale
		if scale == 0 {
			sourceLength, targetLength := vectorLength(sourceBone.Rest.Position), vectorLength(targetBone.Rest.Position)
			scale = 1
			if sourceLength > 1e-9 && targetLength > 1e-9 {
				scale = targetLength / sourceLength
			}
		}
		newTrack := TransformTrack{ID: ID(fmt.Sprintf("%s--%s", newID, targetBoneID)), Armature: mapping.Target, Bone: targetBoneID}
		for _, key := range track.Keys {
			positionDelta := addVec(key.Transform.Position, scaleVec(sourceBone.Rest.Position, -1))
			rotationDelta := sourceBone.Rest.Rotation.Inverse().Mul(key.Transform.Rotation.Normalized())
			key.Transform.Position = addVec(targetBone.Rest.Position, scaleVec(positionDelta, scale))
			key.Transform.Rotation = targetBone.Rest.Rotation.Normalized().Mul(rotationDelta).Normalized()
			key.Transform.Euler = key.Transform.Rotation.Euler()
			newTrack.Keys = append(newTrack.Keys, key)
		}
		result.Tracks[newTrack.ID] = newTrack
	}
	if err := validateAnimationClip(result, document.Entities, document.Armatures); err != nil {
		return AnimationClip{}, err
	}
	return result, nil
}

func StepAnimationMachine(document Document, machineID ID, deltaTime float64) (MachineStepResult, Document, error) {
	machine, ok := document.AnimationMachines[machineID]
	if !ok {
		return MachineStepResult{}, Document{}, fmt.Errorf("animation machine %q does not exist", machineID)
	}
	if !finite(deltaTime) || deltaTime <= 0 || deltaTime > 3600 {
		return MachineStepResult{}, Document{}, fmt.Errorf("deltaTime must be finite and in (0, 3600]")
	}
	from := machine.Current
	if from == "" {
		from = machine.Initial
		machine.Current = from
	}
	state := machine.States[from]
	machine.StateTime += deltaTime * state.Speed
	transition, trace, err := evaluateMachineTransitions(machine)
	if err != nil {
		return MachineStepResult{}, Document{}, err
	}
	if transition != nil {
		machine.Current = transition.To
		machine.StateTime = 0
		state = machine.States[machine.Current]
	}
	working, err := document.Clone()
	if err != nil {
		return MachineStepResult{}, Document{}, err
	}
	working.AnimationMachines[machineID] = machine
	sample, sampled, err := SampleAnimation(working, state.Clip, machine.StateTime)
	if err != nil {
		return MachineStepResult{}, Document{}, err
	}
	result := MachineStepResult{Machine: machineID, From: from, To: machine.Current, StateTime: machine.StateTime, Clip: state.Clip, Sample: sample, Trace: trace}
	if transition != nil {
		result.Transition = transition.ID
	}
	return result, sampled, nil
}

func evaluateMachineTransitions(machine AnimationStateMachine) (*AnimationTransition, []TransitionTrace, error) {
	transitionProgram.Do(func() { transitionProgram.program, transitionProgram.err = arbiter.Compile(animationTransitionRules) })
	if transitionProgram.err != nil {
		return nil, nil, fmt.Errorf("compile animation transition rules: %w", transitionProgram.err)
	}
	candidates := append([]AnimationTransition(nil), machine.Transitions...)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].ID < candidates[j].ID
	})
	trace := []TransitionTrace{}
	for index := range candidates {
		candidate := &candidates[index]
		if candidate.From != machine.Current {
			continue
		}
		value := machine.Parameters[candidate.Parameter]
		context := map[string]any{"transition": map[string]any{"operator": candidate.Operator, "parameter_value": value, "threshold": candidate.Threshold, "state_time": machine.StateTime, "exit_time": candidate.ExitTime}}
		matches, err := arbiter.Eval(transitionProgram.program, arbiter.DataFromMap(context, transitionProgram.program))
		if err != nil {
			return nil, trace, err
		}
		entry := TransitionTrace{Transition: candidate.ID, Parameter: candidate.Parameter, Value: value, Threshold: candidate.Threshold, StateTime: machine.StateTime, Eligible: len(matches) > 0}
		if len(matches) > 0 {
			entry.Rule = matches[0].Name
			trace = append(trace, entry)
			return candidate, trace, nil
		}
		trace = append(trace, entry)
	}
	return nil, trace, nil
}

func applyRetargetAnimation(document *Document, operation Operation) ([]ID, error) {
	clip, err := RetargetAnimationClip(*document, operation.RetargetMapID, operation.SourceClipID, operation.NewID, operation.Name)
	if err != nil {
		return nil, err
	}
	if document.Animations == nil {
		document.Animations = map[ID]AnimationClip{}
	}
	document.Animations[clip.ID] = clip
	return []ID{operation.RetargetMapID, operation.SourceClipID, clip.ID}, nil
}

func applySetAnimationParameter(document *Document, operation Operation) ([]ID, error) {
	machine, ok := document.AnimationMachines[operation.MachineID]
	if !ok {
		return nil, fmt.Errorf("animation machine %q does not exist", operation.MachineID)
	}
	if _, ok := machine.Parameters[operation.Parameter]; !ok || !finite(operation.Number) {
		return nil, fmt.Errorf("animation parameter %q does not exist or value is non-finite", operation.Parameter)
	}
	machine.Parameters[operation.Parameter] = operation.Number
	document.AnimationMachines[operation.MachineID] = machine
	return []ID{operation.MachineID}, nil
}

func applyStepAnimationMachine(document *Document, operation Operation) ([]ID, error) {
	result, stepped, err := StepAnimationMachine(*document, operation.MachineID, operation.DeltaTime)
	if err != nil {
		return nil, err
	}
	*document = stepped
	affected := []ID{operation.MachineID, result.Clip}
	for id := range result.Sample.Transforms {
		affected = append(affected, id)
	}
	return affected, nil
}

func retargetChanges(transaction Transaction) []RetargetChange {
	out := []RetargetChange{}
	for _, operation := range transaction.Operations {
		if operation.Kind == OpRetargetAnimation {
			out = append(out, RetargetChange{Map: operation.RetargetMapID, Source: operation.SourceClipID, Created: operation.NewID})
		}
	}
	return out
}

func machineChanges(transaction Transaction, before, after Document) []MachineChange {
	out := []MachineChange{}
	for _, operation := range transaction.Operations {
		if operation.Kind != OpSetAnimationParameter && operation.Kind != OpStepAnimationMachine {
			continue
		}
		left, leftOK := before.AnimationMachines[operation.MachineID]
		right, rightOK := after.AnimationMachines[operation.MachineID]
		if !leftOK || !rightOK {
			continue
		}
		out = append(out, MachineChange{Machine: operation.MachineID, BeforeState: left.Current, AfterState: right.Current, BeforeTime: left.StateTime, AfterTime: right.StateTime})
	}
	return out
}
