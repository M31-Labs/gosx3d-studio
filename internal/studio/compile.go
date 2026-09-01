package studio

import (
	"fmt"
	"math"
	"strings"

	"m31labs.dev/gosx/scene"
)

func Compile(document Document) (scene.Props, error) {
	return CompileSelected(document, "")
}

func CompileSelected(document Document, selected ID) (scene.Props, error) {
	if err := document.Validate(); err != nil {
		return scene.Props{}, err
	}
	var resolve func(ID) (scene.Node, error)
	resolve = func(id ID) (scene.Node, error) {
		return compileEntity(document, id, selected, resolve)
	}
	nodes := make([]scene.Node, 0, len(document.RootIDs))
	for _, id := range document.RootIDs {
		node, err := resolve(id)
		if err != nil {
			return scene.Props{}, err
		}
		nodes = append(nodes, node)
	}
	return appendSelectionGizmo(document, selected, compileProps(document, nodes)), nil
}

// appendSelectionGizmo attaches the engine TransformControls helper to the
// current selection. It is a visual helper surface whose mode follows
// GizmoInputSignal live; pointer-drag mutation is tracked engine work, so
// transform commits stay on the Inspector/agent transaction path.
func appendSelectionGizmo(document Document, selected ID, props scene.Props) scene.Props {
	if _, ok := document.Entities[selected]; !ok || selected == "" {
		return props
	}
	props.Graph.Nodes = append(props.Graph.Nodes, scene.TransformControls{
		ID:       "studio-gizmo",
		Target:   string(selected),
		Mode:     "translate",
		Size:     1.2,
		Position: toSceneVec(worldPosition(document, selected)),
	})
	return props
}

// worldPosition composes the parent transform chain so root-level helpers can
// track entities that sit inside groups.
func worldPosition(document Document, id ID) Vec3 {
	chain := make([]Transform, 0, 4)
	for current := id; current != ""; {
		entity, ok := document.Entities[current]
		if !ok {
			break
		}
		chain = append(chain, entity.Transform)
		current = entity.Parent
	}
	matrix := identityMatrix()
	for i := len(chain) - 1; i >= 0; i-- {
		matrix = multiplyMatrix(matrix, transformMatrix(chain[i]))
	}
	return Vec3{X: matrix[3], Y: matrix[7], Z: matrix[11]}
}

func compileProps(document Document, nodes []scene.Node) scene.Props {
	return scene.Props{
		Label:                document.Name,
		AriaLabel:            document.Name + " Scene3D viewport",
		Background:           document.Environment.Background,
		Controls:             scene.ControlOrbit,
		Responsive:           scene.Bool(true),
		FillHeight:           scene.Bool(true),
		PickSignalNamespace:  "studio.viewport",
		SelectionInputSignal: "studio.viewport.selectedID",
		GizmoInputSignal:     "studio.viewport.gizmoMode",
		CameraInputSignal:    "studio.viewport.cameraIn",
		CameraOutputSignal:   "studio.viewport.cameraOut",
		Camera: scene.PerspectiveCamera{
			Position: toSceneVec(document.Camera.Position),
			Rotation: toSceneEuler(document.Camera.Rotation),
			FOV:      document.Camera.FOV, Near: document.Camera.Near, Far: document.Camera.Far,
		},
		Environment: scene.Environment{
			AmbientColor:     document.Environment.AmbientColor,
			AmbientIntensity: document.Environment.AmbientIntensity,
			Exposure:         document.Environment.Exposure,
			ToneMapping:      document.Environment.ToneMapping,
		},
		Graph: scene.NewGraph(nodes...),
	}
}

// SceneIR is the shared GoSX SceneIR plus the Studio-owned render-graph plan.
// The embedded framework IR is the canonical scene contract and stays byte
// for byte what GoSX emits; the plan rides beside it under the same
// "renderGraph" key it used when GoSX still declared the type. Studio never
// adds scene semantics here — only the lowering GoSX has no consumer for.
type SceneIR struct {
	scene.SceneIR
	RenderGraph *RenderGraphIR `json:"renderGraph,omitempty"`
}

