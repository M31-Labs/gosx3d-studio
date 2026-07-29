package studio

import (
	"fmt"
	"math"
	"sort"
)

const checkerSpacing = 0.52

type boardHole struct {
	index    int
	position Vec3
}

// SampleDocument is the Studio-owned conversion of the shipped Chinese
// Checkers showcase. Every renderable record has a stable authoring ID.
func SampleDocument() Document {
	entities := map[ID]Entity{}
	root := Entity{ID: "scene-root", Name: "Chinese Checkers", Transform: IdentityTransform(), Visible: true}
	add := func(entity Entity) {
		entity.Parent = root.ID
		root.Children = append(root.Children, entity.ID)
		entities[entity.ID] = entity
	}
	add(Entity{ID: "ambient-light", Name: "Ambient", Transform: IdentityTransform(), Visible: true, Light: &LightComponent{Kind: "ambient", Color: "#d8e7ef", Intensity: 0.42}})
	add(Entity{ID: "key-light", Name: "Key", Transform: IdentityTransform(), Visible: true, Light: &LightComponent{Kind: "directional", Color: "#fff0d2", Intensity: 1.35, Direction: Vec3{X: -0.45, Y: -1, Z: -0.35}, CastShadow: true}})
	add(Entity{ID: "rim-light", Name: "Rim", Transform: IdentityTransform(), Visible: true, Light: &LightComponent{Kind: "point", Color: "#8edbc4", Intensity: 0.75, Position: Vec3{X: 3.5, Y: 4.8, Z: -3}, Range: 16}})
	add(Entity{ID: "fill-light", Name: "Fill", Transform: IdentityTransform(), Visible: true, Light: &LightComponent{Kind: "point", Color: "#ffb66f", Intensity: 0.48, Position: Vec3{X: -4.5, Y: 2.6, Z: 3.2}, Range: 13}})
	add(meshEntity("board", "Board", Vec3{Y: -0.22}, Geometry{Kind: "cylinder", RadiusTop: 4.15, RadiusBottom: 4.25, Height: 0.32, RadialSegments: 48}, "board-material", true))
	add(meshEntity("board-pedestal", "Pedestal", Vec3{Y: -0.52}, Geometry{Kind: "cylinder", RadiusTop: 3.76, RadiusBottom: 3.98, Height: 0.42, RadialSegments: 48}, "pedestal-material", false))
	holes := checkerHoles()
	for _, hole := range holes {
		add(meshEntity(ID(fmt.Sprintf("socket-%03d", hole.index)), fmt.Sprintf("Socket %03d", hole.index), Vec3{X: hole.position.X, Y: -0.015, Z: hole.position.Z}, Geometry{Kind: "cylinder", RadiusTop: 0.145, RadiusBottom: 0.145, Height: 0.07, RadialSegments: 16}, "socket-material", false))
	}
	camps := initialCamps(holes)
	for _, player := range []int{0, 3} {
		for index, position := range camps[player] {
			id := ID(fmt.Sprintf("piece-player-%d-%02d", player+1, index+1))
			add(meshEntity(id, fmt.Sprintf("Player %d Piece %02d", player+1, index+1), Vec3{X: position.X, Y: 0.2, Z: position.Z}, Geometry{Kind: "sphere", Radius: 0.205, Segments: 24}, ID(fmt.Sprintf("player-%d-material", player+1)), true))
		}
	}
	entities[root.ID] = root
	return Document{Schema: SceneDocSchema, ID: "chinese-checkers-showcase", Name: "Chinese Checkers Showcase", Revision: 1, RootIDs: []ID{root.ID}, Entities: entities, Materials: checkerMaterials(), Camera: Camera{Position: Vec3{Y: 8.2, Z: 10.4}, FOV: 45, Near: 0.1, Far: 80}, Environment: Environment{Background: "#080b0f", AmbientColor: "#ffffff", AmbientIntensity: 0.16, Exposure: 1.05, ToneMapping: "aces"}, Metadata: map[string]string{"proof": "m0-chinese-checkers", "source": "gosx/examples/gosx-docs/app/demos/checkers"}}
}

