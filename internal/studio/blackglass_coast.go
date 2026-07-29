package studio

import "math"

// BlackglassCoastDocument is the starter authored world for the Blackglass
// Coast vertical slice. It is intentionally a readable, stable-ID blockout:
// High-quality GLB assets, vegetation batches, and WaterSystem rendering bind
// to this canonical layout rather than recreating world semantics in the app.
func BlackglassCoastDocument() Document {
	entities := map[ID]Entity{}
	root := Entity{ID: "blackglass-coast-root", Name: "Blackglass Coast", Transform: IdentityTransform(), Visible: true}
	add := func(entity Entity) {
		entity.Parent = root.ID
		root.Children = append(root.Children, entity.ID)
		entities[entity.ID] = entity
	}
	mesh := func(id ID, name string, position, scale Vec3, geometry Geometry, material ID) {
		rotation := Vec3{}
		if geometry.Kind == "plane" {
			rotation.X = -math.Pi / 2
		}
		add(Entity{
			ID:        id,
			Name:      name,
			Transform: TransformFromEuler(position, rotation, scale),
			Visible:   true,
			Mesh: &MeshComponent{
				Geometry:      geometry,
				Material:      material,
				Pickable:      geometry.Kind != "plane",
				CastShadow:    true,
				ReceiveShadow: true,
			},
		})
	}

	add(Entity{ID: "coast-sun", Name: "Volcanic Sun", Transform: IdentityTransform(), Visible: true, Light: &LightComponent{Kind: "directional", Color: "#fff0d0", Intensity: 2.1, Direction: Vec3{X: -0.55, Y: -0.82, Z: -0.24}, CastShadow: true}})
	add(Entity{ID: "coast-sky", Name: "Sky Fill", Transform: IdentityTransform(), Visible: true, Light: &LightComponent{Kind: "ambient", Color: "#b9d8e8", Intensity: 0.48}})
	add(Entity{ID: "beacon-fire", Name: "Beacon Fire", Transform: IdentityTransform(), Visible: true, Light: &LightComponent{Kind: "point", Color: "#ff8a48", Intensity: 18, Position: Vec3{X: 8, Y: 7.5, Z: -4}, Range: 26, CastShadow: true}})

	mesh("coast-sand", "Sunlit Shore", Vec3{X: -5, Y: -0.35, Z: 5}, Vec3{X: 1, Y: 1, Z: 1}, Geometry{Kind: "plane", Width: 50, Height: 36}, "sand")
	mesh("cove-water-proxy", "Blackglass Cove Water", Vec3{X: -12, Y: 0, Z: -6}, Vec3{X: 1, Y: 1, Z: 1}, Geometry{Kind: "plane", Width: 34, Height: 24}, "water-proxy")
	mesh("coast-cliff-west", "West Basalt Shelf", Vec3{X: -18, Y: 4.1, Z: -2}, Vec3{X: 8, Y: 6, Z: 5}, Geometry{Kind: "sphere", Radius: 1, Segments: 28}, "basalt")
	mesh("coast-cliff-east", "East Basalt Shelf", Vec3{X: 12, Y: 3.4, Z: -8}, Vec3{X: 7, Y: 5, Z: 4}, Geometry{Kind: "sphere", Radius: 1, Segments: 28}, "basalt")
	mesh("coast-cliff-overlook", "Overlook Cliff", Vec3{X: 4, Y: 5, Z: 11}, Vec3{X: 9, Y: 7, Z: 4}, Geometry{Kind: "sphere", Radius: 1, Segments: 28}, "basalt")
	mesh("beacon-plinth", "Beacon Plinth", Vec3{X: 8, Y: 0.8, Z: -4}, Vec3{X: 1, Y: 1, Z: 1}, Geometry{Kind: "cylinder", RadiusTop: 2.2, RadiusBottom: 2.7, Height: 1.6, RadialSegments: 32}, "ruin-stone")
	mesh("blackglass-beacon", "Blackglass Beacon", Vec3{X: 8, Y: 4.7, Z: -4}, Vec3{X: 1, Y: 1, Z: 1}, Geometry{Kind: "cylinder", RadiusTop: 0.55, RadiusBottom: 1.25, Height: 6.4, RadialSegments: 24}, "beacon-metal")
	mesh("beacon-lens", "Beacon Lens", Vec3{X: 8, Y: 8.1, Z: -4}, Vec3{X: 1, Y: 1, Z: 1}, Geometry{Kind: "sphere", Radius: 0.72, Segments: 24}, "beacon-glow")
	mesh("ruin-arch-left", "Ruin Arch Left", Vec3{X: 2.6, Y: 1.8, Z: 5.3}, Vec3{X: 0.8, Y: 3.6, Z: 0.8}, Geometry{Kind: "box", Width: 1, Height: 1, Depth: 1}, "ruin-stone")
	mesh("ruin-arch-right", "Ruin Arch Right", Vec3{X: 6.4, Y: 1.8, Z: 5.3}, Vec3{X: 0.8, Y: 3.6, Z: 0.8}, Geometry{Kind: "box", Width: 1, Height: 1, Depth: 1}, "ruin-stone")
	mesh("ruin-arch-lintel", "Ruin Arch Lintel", Vec3{X: 4.5, Y: 4.7, Z: 5.3}, Vec3{X: 2.7, Y: 0.65, Z: 0.8}, Geometry{Kind: "box", Width: 1, Height: 1, Depth: 1}, "ruin-stone")
	mesh("arrival-marker", "Arrival Beach", Vec3{X: -1.8, Y: 0.2, Z: 9.2}, Vec3{X: 1, Y: 1, Z: 1}, Geometry{Kind: "sphere", Radius: 0.46, Segments: 20}, "arrival-marker")

	orderedChildren := []ID{"coast-cliff-overlook"}
	for _, id := range root.Children {
		if id != "coast-cliff-overlook" {
			orderedChildren = append(orderedChildren, id)
		}
	}
	root.Children = orderedChildren
	entities[root.ID] = root
	return Document{
		Schema:   SceneDocSchema,
		ID:       "blackglass-coast",
		Name:     "Blackglass Coast",
		Revision: 1,
		RootIDs:  []ID{root.ID},
		Entities: entities,
		Materials: map[ID]Material{
			"sand":           {ID: "sand", Name: "Sunlit Sand", Color: "#d9af70", Roughness: 0.9, Metalness: 0},
			"basalt":         {ID: "basalt", Name: "Wet Basalt", Color: "#1f2428", Roughness: 0.42, Metalness: 0.08, Clearcoat: 0.18},
			"ruin-stone":     {ID: "ruin-stone", Name: "Weathered Ruin Stone", Color: "#716450", Roughness: 0.76, Metalness: 0},
			"beacon-metal":   {ID: "beacon-metal", Name: "Blackglass Alloy", Color: "#131b21", Roughness: 0.24, Metalness: 0.78, Clearcoat: 0.55},
			"beacon-glow":    {ID: "beacon-glow", Name: "Beacon Ember", Color: "#ff9c4e", Roughness: 0.2, Metalness: 0.12, Emissive: 1.25, Selena: &SelenaShader{Material: "BeaconEmber", Source: "material BeaconEmber {\n  param tint : color = rgb(1.0, 0.46, 0.16)\n  surface(geo) -> color { return vec4f(tint.rgb, 1.0) }\n}"}},
			"water-proxy":    {ID: "water-proxy", Name: "Cove Water Proxy", Color: "#167b95", Roughness: 0.12, Metalness: 0.18, Clearcoat: 0.65, Transmission: 0.24},
			"arrival-marker": {ID: "arrival-marker", Name: "Arrival Marker", Color: "#e7bd6b", Roughness: 0.4, Metalness: 0.3},
		},
		Camera:      Camera{Position: Vec3{X: -2, Y: 7.2, Z: 22}, Rotation: Vec3{X: -0.2, Y: -0.18}, FOV: 48, Near: 0.1, Far: 140},
		Environment: Environment{Background: "#77b6cf", AmbientColor: "#c2e2ea", AmbientIntensity: 0.5, Exposure: 1.1, ToneMapping: "aces"},
		World: &WorldContract{
			WaterZones: map[ID]WaterZone{
				"blackglass-cove": {ID: "blackglass-cove", Name: "Blackglass Cove", Center: Vec3{X: -12, Y: -3, Z: -6}, Size: Vec3{X: 34, Y: 10, Z: 24}, SurfaceY: 0, Current: Vec3{X: 0.16, Z: -0.05}, BuoyancyScale: 1.15, LinearDrag: 0.35, RuntimeProfile: "blackglass-coast"},
			},
			Markers: map[ID]WorldMarker{
				"arrival":          {ID: "arrival", Name: "Arrival Beach", Kind: "player-spawn", Entity: "arrival-marker", Position: Vec3{X: -1.8, Y: 0.2, Z: 9.2}},
				"opening-camera":   {ID: "opening-camera", Name: "Opening Overlook", Kind: "camera-start", Entity: "coast-cliff-overlook", Position: Vec3{X: -2, Y: 7.2, Z: 22}},
				"beacon-terrace":   {ID: "beacon-terrace", Name: "Beacon Terrace", Kind: "checkpoint", Entity: "beacon-plinth", Position: Vec3{X: 8, Y: 1.6, Z: -1.5}},
				"beacon-lens":      {ID: "beacon-lens", Name: "Beacon Lens", Kind: "interactable", Entity: "beacon-lens", Position: Vec3{X: 8, Y: 8.1, Z: -4}},
				"cinematic-beacon": {ID: "cinematic-beacon", Name: "Beacon Hero Frame", Kind: "cinematic-target", Entity: "blackglass-beacon", Position: Vec3{X: 8, Y: 4.7, Z: -4}},
			},
		},
		Metadata: map[string]string{
			"art-direction": "sunlit volcanic naturalism",
			"runtime-owner": "gosx Blackglass Coast showcase",
		},
	}
}
