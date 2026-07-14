package studio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

const SceneDocSchema = "gosx.scene3d.document/v1"

type ID string

type Document struct {
	Schema            string                       `json:"schema"`
	ID                ID                           `json:"id"`
	Name              string                       `json:"name"`
	Revision          uint64                       `json:"revision"`
	RootIDs           []ID                         `json:"rootIds"`
	Entities          map[ID]Entity                `json:"entities"`
	Materials         map[ID]Material              `json:"materials"`
	Prefabs           map[ID]PrefabDefinition      `json:"prefabs,omitempty"`
	Assets            map[ID]AssetRecord           `json:"assets,omitempty"`
	Armatures         map[ID]Armature              `json:"armatures,omitempty"`
	Animations        map[ID]AnimationClip         `json:"animations,omitempty"`
	Simulations       map[ID]SimulationProfile     `json:"simulations,omitempty"`
	RetargetMaps      map[ID]RetargetMap           `json:"retargetMaps,omitempty"`
	AnimationMachines map[ID]AnimationStateMachine `json:"animationMachines,omitempty"`
	RenderGraph       *RenderGraph                 `json:"renderGraph,omitempty"`
	Camera            Camera                       `json:"camera"`
	Environment       Environment                  `json:"environment"`
	Metadata          map[string]string            `json:"metadata,omitempty"`
}

type Entity struct {
	ID        ID              `json:"id"`
	Name      string          `json:"name"`
	Parent    ID              `json:"parent,omitempty"`
	Children  []ID            `json:"children,omitempty"`
	Transform Transform       `json:"transform"`
	Mesh      *MeshComponent  `json:"mesh,omitempty"`
	Light     *LightComponent `json:"light,omitempty"`
	Prefab    *PrefabInstance `json:"prefab,omitempty"`
	Model     *ModelComponent `json:"model,omitempty"`
	Skin      *SkinComponent  `json:"skin,omitempty"`
	Physics   *PhysicsBody    `json:"physics,omitempty"`
	Visible   bool            `json:"visible"`
	Locked    bool            `json:"locked,omitempty"`
}

