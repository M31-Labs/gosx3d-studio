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
	return compileProps(document, nodes), nil
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

// CompileIR emits the portable SceneIR artifact shape. Props.SceneIR keeps the
// schema optional for same-version runtime traffic; Studio artifacts cross
// process and persistence boundaries, so they always carry the schema.
func CompileIR(document Document) (scene.SceneIR, error) {
	props, err := Compile(document)
	if err != nil {
		return scene.SceneIR{}, err
	}
	ir := props.SceneIR()
	ir.Schema = scene.SceneIRSchema
	if document.RenderGraph != nil {
		plan, err := CompileRenderGraph(*document.RenderGraph)
		if err != nil {
			return scene.SceneIR{}, err
		}
		ir.RenderGraph = &plan
	}
	return ir, nil
}

func compileEntity(document Document, id ID, selected ID, resolve func(ID) (scene.Node, error)) (scene.Node, error) {
	entity := document.Entities[id]
	if !unitScale(entity.Transform.Scale) {
		return nil, fmt.Errorf("entity %q has scale %+v; SceneDoc scale compilation is not implemented", id, entity.Transform.Scale)
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
		definition := document.Prefabs[entity.Prefab.Prefab]
		prefabRoot, err := compilePrefabEntity(document, definition, definition.Root, entity.ID, entity.Prefab.Overrides, selected)
		if err != nil {
			return nil, fmt.Errorf("entity %q prefab %q: %w", id, definition.ID, err)
		}
		children = append([]scene.Node{prefabRoot}, children...)
		return scene.Group{ID: string(entity.ID), Position: toSceneVec(entity.Transform.Position), Rotation: toSceneEuler(entity.Transform.Rotation), Children: children}, nil
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
			entity.Mesh.Material = override.Material
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
	return compileEntityValue(document, entity, runtimeID, selected == runtimeID, children)
}

func compileEntityValue(document Document, entity Entity, runtimeID ID, selected bool, children []scene.Node) (scene.Node, error) {
	if !unitScale(entity.Transform.Scale) {
		return nil, fmt.Errorf("entity %q has scale %+v; SceneDoc scale compilation is not implemented", runtimeID, entity.Transform.Scale)
	}
	position := toSceneVec(entity.Transform.Position)
	rotation := toSceneEuler(entity.Transform.Rotation)
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
		return scene.Mesh{
			ID: string(runtimeID), Geometry: geometry,
			Material: compiledMaterial,
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
			Position: position, Rotation: rotation, Scale: scene.Vector3{X: 1, Y: 1, Z: 1},
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
	return scene.Group{ID: string(runtimeID), Position: position, Rotation: rotation, Children: children}, nil
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
