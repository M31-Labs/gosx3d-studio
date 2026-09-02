package studio

// The board library stays on GoSX's Standard PBR path so every finish keeps
// built-in lighting, IBL, and shadows with matching WebGPU/WebGL semantics.
// The expanded physical fields exercise the current GoSX material transport
// without trading those semantics for a full custom-shader replacement.

func checkerMaterials() map[ID]Material {
	return map[ID]Material{
		"board-material": {
			ID: "board-material", Name: "Carved Wood", Color: "#5f4032",
			Roughness: 0.5, Metalness: 0, Clearcoat: 0.24, Sheen: 0.08,
		},
		"board-jade-material": {
			ID: "board-jade-material", Name: "Imperial Jade", Color: "#4fa979",
			Roughness: 0.24, Metalness: 0.02, Clearcoat: 0.7, Transmission: 0.18, Iridescence: 0.08,
		},
		"board-steel-material": {
			ID: "board-steel-material", Name: "Brushed Steel", Color: "#7d898f",
			Roughness: 0.42, Metalness: 0.84, Clearcoat: 0.28, Anisotropy: 0.58,
		},
		"board-lacquer-material": {
			ID: "board-lacquer-material", Name: "Midnight Lacquer", Color: "#5a1519",
			Roughness: 0.16, Metalness: 0.12, Clearcoat: 0.9, Sheen: 0.14,
		},
		"board-porcelain-material": {
			ID: "board-porcelain-material", Name: "Moon Porcelain", Color: "#cbd9df",
			Roughness: 0.28, Metalness: 0.01, Clearcoat: 0.68, Iridescence: 0.18,
		},
		"pedestal-material": {
			ID: "pedestal-material", Name: "Blackened Steel", Color: "#222a30",
			Roughness: 0.26, Metalness: 0.78, Clearcoat: 0.35, Anisotropy: 0.22,
		},
		"board-rim-material": {
			ID: "board-rim-material", Name: "Machined Rim", Color: "#4d5a60",
			Roughness: 0.38, Metalness: 0.82, Clearcoat: 0.28, Anisotropy: 0.22,
		},
		"socket-material": {
			ID: "socket-material", Name: "Countersunk Socket", Color: "#070a0c",
			Roughness: 0.68, Metalness: 0.12, Clearcoat: 0.28, Iridescence: 0.04,
		},
		"player-1-material": {
			ID: "player-1-material", Name: "Coral Pieces", Color: "#e66b62",
			Roughness: 0.2, Metalness: 0.08, Clearcoat: 0.68, Sheen: 0.18,
			Transmission: 0.04, Iridescence: 0.08, Emissive: 0.025,
			Selena: &SelenaShader{Material: "CoralPiece", Source: `
material CoralPiece {
  param tint : color = rgb(0.902, 0.420, 0.384)
  surface(geo) -> color { return vec4f(tint.rgb, 1.0) }
}`},
		},
		"player-4-material": {
			ID: "player-4-material", Name: "Cobalt Pieces", Color: "#3d7ba1",
			Roughness: 0.36, Metalness: 0.22, Clearcoat: 0.54, Sheen: 0.1,
			Iridescence: 0.05, Emissive: 0.008,
		},
	}
}
