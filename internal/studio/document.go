package studio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const SceneDocSchema = "gosx3d.scene-document/v1"

type ID string

type Document struct {
	Schema      string            `json:"schema"`
	ID          ID                `json:"id"`
	Name        string            `json:"name"`
	Revision    uint64            `json:"revision"`
	RootIDs     []ID              `json:"rootIds"`
	Entities    map[ID]Entity     `json:"entities"`
	Materials   map[ID]Material   `json:"materials"`
	Camera      Camera            `json:"camera"`
	Environment Environment       `json:"environment"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type Entity struct {
	ID        ID              `json:"id"`
	Name      string          `json:"name"`
	Parent    ID              `json:"parent,omitempty"`
	Children  []ID            `json:"children,omitempty"`
	Transform Transform       `json:"transform"`
	Mesh      *MeshComponent  `json:"mesh,omitempty"`
	Light     *LightComponent `json:"light,omitempty"`
	Visible   bool            `json:"visible"`
	Locked    bool            `json:"locked,omitempty"`
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

type MeshComponent struct {
	Geometry      Geometry `json:"geometry"`
	Material      ID       `json:"material"`
	Pickable      bool     `json:"pickable"`
	CastShadow    bool     `json:"castShadow,omitempty"`
	ReceiveShadow bool     `json:"receiveShadow,omitempty"`
}

type Geometry struct {
	Kind           string  `json:"kind"`
	Width          float64 `json:"width,omitempty"`
	Height         float64 `json:"height,omitempty"`
	Depth          float64 `json:"depth,omitempty"`
	Radius         float64 `json:"radius,omitempty"`
	Segments       int     `json:"segments,omitempty"`
	RadiusTop      float64 `json:"radiusTop,omitempty"`
	RadiusBottom   float64 `json:"radiusBottom,omitempty"`
	RadialSegments int     `json:"radialSegments,omitempty"`
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
	ID           ID      `json:"id"`
	Name         string  `json:"name"`
	Color        string  `json:"color"`
	Roughness    float64 `json:"roughness"`
	Metalness    float64 `json:"metalness"`
	Clearcoat    float64 `json:"clearcoat,omitempty"`
	Transmission float64 `json:"transmission,omitempty"`
	Emissive     float64 `json:"emissive,omitempty"`
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
		if material.Roughness < 0 || material.Roughness > 1 || material.Metalness < 0 || material.Metalness > 1 || material.Transmission < 0 || material.Transmission > 1 {
			return fmt.Errorf("material %q has a value outside [0,1]", key)
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
	case "box", "plane", "sphere", "cylinder":
		return true
	default:
		return false
	}
}
