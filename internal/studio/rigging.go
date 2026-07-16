package studio

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Armature is authoring truth for a stable bone graph. Bone Entity bindings
// let rigid articulated assets and skinned meshes share one evaluator.
type Armature struct {
	ID          ID               `json:"id"`
	Name        string           `json:"name"`
	RootBones   []ID             `json:"rootBones"`
	Bones       map[ID]Bone      `json:"bones"`
	Pose        map[ID]Transform `json:"pose,omitempty"`
	Constraints []IKConstraint   `json:"constraints,omitempty"`
}

type Bone struct {
	ID       ID        `json:"id"`
	Name     string    `json:"name"`
	Parent   ID        `json:"parent,omitempty"`
	Children []ID      `json:"children,omitempty"`
	Rest     Transform `json:"rest"`
	Entity   ID        `json:"entity,omitempty"`
}

type IKConstraint struct {
	ID      ID   `json:"id"`
	Root    ID   `json:"root"`
	Mid     ID   `json:"mid"`
	Tip     ID   `json:"tip"`
	Target  Vec3 `json:"target"`
	Enabled bool `json:"enabled"`
}

type SkinComponent struct {
	Armature ID                       `json:"armature"`
	Weights  map[ID][]VertexInfluence `json:"weights"`
}

type VertexInfluence struct {
	Bone   ID      `json:"bone"`
	Weight float64 `json:"weight"`
}

type AnimationClip struct {
	ID       ID                    `json:"id"`
	Name     string                `json:"name"`
	Duration float64               `json:"duration"`
	Loop     bool                  `json:"loop,omitempty"`
	Tracks   map[ID]TransformTrack `json:"tracks"`
}

type TransformTrack struct {
	ID       ID             `json:"id"`
	Entity   ID             `json:"entity,omitempty"`
	Armature ID             `json:"armature,omitempty"`
	Bone     ID             `json:"bone,omitempty"`
	Keys     []TransformKey `json:"keys"`
}

type TransformKey struct {
	Time      float64   `json:"time"`
	Transform Transform `json:"transform"`
}

type AnimationSample struct {
	Clip       ID                      `json:"clip"`
	Time       float64                 `json:"time"`
	Transforms map[ID]Transform        `json:"transforms"`
	BonePoses  map[ID]map[ID]Transform `json:"bonePoses,omitempty"`
}

type IKResult struct {
	Armature   ID               `json:"armature"`
	Constraint ID               `json:"constraint"`
	Target     Vec3             `json:"target"`
	Reached    Vec3             `json:"reached"`
	Error      float64          `json:"error"`
	Pose       map[ID]Transform `json:"pose"`
}