// CompileIR emits the portable SceneIR artifact shape. Props.SceneIR keeps the
// schema optional for same-version runtime traffic; Studio artifacts cross
// process and persistence boundaries, so they always carry the schema.
func CompileIR(document Document) (SceneIR, error) {
	props, err := Compile(document)
	if err != nil {
		return SceneIR{}, err
	}
	ir := SceneIR{SceneIR: props.SceneIR()}
	ir.Schema = scene.SceneIRSchema
	if document.RenderGraph != nil {
		plan, err := CompileRenderGraph(*document.RenderGraph)
		if err != nil {
			return SceneIR{}, err
		}
		ir.RenderGraph = &plan
	}
	return ir, nil
}

func compileEntity(document Document, id ID, selected ID, resolve func(ID) (scene.Node, error)) (scene.Node, error) {
	entity := document.Entities[id]
	if entity.Light != nil && !unitScale(entity.Transform.Scale) {
		return nil, fmt.Errorf("light entity %q has scale %+v; light scale has no render meaning", id, entity.Transform.Scale)
	}
	children := make([]scene.Node, 0, len(entity.Children))
	for _, childID := range entity.Children {
		child, err := resolve(childID)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	if entity.Prefab != nil {
		definition, err := resolvePrefabDefinition(document.Prefabs, entity.Prefab.Prefab)
		if err != nil {
			return nil, fmt.Errorf("entity %q prefab: %w", id, err)
		}
		prefabRoot, err := compilePrefabEntity(document, definition, definition.Root, entity.ID, entity.Prefab.Overrides, selected)
		if err != nil {
			return nil, fmt.Errorf("entity %q prefab %q: %w", id, definition.ID, err)
		}
		children = append([]scene.Node{prefabRoot}, children...)
		return scene.Group{
			ID: string(entity.ID), Position: toSceneVec(entity.Transform.Position), Rotation: toSceneRotation(entity.Transform.Rotation),
			Scale: toSceneVec(entity.Transform.Scale), Children: children,
		}, nil
	}
	return compileEntityValue(document, entity, entity.ID, selected == id, children)
}

func compilePrefabEntity(document Document, definition PrefabDefinition, localID, instanceID ID, overrides map[ID]PrefabEntityOverride, selected ID) (scene.Node, error) {
	entity := definition.Entities[localID]
	if override, ok := overrides[localID]; ok {
		if override.Transform != nil {
			entity.Transform = *override.Transform
		}
		if override.Material != "" && entity.Mesh != nil {
			mesh := *entity.Mesh
			mesh.Material = override.Material
			entity.Mesh = &mesh
		}
		if override.Visible != nil {
			entity.Visible = *override.Visible
		}
	}
	children := make([]scene.Node, 0, len(entity.Children))
	for _, childID := range entity.Children {
		child, err := compilePrefabEntity(document, definition, childID, instanceID, overrides, selected)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	runtimeID := ID(fmt.Sprintf("%s--%s", instanceID, localID))
	if entity.Prefab != nil {
		// Nested instance: resolve its definition and recurse with the
		// namespaced runtime ID so every level keeps stable identity.
		nested, err := resolvePrefabDefinition(document.Prefabs, entity.Prefab.Prefab)
		if err != nil {
			return nil, fmt.Errorf("nested prefab %q: %w", localID, err)
		}
		nestedRoot, err := compilePrefabEntity(document, nested, nested.Root, runtimeID, entity.Prefab.Overrides, selected)
		if err != nil {
			return nil, fmt.Errorf("nested prefab %q: %w", localID, err)
		}
		children = append([]scene.Node{nestedRoot}, children...)
		return scene.Group{
			ID: string(runtimeID), Position: toSceneVec(entity.Transform.Position), Rotation: toSceneRotation(entity.Transform.Rotation),
			Scale: toSceneVec(entity.Transform.Scale), Children: children,
		}, nil
	}
	return compileEntityValue(document, entity, runtimeID, selected == runtimeID, children)
}

func compileEntityValue(document Document, entity Entity, runtimeID ID, selected bool, children []scene.Node) (scene.Node, error) {
	if entity.Light != nil && !unitScale(entity.Transform.Scale) {
		return nil, fmt.Errorf("light entity %q has scale %+v; light scale has no render meaning", runtimeID, entity.Transform.Scale)
	}
	position := toSceneVec(entity.Transform.Position)
	rotation := toSceneRotation(entity.Transform.Rotation)
	if entity.Mesh != nil {
		authored := entity.Mesh.Geometry
		var err error
		if entity.Skin != nil {
			authored, _, err = deformSkinGeometry(document, entity, runtimeID)
			if err != nil {
				return nil, fmt.Errorf("entity %q: %w", runtimeID, err)
			}
		}
		evaluated, err := evaluateModifiers(authored, entity.Mesh.Modifiers)
		if err != nil {
			return nil, fmt.Errorf("entity %q: %w", runtimeID, err)
		}
		geometry, err := compileGeometry(evaluated)
		if err != nil {
			return nil, fmt.Errorf("entity %q: %w", runtimeID, err)
		}
		material := document.Materials[entity.Mesh.Material]
		standard := scene.StandardMaterial{
			Color: material.Color, Roughness: material.Roughness, Metalness: material.Metalness,
			Clearcoat: material.Clearcoat, Transmission: material.Transmission, Emissive: material.Emissive,
		}
		for channel, slot := range material.Textures {
			uri := document.Assets[slot.Asset].URI
			switch channel {
			case "color":
				standard.Texture = uri
			case "normal":
				standard.NormalMap = uri
			case "roughness":
				standard.RoughnessMap = uri
			case "metalness":
				standard.MetalnessMap = uri
			case "emissive":
				standard.EmissiveMap = uri
			}
		}
		var compiledMaterial scene.Material = standard
		if material.Selena != nil {
			compiled, _, err := scene.CompileSelenaMaterial([]byte(material.Selena.Source), scene.SelenaMaterialOptions{Material: material.Selena.Material, Standard: standard})
			if err != nil {
				return nil, fmt.Errorf("entity %q material %q Selena compile: %w", runtimeID, material.ID, err)
			}
			compiledMaterial = compiled
		}
		visible := entity.Visible
		pickable := entity.Mesh.Pickable
		meshScale := scene.Vector3{}
		if !unitScale(entity.Transform.Scale) {
			meshScale = toSceneVec(entity.Transform.Scale)
		}
		return scene.Mesh{
			ID: string(runtimeID), Geometry: geometry,
			Material: compiledMaterial, Scale: meshScale,
			Position: position, Rotation: rotation, Visible: &visible, Pickable: &pickable,
			CastShadow: entity.Mesh.CastShadow, ReceiveShadow: entity.Mesh.ReceiveShadow,
			Children: children,
			Selected: selected,
		}, nil
	}
	if entity.Model != nil {
		asset := document.Assets[entity.Model.Asset]
		visible := entity.Visible
		pickable := entity.Model.Pickable
		return scene.Model{
			ID: string(runtimeID), Src: asset.URI,
			Position: position, Rotation: rotation, Scale: toSceneVec(scaleOrUnit(entity.Transform.Scale)),
			Bounds: entity.Model.Bounds, Fit: entity.Model.Fit, FitAlign: entity.Model.FitAlign,
			CastShadow: entity.Model.CastShadow, ReceiveShadow: entity.Model.ReceiveShadow,
			Pickable: &pickable, Visible: &visible,
		}, nil
	}
	if entity.Light != nil {
		light := entity.Light
		switch strings.ToLower(strings.TrimSpace(light.Kind)) {
		case "ambient":
			return scene.AmbientLight{ID: string(runtimeID), Color: light.Color, Intensity: light.Intensity}, nil
		case "directional":
			return scene.DirectionalLight{ID: string(runtimeID), Color: light.Color, Intensity: light.Intensity, Direction: toSceneVec(light.Direction), CastShadow: light.CastShadow}, nil
		case "point":
			return scene.PointLight{ID: string(runtimeID), Color: light.Color, Intensity: light.Intensity, Position: toSceneVec(light.Position), Range: light.Range}, nil
		default:
			return nil, fmt.Errorf("entity %q uses unsupported light %q", runtimeID, light.Kind)
		}
	}
	return scene.Group{ID: string(runtimeID), Position: position, Rotation: rotation, Scale: toSceneVec(entity.Transform.Scale), Children: children}, nil
}

func compileGeometry(geometry Geometry) (scene.Geometry, error) {
	switch strings.ToLower(strings.TrimSpace(geometry.Kind)) {
	case "box":
		return scene.BoxGeometry{Width: geometry.Width, Height: geometry.Height, Depth: geometry.Depth}, nil
	case "plane":
		return scene.PlaneGeometry{Width: geometry.Width, Height: geometry.Height}, nil
	case "sphere":
		return scene.SphereGeometry{Radius: geometry.Radius, Segments: geometry.Segments}, nil
	case "cylinder":
		return scene.CylinderGeometry{RadiusTop: geometry.RadiusTop, RadiusBottom: geometry.RadiusBottom, Height: geometry.Height, Segments: geometry.RadialSegments}, nil
	case "indexed-mesh":
		return compileIndexedGeometry(geometry)
	case "nurbs-curve":
		return compileNURBSCurve(geometry.Curve)
	default:
		return nil, fmt.Errorf("unsupported geometry %q", geometry.Kind)
	}
}

func compileIndexedGeometry(geometry Geometry) (scene.Geometry, error) {
	indexByID := make(map[ID]int, len(geometry.Vertices))
	positions := make([]float64, 0, len(geometry.Vertices)*3)
	normals := make([]float64, 0, len(geometry.Vertices)*3)
	hasNormals := false
	uvs := make([]float64, 0, len(geometry.Vertices)*2)
	allUVs := len(geometry.Vertices) > 0
	for index, vertex := range geometry.Vertices {
		indexByID[vertex.ID] = index
		positions = append(positions, vertex.Position.X, vertex.Position.Y, vertex.Position.Z)
		normals = append(normals, vertex.Normal.X, vertex.Normal.Y, vertex.Normal.Z)
		hasNormals = hasNormals || vertex.Normal != (Vec3{})
		if vertex.UV == nil {
			allUVs = false
			uvs = append(uvs, 0, 0)
		} else {
			uvs = append(uvs, vertex.UV.X, vertex.UV.Y)
		}
	}
	indices := make([]int, 0, len(geometry.Faces)*3)
	for _, face := range geometry.Faces {
		first := indexByID[face.Vertices[0]]
		for index := 1; index+1 < len(face.Vertices); index++ {
			indices = append(indices, first, indexByID[face.Vertices[index]], indexByID[face.Vertices[index+1]])
		}
	}
	if !hasNormals {
		normals = nil
	}
	if !allUVs {
		uvs = nil
	}
	return scene.BufferGeometry{Positions: positions, Normals: normals, UVs: uvs, Indices: indices}, nil
}

func unitScale(scale Vec3) bool {
	return near(scale.X, 1) && near(scale.Y, 1) && near(scale.Z, 1)
}

func near(left, right float64) bool { return math.Abs(left-right) < 1e-9 }

func toSceneVec(value Vec3) scene.Vector3 { return scene.Vec3(value.X, value.Y, value.Z) }
func toSceneEuler(value Vec3) scene.Euler { return scene.Rotate(value.X, value.Y, value.Z) }

// toSceneRotation lowers the authoritative quaternion through the engine's
// euler contract; the engine rebuilds the same quaternion internally because
// both sides share the Rz*Ry*Rx convention.
func toSceneRotation(value Quaternion) scene.Euler { return toSceneEuler(value.Euler()) }

// scaleOrUnit treats the zero-value scale as unit for lowering into engine
// records that require an explicit scale.
func scaleOrUnit(scale Vec3) Vec3 {
	if scale == (Vec3{}) {
		return Vec3{X: 1, Y: 1, Z: 1}
	}
	return scale
}
