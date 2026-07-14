package studio

func SampleDocument() Document {
	entities := map[ID]Entity{}
	root := Entity{ID: "scene-root", Name: "Scene Root", Transform: IdentityTransform(), Visible: true}
	addChild := func(entity Entity) {
		entity.Parent = root.ID
		root.Children = append(root.Children, entity.ID)
		entities[entity.ID] = entity
	}
	addChild(Entity{ID: "ambient-light", Name: "Ambient", Transform: IdentityTransform(), Visible: true, Light: &LightComponent{Kind: "ambient", Color: "#d8d2c5", Intensity: 0.45}})
	addChild(Entity{ID: "key-light", Name: "Key", Transform: IdentityTransform(), Visible: true, Light: &LightComponent{Kind: "directional", Color: "#ffd6a3", Intensity: 1.4, Direction: Vec3{X: -0.35, Y: -1, Z: -0.25}, CastShadow: true}})
	addChild(Entity{ID: "board", Name: "Board", Transform: Transform{Position: Vec3{Y: -0.3}, Scale: Vec3{X: 1, Y: 1, Z: 1}}, Visible: true, Mesh: &MeshComponent{Geometry: Geometry{Kind: "box", Width: 7.2, Height: 0.5, Depth: 6.4}, Material: "board-material", Pickable: true, CastShadow: true, ReceiveShadow: true}})
	pieces := []struct {
		id       ID
		name     string
		position Vec3
		material ID
	}{
		{"piece-jade-01", "Jade Piece 01", Vec3{X: 0, Y: 0.35, Z: 0}, "jade-material"},
		{"piece-jade-02", "Jade Piece 02", Vec3{X: 0.9, Y: 0.35, Z: 0.65}, "jade-material"},
		{"piece-jade-03", "Jade Piece 03", Vec3{X: -0.9, Y: 0.35, Z: 0.65}, "jade-material"},
		{"piece-gold-01", "Gold Piece 01", Vec3{X: 0, Y: 0.35, Z: 1.35}, "gold-material"},
		{"piece-blue-01", "Blue Piece 01", Vec3{X: -1.8, Y: 0.35, Z: -0.8}, "blue-material"},
		{"piece-ivory-01", "Ivory Piece 01", Vec3{X: 1.8, Y: 0.35, Z: -0.8}, "ivory-material"},
	}
	for _, piece := range pieces {
		addChild(Entity{ID: piece.id, Name: piece.name, Transform: Transform{Position: piece.position, Scale: Vec3{X: 1, Y: 1, Z: 1}}, Visible: true, Mesh: &MeshComponent{Geometry: Geometry{Kind: "sphere", Radius: 0.42, Segments: 32}, Material: piece.material, Pickable: true, CastShadow: true, ReceiveShadow: true}})
	}
	entities[root.ID] = root
	return Document{
		Schema: SceneDocSchema, ID: "chinese-checkers-bootstrap", Name: "Chinese Checkers Bootstrap", Revision: 1,
		RootIDs: []ID{root.ID}, Entities: entities,
		Materials: map[ID]Material{
			"board-material": {ID: "board-material", Name: "Dark Wood", Color: "#30251f", Roughness: 0.58, Metalness: 0.12, Clearcoat: 0.35},
			"jade-material":  {ID: "jade-material", Name: "Selena Jade", Color: "#477d62", Roughness: 0.28, Metalness: 0.05, Clearcoat: 0.55, Transmission: 0.22},
			"gold-material":  {ID: "gold-material", Name: "Warm Gold", Color: "#c79a39", Roughness: 0.24, Metalness: 0.78, Clearcoat: 0.35},
			"blue-material":  {ID: "blue-material", Name: "Cobalt Ceramic", Color: "#3479ad", Roughness: 0.21, Metalness: 0.08, Clearcoat: 0.7},
			"ivory-material": {ID: "ivory-material", Name: "Ivory Ceramic", Color: "#d8d2c5", Roughness: 0.32, Metalness: 0.02, Clearcoat: 0.6},
		},
		Camera:      Camera{Position: Vec3{X: 0, Y: 4.8, Z: 8.5}, FOV: 48, Near: 0.1, Far: 100},
		Environment: Environment{Background: "#0b0d10", AmbientColor: "#b9b4a8", AmbientIntensity: 0.34, Exposure: 1.08, ToneMapping: "aces"},
		Metadata:    map[string]string{"proof": "bootstrap", "product": "gosx3d-studio"},
	}
}