// ArticulatedProofDocument is the M2 follow-on proof required by the product
// spec: a stable three-bone machine, normalized vertex weights, a two-bone IK
// contract, and an authored transform clip. Rendering remains ordinary typed
// Scene3D entity lowering while deforming skin support is still partial.
func ArticulatedProofDocument() Document {
	document := SampleDocument()
	document.ID = "articulated-machine-proof"
	document.Name = "Articulated Machine Proof"
	document.Metadata["proof"] = "m2-articulated-machine-foundation"
	root := document.Entities["scene-root"]
	upper := meshEntity("upper-arm", "Upper arm", Vec3{}, Geometry{Kind: "box", Width: 0.2, Height: 1, Depth: 0.2}, "board-material", true)
	upper.Parent = root.ID
	forearm := meshEntity("forearm", "Forearm", Vec3{Y: 1}, Geometry{Kind: "box", Width: 0.2, Height: 1, Depth: 0.2}, "board-material", true)
	forearm.Parent = root.ID
	geometry := Geometry{Kind: "indexed-mesh", Vertices: []Vertex{{ID: "v0", Position: Vec3{}}, {ID: "v1", Position: Vec3{X: 1}}, {ID: "v2", Position: Vec3{Y: 1}}}, Faces: []Face{{ID: "f", Vertices: []ID{"v0", "v1", "v2"}}}}
	skinned := meshEntity("skinned", "Skinned proof", Vec3{}, geometry, "board-material", true)
	skinned.Parent = root.ID
	skinned.Skin = &SkinComponent{Armature: "arm", Weights: map[ID][]VertexInfluence{
		"v0": {{Bone: "root", Weight: 1}}, "v1": {{Bone: "lower", Weight: 1}}, "v2": {{Bone: "tip", Weight: 1}},
	}}
	ground := meshEntity("physics-ground", "Physics ground", Vec3{}, Geometry{Kind: "plane", Width: 4, Height: 4}, "pedestal-material", true)
	ground.Parent = root.ID
	ground.Physics = &PhysicsBody{Kind: "static", Collider: Collider{Kind: "plane", Normal: Vec3{Y: 1}}}
	payload := meshEntity("physics-payload", "Physics payload", Vec3{X: 0.5, Y: 3}, Geometry{Kind: "sphere", Radius: 0.2, Segments: 16}, "player-1-material", true)
	payload.Parent = root.ID
	payload.Physics = &PhysicsBody{Kind: "dynamic", Mass: 1, GravityScale: 1, Restitution: 0.35, Collider: Collider{Kind: "sphere", Radius: 0.2}}
	root.Children = append(root.Children, upper.ID, forearm.ID, skinned.ID, ground.ID, payload.ID)
	document.Entities[root.ID] = root
	document.Entities[upper.ID] = upper
	document.Entities[forearm.ID] = forearm
	document.Entities[skinned.ID] = skinned
	document.Entities[ground.ID] = ground
	document.Entities[payload.ID] = payload
	document.Armatures = map[ID]Armature{"arm": {ID: "arm", Name: "Robot arm", RootBones: []ID{"root"}, Bones: map[ID]Bone{
		"root":  {ID: "root", Name: "Shoulder", Children: []ID{"lower"}, Rest: IdentityTransform(), Entity: upper.ID},
		"lower": {ID: "lower", Name: "Elbow", Parent: "root", Children: []ID{"tip"}, Rest: TransformFromEuler(Vec3{Y: 1}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Entity: forearm.ID},
		"tip":   {ID: "tip", Name: "Tool", Parent: "lower", Rest: TransformFromEuler(Vec3{Y: 1}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})},
	}, Pose: map[ID]Transform{"lower": forearm.Transform}, Constraints: []IKConstraint{{ID: "reach-ik", Root: "root", Mid: "lower", Tip: "tip", Target: Vec3{X: 1, Y: 1}, Enabled: true}}}}
	document.Armatures["target-arm"] = Armature{ID: "target-arm", Name: "Tall robot arm", RootBones: []ID{"target-root"}, Bones: map[ID]Bone{
		"target-root":  {ID: "target-root", Name: "Target shoulder", Children: []ID{"target-lower"}, Rest: IdentityTransform()},
		"target-lower": {ID: "target-lower", Name: "Target elbow", Parent: "target-root", Children: []ID{"target-tip"}, Rest: TransformFromEuler(Vec3{Y: 2}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})},
		"target-tip":   {ID: "target-tip", Name: "Target tool", Parent: "target-lower", Rest: TransformFromEuler(Vec3{Y: 2}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})},
	}}
	start := forearm.Transform
	end := forearm.Transform
	end.Euler = Vec3{X: 2, Y: 2}
	end.Rotation = QuaternionFromEuler(end.Euler)
	document.Animations = map[ID]AnimationClip{"reach": {ID: "reach", Name: "Reach", Duration: 1, Tracks: map[ID]TransformTrack{
		"lower-track": {ID: "lower-track", Armature: "arm", Bone: "lower", Keys: []TransformKey{{Time: 0, Transform: start}, {Time: 1, Transform: end}}},
	}}}
	document.Animations["idle"] = AnimationClip{ID: "idle", Name: "Idle", Duration: 1, Loop: true, Tracks: map[ID]TransformTrack{"idle-lower": {ID: "idle-lower", Armature: "arm", Bone: "lower", Keys: []TransformKey{{Time: 0, Transform: start}, {Time: 1, Transform: start}}}}}
	document.RetargetMaps = map[ID]RetargetMap{"arm-to-tall": {ID: "arm-to-tall", Name: "Arm to tall arm", Source: "arm", Target: "target-arm", Bones: map[ID]ID{"root": "target-root", "lower": "target-lower", "tip": "target-tip"}}}
	document.AnimationMachines = map[ID]AnimationStateMachine{"locomotion": {ID: "locomotion", Name: "Articulated locomotion", Initial: "idle", Current: "idle", Parameters: map[string]float64{"speed": 0}, States: map[ID]AnimationState{"idle": {ID: "idle", Name: "Idle", Clip: "idle", Speed: 1}, "reach": {ID: "reach", Name: "Reach", Clip: "reach", Speed: 1}}, Transitions: []AnimationTransition{{ID: "idle-to-reach", From: "idle", To: "reach", Parameter: "speed", Operator: "greater", Threshold: 0.5, Priority: 10}, {ID: "reach-to-idle", From: "reach", To: "idle", Parameter: "speed", Operator: "less-equal", Threshold: 0.5, Priority: 10}}}}
	document.Simulations = map[ID]SimulationProfile{"articulated-physics": {ID: "articulated-physics", Name: "Articulated payload interaction", TickRate: 60, Gravity: Vec3{Y: -9.81}, Bodies: []ID{ground.ID, payload.ID}}}
	return document
}

