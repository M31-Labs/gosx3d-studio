package studio

import (
	"fmt"
	"math"
	"strings"

	"m31labs.dev/gosx/scene"
)

func Compile(document Document) (scene.Props, error) {
	if err := document.Validate(); err != nil {
		return scene.Props{}, err
	}
	nodes := make([]scene.Node, 0, len(document.RootIDs))
	for _, id := range document.RootIDs {
		node, err := compileEntity(document, id)
		if err != nil {
			return scene.Props{}, err
		}
		nodes = append(nodes, node)
	}
	return scene.Props{
		Label:      document.Name,
		AriaLabel:  document.Name + " Scene3D viewport",
		Background: document.Environment.Background,
		Controls:   scene.ControlOrbit,
		Responsive: scene.Bool(true),
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
	}, nil
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
	return ir, nil
}

func compileEntity(document Document, id ID) (scene.Node, error) {
	entity := document.Entities[id]
	if !unitScale(entity.Transform.Scale) {
		return nil, fmt.Errorf("entity %q has scale %+v; SceneDoc scale compilation is not implemented", id, entity.Transform.Scale)
	}
	children := make([]scene.Node, 0, len(entity.Children))
	for _, childID := range entity.Children {
		child, err := compileEntity(document, childID)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	position := toSceneVec(entity.Transform.Position)
	rotation := toSceneEuler(entity.Transform.Rotation)
	if entity.Mesh != nil {
		geometry, err := compileGeometry(entity.Mesh.Geometry)
		if err != nil {
			return nil, fmt.Errorf("entity %q: %w", id, err)
		}
		material := document.Materials[entity.Mesh.Material]
		visible := entity.Visible
		pickable := entity.Mesh.Pickable
		return scene.Mesh{
			ID: string(entity.ID), Geometry: geometry,
			Material: scene.StandardMaterial{
				Color: material.Color, Roughness: material.Roughness, Metalness: material.Metalness,
				Clearcoat: material.Clearcoat, Transmission: material.Transmission, Emissive: material.Emissive,
			},
			Position: position, Rotation: rotation, Visible: &visible, Pickable: &pickable,
			CastShadow: entity.Mesh.CastShadow, ReceiveShadow: entity.Mesh.ReceiveShadow,
			Children: children,
		}, nil
	}
	if entity.Light != nil {
		light := entity.Light
		switch strings.ToLower(strings.TrimSpace(light.Kind)) {
		case "ambient":
			return scene.AmbientLight{ID: string(entity.ID), Color: light.Color, Intensity: light.Intensity}, nil
		case "directional":
			return scene.DirectionalLight{ID: string(entity.ID), Color: light.Color, Intensity: light.Intensity, Direction: toSceneVec(light.Direction), CastShadow: light.CastShadow}, nil
		case "point":
			return scene.PointLight{ID: string(entity.ID), Color: light.Color, Intensity: light.Intensity, Position: toSceneVec(light.Position), Range: light.Range}, nil
		default:
			return nil, fmt.Errorf("entity %q uses unsupported light %q", id, light.Kind)
		}
	}
	return scene.Group{ID: string(entity.ID), Position: position, Rotation: rotation, Children: children}, nil
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
	default:
		return nil, fmt.Errorf("unsupported geometry %q", geometry.Kind)
	}
}

func unitScale(scale Vec3) bool {
	return near(scale.X, 1) && near(scale.Y, 1) && near(scale.Z, 1)
}

func near(left, right float64) bool { return math.Abs(left-right) < 1e-9 }

func toSceneVec(value Vec3) scene.Vector3 { return scene.Vec3(value.X, value.Y, value.Z) }
func toSceneEuler(value Vec3) scene.Euler { return scene.Rotate(value.X, value.Y, value.Z) }