type AssetRecord struct {
	ID          ID                `json:"id"`
	Kind        string            `json:"kind"`
	Format      string            `json:"format"`
	MediaType   string            `json:"mediaType"`
	ContentHash string            `json:"contentHash"`
	Bytes       int64             `json:"bytes"`
	URI         string            `json:"uri"`
	StorePath   string            `json:"storePath"`
	SourceName  string            `json:"sourceName"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ModelComponent struct {
	Asset         ID      `json:"asset"`
	Bounds        float64 `json:"bounds,omitempty"`
	Fit           string  `json:"fit,omitempty"`
	FitAlign      string  `json:"fitAlign,omitempty"`
	Pickable      bool    `json:"pickable"`
	CastShadow    bool    `json:"castShadow,omitempty"`
	ReceiveShadow bool    `json:"receiveShadow,omitempty"`
}

type PrefabDefinition struct {
	ID       ID            `json:"id"`
	Name     string        `json:"name"`
	Root     ID            `json:"root"`
	Entities map[ID]Entity `json:"entities"`
}

type PrefabInstance struct {
	Prefab    ID                          `json:"prefab"`
	Overrides map[ID]PrefabEntityOverride `json:"overrides,omitempty"`
}

type PrefabEntityOverride struct {
	Transform *Transform `json:"transform,omitempty"`
	Material  ID         `json:"material,omitempty"`
	Visible   *bool      `json:"visible,omitempty"`
}

type Transform struct {
	Position Vec3 `json:"position"`
	Rotation Vec3 `json:"rotation"`
	Scale    Vec3 `json:"scale"`
}

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type MeshComponent struct {
	Geometry      Geometry   `json:"geometry"`
	Material      ID         `json:"material"`
	Pickable      bool       `json:"pickable"`
	CastShadow    bool       `json:"castShadow,omitempty"`
	ReceiveShadow bool       `json:"receiveShadow,omitempty"`
	Modifiers     []Modifier `json:"modifiers,omitempty"`
}

type Modifier struct {
	ID        ID      `json:"id"`
	Kind      string  `json:"kind"`
	Enabled   bool    `json:"enabled"`
	Axis      string  `json:"axis,omitempty"`
	Count     int     `json:"count,omitempty"`
	Offset    Vec3    `json:"offset,omitempty"`
	Thickness float64 `json:"thickness,omitempty"`
	Levels    int     `json:"levels,omitempty"`
}

type Geometry struct {
	Kind           string         `json:"kind"`
	Width          float64        `json:"width,omitempty"`
	Height         float64        `json:"height,omitempty"`
	Depth          float64        `json:"depth,omitempty"`
	Radius         float64        `json:"radius,omitempty"`
	Segments       int            `json:"segments,omitempty"`
	RadiusTop      float64        `json:"radiusTop,omitempty"`
	RadiusBottom   float64        `json:"radiusBottom,omitempty"`
	RadialSegments int            `json:"radialSegments,omitempty"`
	Vertices       []Vertex       `json:"vertices,omitempty"`
	Faces          []Face         `json:"faces,omitempty"`
	Curve          *CurveGeometry `json:"curve,omitempty"`
}

type CurveGeometry struct {
	Degree         int                 `json:"degree"`
	ControlPoints  []CurveControlPoint `json:"controlPoints"`
	Knots          []float64           `json:"knots"`
	Segments       int                 `json:"segments"`
	Radius         float64             `json:"radius"`
	RadialSegments int                 `json:"radialSegments"`
}

type CurveControlPoint struct {
	ID       ID      `json:"id"`
	Position Vec3    `json:"position"`
	Weight   float64 `json:"weight"`
}

type Vertex struct {
	ID       ID    `json:"id"`
	Position Vec3  `json:"position"`
	Normal   Vec3  `json:"normal,omitempty"`
	UV       *Vec2 `json:"uv,omitempty"`
}

type Face struct {
	ID       ID   `json:"id"`
	Vertices []ID `json:"vertices"`
}

type LightComponent struct {
	Kind       string  `json:"kind"`
	Color      string  `json:"color"`
	Intensity  float64 `json:"intensity"`
	Direction  Vec3    `json:"direction,omitempty"`
	Position   Vec3    `json:"position,omitempty"`
	Range      float64 `json:"range,omitempty"`
	CastShadow bool    `json:"castShadow,omitempty"`
}

type Material struct {
	ID           ID            `json:"id"`
	Name         string        `json:"name"`
	Color        string        `json:"color"`
	Roughness    float64       `json:"roughness"`
	Metalness    float64       `json:"metalness"`
	Clearcoat    float64       `json:"clearcoat,omitempty"`
	Transmission float64       `json:"transmission,omitempty"`
	Emissive     float64       `json:"emissive,omitempty"`
	Selena       *SelenaShader `json:"selena,omitempty"`
}

type SelenaShader struct {
	Material string `json:"material"`
	Source   string `json:"source"`
}

type Camera struct {
	Position Vec3    `json:"position"`
	Rotation Vec3    `json:"rotation"`
	FOV      float64 `json:"fov"`
	Near     float64 `json:"near"`
	Far      float64 `json:"far"`
}

type Environment struct {
	Background       string  `json:"background"`
	AmbientColor     string  `json:"ambientColor"`
	AmbientIntensity float64 `json:"ambientIntensity"`
	Exposure         float64 `json:"exposure"`
	ToneMapping      string  `json:"toneMapping"`
}

func IdentityTransform() Transform {
	return Transform{Scale: Vec3{X: 1, Y: 1, Z: 1}}
}

func (d Document) Clone() (Document, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return Document{}, fmt.Errorf("clone SceneDoc: %w", err)
	}
	var clone Document
	if err := json.Unmarshal(data, &clone); err != nil {
		return Document{}, fmt.Errorf("clone SceneDoc: %w", err)
	}
	return clone, nil
}

func (d Document) Fingerprint() (string, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("fingerprint SceneDoc: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (d Document) Validate() error {
	if d.Schema != SceneDocSchema {
		return fmt.Errorf("SceneDoc schema %q, want %q", d.Schema, SceneDocSchema)
	}
	if strings.TrimSpace(string(d.ID)) == "" {
		return fmt.Errorf("SceneDoc id is required")
	}
	if len(d.Entities) == 0 {
		return fmt.Errorf("SceneDoc requires at least one entity")
	}
	if len(d.RootIDs) == 0 {
		return fmt.Errorf("SceneDoc requires at least one root entity")
	}
	rootSet := make(map[ID]bool, len(d.RootIDs))
	for _, id := range d.RootIDs {
		if rootSet[id] {
			return fmt.Errorf("duplicate root entity %q", id)
		}
		rootSet[id] = true
		entity, ok := d.Entities[id]
		if !ok {
			return fmt.Errorf("root entity %q does not exist", id)
		}
		if entity.Parent != "" {
			return fmt.Errorf("root entity %q has parent %q", id, entity.Parent)
		}
	}
	for key, material := range d.Materials {
		if key == "" || material.ID != key {
			return fmt.Errorf("material map key %q does not match id %q", key, material.ID)
		}
		if err := validateMaterialRecord(material); err != nil {
			return err
		}
	}
	for key, prefab := range d.Prefabs {
		if key == "" || prefab.ID != key {
			return fmt.Errorf("prefab map key %q does not match id %q", key, prefab.ID)
		}
		if err := validatePrefabDefinition(prefab, d.Materials, d.Assets); err != nil {
			return fmt.Errorf("prefab %q: %w", key, err)
		}
	}
	for key, asset := range d.Assets {
		if key == "" || asset.ID != key {
			return fmt.Errorf("asset map key %q does not match id %q", key, asset.ID)
		}
		if err := validateAssetRecord(asset); err != nil {
			return fmt.Errorf("asset %q: %w", key, err)
		}
	}
	for key, armature := range d.Armatures {
		if key == "" || armature.ID != key {
			return fmt.Errorf("armature map key %q does not match id %q", key, armature.ID)
		}
		if err := validateArmature(armature, d.Entities); err != nil {
			return fmt.Errorf("armature %q: %w", key, err)
		}
	}
	for key, clip := range d.Animations {
		if key == "" || clip.ID != key {
			return fmt.Errorf("animation map key %q does not match id %q", key, clip.ID)
		}
		if err := validateAnimationClip(clip, d.Entities, d.Armatures); err != nil {
			return fmt.Errorf("animation %q: %w", key, err)
		}
	}
	for key, profile := range d.Simulations {
		if key == "" || profile.ID != key {
			return fmt.Errorf("simulation map key %q does not match id %q", key, profile.ID)
		}
		if err := validateSimulationProfile(profile, d.Entities); err != nil {
			return fmt.Errorf("simulation %q: %w", key, err)
		}
	}
	for key, mapping := range d.RetargetMaps {
		if key == "" || mapping.ID != key {
			return fmt.Errorf("retarget map key %q does not match id %q", key, mapping.ID)
		}
		if err := validateRetargetMap(mapping, d.Armatures); err != nil {
			return fmt.Errorf("retarget map %q: %w", key, err)
		}
	}
	for key, machine := range d.AnimationMachines {
		if key == "" || machine.ID != key {
			return fmt.Errorf("animation machine key %q does not match id %q", key, machine.ID)
		}
		if err := validateAnimationMachine(machine, d.Animations); err != nil {
			return fmt.Errorf("animation machine %q: %w", key, err)
		}
	}
	for key, entity := range d.Entities {
		if key == "" || entity.ID != key {
			return fmt.Errorf("entity map key %q does not match id %q", key, entity.ID)
		}
		if entity.Parent == "" && !rootSet[key] {
			return fmt.Errorf("unlisted root entity %q", key)
		}
		if entity.Parent != "" {
			parent, ok := d.Entities[entity.Parent]
			if !ok {
				return fmt.Errorf("entity %q references missing parent %q", key, entity.Parent)
			}
			if !containsID(parent.Children, key) {
				return fmt.Errorf("parent %q does not list child %q", entity.Parent, key)
			}
		}
		seenChildren := map[ID]bool{}
		for _, childID := range entity.Children {
			if seenChildren[childID] {
				return fmt.Errorf("entity %q lists child %q twice", key, childID)
			}
			seenChildren[childID] = true
			child, ok := d.Entities[childID]
			if !ok {
				return fmt.Errorf("entity %q references missing child %q", key, childID)
			}
			if child.Parent != key {
				return fmt.Errorf("child %q parent is %q, want %q", childID, child.Parent, key)
			}
		}
		if entity.Mesh != nil {
			if _, ok := d.Materials[entity.Mesh.Material]; !ok {
				return fmt.Errorf("entity %q references missing material %q", key, entity.Mesh.Material)
			}
			if !supportedGeometry(entity.Mesh.Geometry.Kind) {
				return fmt.Errorf("entity %q uses unsupported geometry %q", key, entity.Mesh.Geometry.Kind)
			}
			if err := validateGeometry(entity.Mesh.Geometry); err != nil {
				return fmt.Errorf("entity %q geometry: %w", key, err)
			}
			if err := validateModifiers(entity.Mesh.Geometry, entity.Mesh.Modifiers); err != nil {
				return fmt.Errorf("entity %q modifiers: %w", key, err)
			}
		}
		if entity.Prefab != nil {
			if entity.Mesh != nil || entity.Light != nil || entity.Model != nil {
				return fmt.Errorf("prefab instance %q cannot also carry mesh, model, or light", key)
			}
			definition, ok := d.Prefabs[entity.Prefab.Prefab]
			if !ok {
				return fmt.Errorf("entity %q references missing prefab %q", key, entity.Prefab.Prefab)
			}
			for localID, override := range entity.Prefab.Overrides {
				if _, ok := definition.Entities[localID]; !ok {
					return fmt.Errorf("entity %q overrides missing prefab entity %q", key, localID)
				}
				if override.Material != "" {
					if _, ok := d.Materials[override.Material]; !ok {
						return fmt.Errorf("entity %q prefab override references missing material %q", key, override.Material)
					}
				}
				if override.Transform != nil && !finiteTransform(*override.Transform) {
					return fmt.Errorf("entity %q prefab override %q has non-finite transform", key, localID)
				}
			}
		}
		if entity.Model != nil {
			if entity.Mesh != nil || entity.Light != nil {
				return fmt.Errorf("model entity %q cannot also carry mesh or light", key)
			}
			asset, ok := d.Assets[entity.Model.Asset]
			if !ok {
				return fmt.Errorf("entity %q references missing asset %q", key, entity.Model.Asset)
			}
			if asset.Format != "glb" && asset.Format != "gltf" {
				return fmt.Errorf("entity %q model asset %q is not glb/gltf", key, asset.ID)
			}
			if !finite(entity.Model.Bounds) || entity.Model.Bounds < 0 {
				return fmt.Errorf("entity %q has invalid model bounds", key)
			}
		}
		if entity.Skin != nil {
			if entity.Mesh == nil {
				return fmt.Errorf("skinned entity %q requires indexed mesh geometry", key)
			}
			if len(entity.Mesh.Modifiers) != 0 {
				return fmt.Errorf("entity %q skin/modifier ordering is not implemented", key)
			}
			armature, ok := d.Armatures[entity.Skin.Armature]
			if !ok {
				return fmt.Errorf("entity %q references missing armature %q", key, entity.Skin.Armature)
			}
			if err := validateSkin(*entity.Skin, entity.Mesh.Geometry, armature); err != nil {
				return fmt.Errorf("entity %q skin: %w", key, err)
			}
		}
		if entity.Physics != nil {
			if err := validatePhysicsBody(*entity.Physics); err != nil {
				return fmt.Errorf("entity %q physics: %w", key, err)
			}
		}
	}
	visiting := map[ID]bool{}
	visited := map[ID]bool{}
	var walk func(ID) error
	walk = func(id ID) error {
		if visiting[id] {
			return fmt.Errorf("entity cycle reaches %q", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, child := range d.Entities[id].Children {
			if err := walk(child); err != nil {
				return err
			}
		}
		delete(visiting, id)
		visited[id] = true
		return nil
	}
	for _, root := range d.RootIDs {
		if err := walk(root); err != nil {
			return err
		}
	}
	if len(visited) != len(d.Entities) {
		missing := make([]string, 0, len(d.Entities)-len(visited))
		for id := range d.Entities {
			if !visited[id] {
				missing = append(missing, string(id))
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("entities are unreachable from roots: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validatePrefabDefinition(prefab PrefabDefinition, materials map[ID]Material, assets map[ID]AssetRecord) error {
	if prefab.ID == "" || strings.TrimSpace(prefab.Name) == "" || prefab.Root == "" || len(prefab.Entities) == 0 {
		return fmt.Errorf("id, name, root, and entities are required")
	}
	root, ok := prefab.Entities[prefab.Root]
	if !ok || root.Parent != "" {
		return fmt.Errorf("root must exist and have no parent")
	}
	_ = root
	for key, entity := range prefab.Entities {
		if key == "" || entity.ID != key {
			return fmt.Errorf("entity key %q does not match id %q", key, entity.ID)
		}
		if entity.Prefab != nil {
			return fmt.Errorf("nested prefab instances are not supported")
		}
		if entity.Mesh != nil {
			if _, ok := materials[entity.Mesh.Material]; !ok {
				return fmt.Errorf("entity %q references missing material %q", key, entity.Mesh.Material)
			}
			if err := validateGeometry(entity.Mesh.Geometry); err != nil {
				return err
			}
			if err := validateModifiers(entity.Mesh.Geometry, entity.Mesh.Modifiers); err != nil {
				return err
			}
		}
		if entity.Model != nil {
			asset, ok := assets[entity.Model.Asset]
			if !ok {
				return fmt.Errorf("entity %q references missing asset %q", key, entity.Model.Asset)
			}
			if asset.Format != "glb" && asset.Format != "gltf" {
				return fmt.Errorf("entity %q model asset %q is not glb/gltf", key, asset.ID)
			}
		}
		if entity.Parent != "" {
			parent, ok := prefab.Entities[entity.Parent]
			if !ok || !containsID(parent.Children, key) {
				return fmt.Errorf("entity %q has inconsistent parent %q", key, entity.Parent)
			}
		}
		for _, child := range entity.Children {
			value, ok := prefab.Entities[child]
			if !ok || value.Parent != key {
				return fmt.Errorf("entity %q has inconsistent child %q", key, child)
			}
		}
	}
	visited := map[ID]bool{}
	visiting := map[ID]bool{}
	var walk func(ID) error
	walk = func(id ID) error {
		if visiting[id] {
			return fmt.Errorf("entity cycle reaches %q", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, child := range prefab.Entities[id].Children {
			if err := walk(child); err != nil {
				return err
			}
		}
		delete(visiting, id)
		visited[id] = true
		return nil
	}
	if err := walk(prefab.Root); err != nil {
		return err
	}
	if len(visited) != len(prefab.Entities) {
		return fmt.Errorf("entities are unreachable from root")
	}
	return nil
}

func finiteTransform(value Transform) bool {
	return finiteVec3(value.Position) && finiteVec3(value.Rotation) && finiteVec3(value.Scale)
}

func validateModifiers(geometry Geometry, modifiers []Modifier) error {
	if len(modifiers) > 0 && strings.ToLower(strings.TrimSpace(geometry.Kind)) != "indexed-mesh" {
		return fmt.Errorf("modifiers currently require indexed-mesh geometry")
	}
	seen := map[ID]bool{}
	for _, modifier := range modifiers {
		if modifier.ID == "" || seen[modifier.ID] {
			return fmt.Errorf("modifier ids must be non-empty and unique")
		}
		seen[modifier.ID] = true
		switch modifier.Kind {
		case "mirror":
			if modifier.Axis != "x" && modifier.Axis != "y" && modifier.Axis != "z" {
				return fmt.Errorf("mirror modifier %q requires axis x, y, or z", modifier.ID)
			}
		case "array":
			if modifier.Count < 2 || modifier.Count > 1000 || !finiteVec3(modifier.Offset) || modifier.Offset == (Vec3{}) {
				return fmt.Errorf("array modifier %q requires count 2..1000 and a finite non-zero offset", modifier.ID)
			}
		case "solidify":
			if !finite(modifier.Thickness) || modifier.Thickness <= 0 || modifier.Thickness > 1_000_000 {
				return fmt.Errorf("solidify modifier %q requires finite thickness in (0, 1000000]", modifier.ID)
			}
		case "subdivision":
			if modifier.Levels < 1 || modifier.Levels > 4 {
				return fmt.Errorf("subdivision modifier %q requires levels 1..4", modifier.ID)
			}
		default:
			return fmt.Errorf("modifier %q has unsupported kind %q", modifier.ID, modifier.Kind)
		}
	}
	return nil
}

func containsID(values []ID, target ID) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func supportedGeometry(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "box", "plane", "sphere", "cylinder", "indexed-mesh", "nurbs-curve":
		return true
	default:
		return false
	}
}

func validateGeometry(geometry Geometry) error {
	kind := strings.ToLower(strings.TrimSpace(geometry.Kind))
	if kind == "nurbs-curve" {
		return validateCurveGeometry(geometry.Curve)
	}
	if kind != "indexed-mesh" {
		return nil
	}
	if len(geometry.Vertices) < 3 || len(geometry.Faces) == 0 {
		return fmt.Errorf("indexed-mesh requires vertices and faces")
	}
	vertices := make(map[ID]bool, len(geometry.Vertices))
	for _, vertex := range geometry.Vertices {
		if vertex.ID == "" || vertices[vertex.ID] {
			return fmt.Errorf("vertex ids must be non-empty and unique")
		}
		if !finiteVec3(vertex.Position) || !finiteVec3(vertex.Normal) {
			return fmt.Errorf("vertex %q contains a non-finite vector", vertex.ID)
		}
		if vertex.UV != nil && (!finite(vertex.UV.X) || !finite(vertex.UV.Y)) {
			return fmt.Errorf("vertex %q contains a non-finite uv", vertex.ID)
		}
		vertices[vertex.ID] = true
	}
	faces := make(map[ID]bool, len(geometry.Faces))
	for _, face := range geometry.Faces {
		if face.ID == "" || faces[face.ID] || len(face.Vertices) < 3 {
			return fmt.Errorf("face ids must be unique and contain at least three vertices")
		}
		faces[face.ID] = true
		for _, vertexID := range face.Vertices {
			if !vertices[vertexID] {
				return fmt.Errorf("face %q references missing vertex %q", face.ID, vertexID)
			}
		}
	}
	return nil
}

func validateCurveGeometry(curve *CurveGeometry) error {
	if curve == nil {
		return fmt.Errorf("nurbs-curve requires curve data")
	}
	count := len(curve.ControlPoints)
	if count < 2 || curve.Degree < 1 || curve.Degree >= count {
		return fmt.Errorf("nurbs-curve degree must be between one and controlPoints-1")
	}
	if len(curve.Knots) != count+curve.Degree+1 {
		return fmt.Errorf("nurbs-curve requires controlPoints+degree+1 knots")
	}
	for index, knot := range curve.Knots {
		if !finite(knot) || (index > 0 && knot < curve.Knots[index-1]) {
			return fmt.Errorf("nurbs-curve knots must be finite and nondecreasing")
		}
	}
	if curve.Knots[curve.Degree] >= curve.Knots[count] {
		return fmt.Errorf("nurbs-curve has an empty parameter domain")
	}
	seen := map[ID]bool{}
	for _, point := range curve.ControlPoints {
		if point.ID == "" || seen[point.ID] || !finiteVec3(point.Position) || !finite(point.Weight) || point.Weight <= 0 {
			return fmt.Errorf("nurbs-curve control points require unique ids, finite positions, and positive weights")
		}
		seen[point.ID] = true
	}
	if curve.Segments < 2 || curve.Radius <= 0 || curve.RadialSegments < 3 {
		return fmt.Errorf("nurbs-curve requires at least two path segments, positive radius, and three radial segments")
	}
	return nil
}

func finiteVec3(value Vec3) bool {
	return finite(value.X) && finite(value.Y) && finite(value.Z)
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