func validateArmature(armature Armature, entities map[ID]Entity) error {
	if armature.ID == "" || strings.TrimSpace(armature.Name) == "" || len(armature.Bones) == 0 || len(armature.RootBones) == 0 {
		return fmt.Errorf("id, name, bones, and rootBones are required")
	}
	roots := map[ID]bool{}
	for _, id := range armature.RootBones {
		bone, ok := armature.Bones[id]
		if !ok || bone.Parent != "" || roots[id] {
			return fmt.Errorf("invalid or duplicate root bone %q", id)
		}
		roots[id] = true
	}
	for id, bone := range armature.Bones {
		if id == "" || bone.ID != id || strings.TrimSpace(bone.Name) == "" || !finiteTransform(bone.Rest) {
			return fmt.Errorf("invalid bone %q", id)
		}
		if bone.Parent == "" && !roots[id] {
			return fmt.Errorf("unlisted root bone %q", id)
		}
		if bone.Parent != "" {
			parent, ok := armature.Bones[bone.Parent]
			if !ok || !containsID(parent.Children, id) {
				return fmt.Errorf("bone %q has inconsistent parent %q", id, bone.Parent)
			}
		}
		for _, child := range bone.Children {
			value, ok := armature.Bones[child]
			if !ok || value.Parent != id {
				return fmt.Errorf("bone %q has inconsistent child %q", id, child)
			}
		}
		if bone.Entity != "" {
			if _, ok := entities[bone.Entity]; !ok {
				return fmt.Errorf("bone %q references missing entity %q", id, bone.Entity)
			}
		}
	}
	for id, pose := range armature.Pose {
		if _, ok := armature.Bones[id]; !ok || !finiteTransform(pose) {
			return fmt.Errorf("invalid pose for bone %q", id)
		}
	}
	visiting, visited := map[ID]bool{}, map[ID]bool{}
	var walk func(ID) error
	walk = func(id ID) error {
		if visiting[id] {
			return fmt.Errorf("bone cycle reaches %q", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, child := range armature.Bones[id].Children {
			if err := walk(child); err != nil {
				return err
			}
		}
		delete(visiting, id)
		visited[id] = true
		return nil
	}
	for _, root := range armature.RootBones {
		if err := walk(root); err != nil {
			return err
		}
	}
	if len(visited) != len(armature.Bones) {
		return fmt.Errorf("bone graph contains unreachable records")
	}
	constraintIDs := map[ID]bool{}
	for _, constraint := range armature.Constraints {
		if constraint.ID == "" || constraintIDs[constraint.ID] {
			return fmt.Errorf("constraint IDs must be non-empty and unique")
		}
		constraintIDs[constraint.ID] = true
		if !constraint.Enabled {
			continue
		}
		root, rootOK := armature.Bones[constraint.Root]
		mid, midOK := armature.Bones[constraint.Mid]
		tip, tipOK := armature.Bones[constraint.Tip]
		if !rootOK || !midOK || !tipOK || mid.Parent != root.ID || tip.Parent != mid.ID || !finiteVec(constraint.Target) {
			return fmt.Errorf("constraint %q is not a valid two-bone chain", constraint.ID)
		}
	}
	return nil
}

func validateSkin(skin SkinComponent, geometry Geometry, armature Armature) error {
	if geometry.Kind != "indexed-mesh" {
		return fmt.Errorf("skin requires indexed-mesh geometry")
	}
	for _, vertex := range geometry.Vertices {
		influences := skin.Weights[vertex.ID]
		if len(influences) == 0 || len(influences) > 4 {
			return fmt.Errorf("vertex %q requires one to four influences", vertex.ID)
		}
		total := 0.0
		seen := map[ID]bool{}
		for _, influence := range influences {
			if _, ok := armature.Bones[influence.Bone]; !ok || seen[influence.Bone] || !finite(influence.Weight) || influence.Weight <= 0 {
				return fmt.Errorf("vertex %q has invalid influence %q", vertex.ID, influence.Bone)
			}
			seen[influence.Bone] = true
			total += influence.Weight
		}
		if math.Abs(total-1) > 1e-6 {
			return fmt.Errorf("vertex %q influence sum %.9f, want 1", vertex.ID, total)
		}
	}
	return nil
}

func validateAnimationClip(clip AnimationClip, entities map[ID]Entity, armatures map[ID]Armature) error {
	if clip.ID == "" || strings.TrimSpace(clip.Name) == "" || !finite(clip.Duration) || clip.Duration <= 0 || len(clip.Tracks) == 0 {
		return fmt.Errorf("id, name, positive duration, and tracks are required")
	}
	for id, track := range clip.Tracks {
		if id == "" || track.ID != id || len(track.Keys) == 0 || (track.Entity == "") == (track.Bone == "") {
			return fmt.Errorf("track %q must target exactly one entity or bone", id)
		}
		if track.Entity != "" {
			if _, ok := entities[track.Entity]; !ok {
				return fmt.Errorf("track %q references missing entity %q", id, track.Entity)
			}
		} else {
			armature, ok := armatures[track.Armature]
			if !ok {
				return fmt.Errorf("track %q references missing armature %q", id, track.Armature)
			}
			if _, ok := armature.Bones[track.Bone]; !ok {
				return fmt.Errorf("track %q references missing bone %q", id, track.Bone)
			}
		}
		previous := -1.0
		for _, key := range track.Keys {
			if !finite(key.Time) || key.Time < 0 || key.Time > clip.Duration || key.Time <= previous || !finiteTransform(key.Transform) {
				return fmt.Errorf("track %q has invalid or unordered key at %.6f", id, key.Time)
			}
			previous = key.Time
		}
	}
	return nil
}

// SampleAnimation deterministically evaluates entity and bone tracks. The
// returned document uses ordinary entity transforms and therefore compiles
// through the same Scene3D/SceneIR path as authored state.
func SampleAnimation(document Document, clipID ID, time float64) (AnimationSample, Document, error) {
	clip, ok := document.Animations[clipID]
	if !ok {
		return AnimationSample{}, Document{}, fmt.Errorf("animation %q does not exist", clipID)
	}
	if !finite(time) || time < 0 {
		return AnimationSample{}, Document{}, fmt.Errorf("sample time must be finite and non-negative")
	}
	if clip.Loop {
		time = math.Mod(time, clip.Duration)
	} else if time > clip.Duration {
		time = clip.Duration
	}
	result, err := document.Clone()
	if err != nil {
		return AnimationSample{}, Document{}, err
	}
	sample := AnimationSample{Clip: clipID, Time: time, Transforms: map[ID]Transform{}, BonePoses: map[ID]map[ID]Transform{}}
	ids := make([]string, 0, len(clip.Tracks))
	for id := range clip.Tracks {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, rawID := range ids {
		track := clip.Tracks[ID(rawID)]
		value := sampleTransformTrack(track.Keys, time)
		if track.Entity != "" {
			entity := result.Entities[track.Entity]
			entity.Transform = value
			result.Entities[track.Entity] = entity
			sample.Transforms[track.Entity] = value
			continue
		}
		if sample.BonePoses[track.Armature] == nil {
			sample.BonePoses[track.Armature] = map[ID]Transform{}
		}
		sample.BonePoses[track.Armature][track.Bone] = value
		armature := result.Armatures[track.Armature]
		if armature.Pose == nil {
			armature.Pose = map[ID]Transform{}
		}
		armature.Pose[track.Bone] = value
		result.Armatures[track.Armature] = armature
		bone := armature.Bones[track.Bone]
		if bone.Entity != "" {
			entity := result.Entities[bone.Entity]
			entity.Transform = value
			result.Entities[bone.Entity] = entity
			sample.Transforms[bone.Entity] = value
		}
	}
	return sample, result, nil
}

// SolveTwoBoneIK solves a planar XY chain using the law of cosines. It is a
// deterministic CPU reference used by authoring, agents, replay, and headless
// certification; runtime visualization consumes the resulting entity poses.
func SolveTwoBoneIK(document Document, armatureID, constraintID ID) (IKResult, Document, error) {
	armature, ok := document.Armatures[armatureID]
	if !ok {
		return IKResult{}, Document{}, fmt.Errorf("armature %q does not exist", armatureID)
	}
	var constraint IKConstraint
	found := false
	for _, candidate := range armature.Constraints {
		if candidate.ID == constraintID {
			constraint, found = candidate, true
			break
		}
	}
	if !found || !constraint.Enabled {
		return IKResult{}, Document{}, fmt.Errorf("enabled IK constraint %q does not exist", constraintID)
	}
	root, mid, tip := armature.Bones[constraint.Root], armature.Bones[constraint.Mid], armature.Bones[constraint.Tip]
	l1, l2 := vectorLength(mid.Rest.Position), vectorLength(tip.Rest.Position)
	if l1 <= 1e-9 || l2 <= 1e-9 {
		return IKResult{}, Document{}, fmt.Errorf("IK chain %q has zero-length segment", constraintID)
	}
	dx, dy := constraint.Target.X-root.Rest.Position.X, constraint.Target.Y-root.Rest.Position.Y
	distance := math.Hypot(dx, dy)
	clamped := math.Max(math.Abs(l1-l2)+1e-9, math.Min(distance, l1+l2-1e-9))
	cosElbow := (clamped*clamped - l1*l1 - l2*l2) / (2 * l1 * l2)
	cosElbow = math.Max(-1, math.Min(1, cosElbow))
	elbow := math.Acos(cosElbow)
	shoulder := math.Atan2(dy, dx) - math.Atan2(l2*math.Sin(elbow), l1+l2*cosElbow)
	rootPose := poseOrRest(armature, root.ID)
	midPose := poseOrRest(armature, mid.ID)
	rootEuler, midEuler := rootPose.canonical().Euler, midPose.canonical().Euler
	rootEuler.Z, midEuler.Z = shoulder, elbow
	rootPose.Euler, midPose.Euler = rootEuler, midEuler
	rootPose.Rotation, midPose.Rotation = QuaternionFromEuler(rootEuler), QuaternionFromEuler(midEuler)
	if armature.Pose == nil {
		armature.Pose = map[ID]Transform{}
	}
	armature.Pose[root.ID], armature.Pose[mid.ID] = rootPose, midPose
	resultDocument, err := document.Clone()
	if err != nil {
		return IKResult{}, Document{}, err
	}
	resultDocument.Armatures[armatureID] = armature
	for boneID, pose := range map[ID]Transform{root.ID: rootPose, mid.ID: midPose} {
		if entityID := armature.Bones[boneID].Entity; entityID != "" {
			entity := resultDocument.Entities[entityID]
			entity.Transform = pose
			resultDocument.Entities[entityID] = entity
		}
	}
	reached := Vec3{X: root.Rest.Position.X + l1*math.Cos(shoulder) + l2*math.Cos(shoulder+elbow), Y: root.Rest.Position.Y + l1*math.Sin(shoulder) + l2*math.Sin(shoulder+elbow), Z: constraint.Target.Z}
	errorDistance := math.Hypot(reached.X-constraint.Target.X, reached.Y-constraint.Target.Y)
	return IKResult{Armature: armatureID, Constraint: constraintID, Target: constraint.Target, Reached: reached, Error: errorDistance, Pose: map[ID]Transform{root.ID: rootPose, mid.ID: midPose}}, resultDocument, nil
}

func poseOrRest(armature Armature, id ID) Transform {
	if value, ok := armature.Pose[id]; ok {
		return value
	}
	return armature.Bones[id].Rest
}

func sampleTransformTrack(keys []TransformKey, time float64) Transform {
	if time <= keys[0].Time {
		return keys[0].Transform
	}
	for i := 1; i < len(keys); i++ {
		if time <= keys[i].Time {
			span := keys[i].Time - keys[i-1].Time
			alpha := (time - keys[i-1].Time) / span
			return lerpTransform(keys[i-1].Transform, keys[i].Transform, alpha)
		}
	}
	return keys[len(keys)-1].Transform
}

func lerpTransform(a, b Transform, t float64) Transform {
	rotation := Slerp(a.Rotation, b.Rotation, t)
	return Transform{Position: lerpVec(a.Position, b.Position, t), Rotation: rotation, Euler: rotation.Euler(), Scale: lerpVec(a.Scale, b.Scale, t)}
}

func lerpVec(a, b Vec3, t float64) Vec3 {
	return Vec3{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t, Z: a.Z + (b.Z-a.Z)*t}
}

func finiteVec(value Vec3) bool { return finite(value.X) && finite(value.Y) && finite(value.Z) }

func applySetBonePose(document *Document, operation Operation) ([]ID, error) {
	armature, ok := document.Armatures[operation.ArmatureID]
	if !ok {
		return nil, fmt.Errorf("armature %q does not exist", operation.ArmatureID)
	}
	bone, ok := armature.Bones[operation.BoneID]
	if !ok {
		return nil, fmt.Errorf("bone %q does not exist", operation.BoneID)
	}
	if operation.Transform == nil || !finiteTransform(*operation.Transform) {
		return nil, fmt.Errorf("set-bone-pose requires a finite transform")
	}
	if armature.Pose == nil {
		armature.Pose = map[ID]Transform{}
	}
	armature.Pose[operation.BoneID] = *operation.Transform
	document.Armatures[operation.ArmatureID] = armature
	affected := []ID{operation.ArmatureID, operation.BoneID}
	if bone.Entity != "" {
		entity := document.Entities[bone.Entity]
		if entity.Locked {
			return nil, fmt.Errorf("bone entity %q is locked", bone.Entity)
		}
		entity.Transform = *operation.Transform
		document.Entities[bone.Entity] = entity
		affected = append(affected, bone.Entity)
	}
	return affected, nil
}

func applySetAnimationKey(document *Document, operation Operation) ([]ID, error) {
	clip, ok := document.Animations[operation.ClipID]
	if !ok {
		return nil, fmt.Errorf("animation %q does not exist", operation.ClipID)
	}
	track, ok := clip.Tracks[operation.TrackID]
	if !ok {
		return nil, fmt.Errorf("track %q does not exist", operation.TrackID)
	}
	if operation.Key == nil || !finite(operation.Key.Time) || operation.Key.Time < 0 || operation.Key.Time > clip.Duration || !finiteTransform(operation.Key.Transform) {
		return nil, fmt.Errorf("set-animation-key requires a finite key within clip duration")
	}
	replaced := false
	for index := range track.Keys {
		if math.Abs(track.Keys[index].Time-operation.Key.Time) < 1e-9 {
			track.Keys[index] = *operation.Key
			replaced = true
			break
		}
	}
	if !replaced {
		track.Keys = append(track.Keys, *operation.Key)
	}
	sort.Slice(track.Keys, func(i, j int) bool { return track.Keys[i].Time < track.Keys[j].Time })
	clip.Tracks[operation.TrackID] = track
	document.Animations[operation.ClipID] = clip
	return []ID{operation.ClipID, operation.TrackID}, nil
}

func applySolveIK(document *Document, operation Operation) ([]ID, error) {
	result, solved, err := SolveTwoBoneIK(*document, operation.ArmatureID, operation.ConstraintID)
	if err != nil {
		return nil, err
	}
	*document = solved
	affected := []ID{operation.ArmatureID, operation.ConstraintID}
	armature := solved.Armatures[operation.ArmatureID]
	for boneID := range result.Pose {
		affected = append(affected, boneID)
		if entityID := armature.Bones[boneID].Entity; entityID != "" {
			affected = append(affected, entityID)
		}
	}
	return affected, nil
}

func rigChanges(transaction Transaction, before, after Document) []RigChange {
	out := []RigChange{}
	for _, operation := range transaction.Operations {
		if operation.Kind != OpSetBonePose && operation.Kind != OpSolveIK {
			continue
		}
		boneIDs := []ID{operation.BoneID}
		if operation.Kind == OpSolveIK {
			for _, constraint := range after.Armatures[operation.ArmatureID].Constraints {
				if constraint.ID == operation.ConstraintID {
					boneIDs = []ID{constraint.Root, constraint.Mid}
					break
				}
			}
		}
		for _, boneID := range boneIDs {
			change := RigChange{Armature: operation.ArmatureID, Bone: boneID}
			if value, ok := before.Armatures[operation.ArmatureID].Pose[boneID]; ok {
				copy := value
				change.Before = &copy
			}
			if value, ok := after.Armatures[operation.ArmatureID].Pose[boneID]; ok {
				copy := value
				change.After = &copy
			}
			out = append(out, change)
		}
	}
	return out
}

func animationChanges(transaction Transaction, before, after Document) []AnimationChange {
	out := []AnimationChange{}
	for _, operation := range transaction.Operations {
		if operation.Kind != OpSetAnimationKey || operation.Key == nil {
			continue
		}
		change := AnimationChange{Clip: operation.ClipID, Track: operation.TrackID}
		for _, key := range before.Animations[operation.ClipID].Tracks[operation.TrackID].Keys {
			if math.Abs(key.Time-operation.Key.Time) < 1e-9 {
				copy := key
				change.Before = &copy
				break
			}
		}
		for _, key := range after.Animations[operation.ClipID].Tracks[operation.TrackID].Keys {
			if math.Abs(key.Time-operation.Key.Time) < 1e-9 {
				copy := key
				change.After = &copy
				break
			}
		}
		out = append(out, change)
	}
	return out
}