func meshEntity(id ID, name string, position Vec3, geometry Geometry, material ID, pickable bool) Entity {
	return Entity{ID: id, Name: name, Transform: TransformFromEuler(position, Vec3{}, Vec3{X: 1, Y: 1, Z: 1}), Visible: true, Mesh: &MeshComponent{Geometry: geometry, Material: material, Pickable: pickable, CastShadow: true, ReceiveShadow: true}}
}
func checkerMaterials() map[ID]Material {
	return map[ID]Material{
		"board-material":    {ID: "board-material", Name: "Carved Wood", Color: "#49301f", Roughness: 0.72, Metalness: 0.03, Clearcoat: 0.16},
		"pedestal-material": {ID: "pedestal-material", Name: "Dark Pedestal", Color: "#111518", Roughness: 0.3, Metalness: 0.78, Clearcoat: 0.38},
		"socket-material":   {ID: "socket-material", Name: "Socket", Color: "#121719", Roughness: 0.34, Metalness: 0.42, Clearcoat: 0.48},
		"player-1-material": {ID: "player-1-material", Name: "Coral Pieces", Color: "#e66b62", Roughness: 0.2, Metalness: 0.08, Clearcoat: 0.68, Transmission: 0.04, Emissive: 0.025, Selena: &SelenaShader{Material: "CoralPiece", Source: `
material CoralPiece {
  param tint : color = rgb(0.902, 0.420, 0.384)
  surface(geo) -> color { return vec4f(tint.rgb, 1.0) }
}`}},
		"player-4-material": {ID: "player-4-material", Name: "Cobalt Pieces", Color: "#60a8dc", Roughness: 0.2, Metalness: 0.08, Clearcoat: 0.68, Transmission: 0.04, Emissive: 0.025},
	}
}
func checkerHoles() []boardHole {
	counts := [...]int{1, 2, 3, 4, 13, 12, 11, 10, 9, 10, 11, 12, 13, 4, 3, 2, 1}
	holes := make([]boardHole, 0, 121)
	for row, count := range counts {
		z := (float64(row) - 8) * checkerSpacing * math.Sqrt(3) / 2
		for column := 0; column < count; column++ {
			x := (float64(column) - float64(count-1)/2) * checkerSpacing
			holes = append(holes, boardHole{len(holes), Vec3{X: x, Z: z}})
		}
	}
	return holes
}
func initialCamps(holes []boardHole) [6][]Vec3 {
	var camps [6][]Vec3
	used := map[int]bool{}
	for player := 0; player < 6; player++ {
		angle := -math.Pi/2 + float64(player)*math.Pi/3
		dx, dz := math.Cos(angle), math.Sin(angle)
		ordered := append([]boardHole(nil), holes...)
		sort.SliceStable(ordered, func(i, j int) bool {
			a := ordered[i].position.X*dx + ordered[i].position.Z*dz
			b := ordered[j].position.X*dx + ordered[j].position.Z*dz
			if math.Abs(a-b) < 1e-9 {
				return ordered[i].index < ordered[j].index
			}
			return a > b
		})
		for _, hole := range ordered {
			if used[hole.index] {
				continue
			}
			used[hole.index] = true
			camps[player] = append(camps[player], hole.position)
			if len(camps[player]) == 10 {
				break
			}
		}
	}
	return camps
}

func FirstPickTarget(document Document) (ID, Vec3) {
	visited := map[ID]bool{}
	var visit func(ID) (ID, Vec3, bool)
	visit = func(id ID) (ID, Vec3, bool) {
		if visited[id] {
			return "", Vec3{}, false
		}
		entity, ok := document.Entities[id]
		if !ok {
			return "", Vec3{}, false
		}
		visited[id] = true
		if entity.Mesh != nil && entity.Mesh.Pickable && id != "board" {
			return id, entity.Transform.Position, true
		}
		for _, child := range entity.Children {
			if target, position, ok := visit(child); ok {
				return target, position, true
			}
		}
		return "", Vec3{}, false
	}
	for _, root := range document.RootIDs {
		if target, position, ok := visit(root); ok {
			return target, position
		}
	}
	for _, root := range document.RootIDs {
		if entity, ok := document.Entities[root]; ok {
			return root, entity.Transform.Position
		}
	}
	return "", Vec3{}
}
