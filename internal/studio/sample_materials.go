package studio

// The board library pairs GoSX's physical material metadata with the portable
// Selena programs from the framework's Chinese Checkers showcase. Procedural
// grain, veining, lacquer, and glaze make the authored finishes read as real
// surfaces; Brushed Steel intentionally stays on the lit Standard PBR path so
// its metallic response remains driven by the Studio's four-light rig.

const carvedWoodSelena = `
material CarvedWood {
    param darkWalnut : color = rgb(0.07, 0.025, 0.012)
    param warmWalnut : color = rgb(0.30, 0.12, 0.05)
    param satinLift  : float = 0.025
    surface(geo) -> color {
        let grainLine = sin(geo.uv.x * 54.0 + sin(geo.uv.y * 10.0) * 1.6)
        let grain     = grainLine * 0.5 + 0.5
        let endGrain  = sin((geo.uv.x + geo.uv.y) * 19.0) * 0.025
        let holeUV    = geo.uv - vec2f(0.5, 0.5)
        let carved    = smoothstep(0.34, 0.04, length(holeUV)) * 0.10
        let body      = mix(darkWalnut.rgb, warmWalnut.rgb, 0.34 + grain * 0.30 + endGrain)
        return vec4f(body * (1.0 - carved) + vec3f(satinLift, satinLift, satinLift), 1.0)
    }
}`

const imperialJadeSelena = `
material ImperialJade {
    param deepJade : color = rgb(0.08, 0.32, 0.20)
    param paleJade : color = rgb(0.42, 0.86, 0.61)
    param rimTint  : color = rgb(0.70, 0.96, 0.80)
    surface(geo) -> color {
        let n         = normalize(geo.normal)
        let thickness = clamp(1.0 - abs(n.y), 0.0, 1.0)
        let internal  = sin((geo.uv.x + geo.uv.y) * 18.0) * 0.035
        let body      = mix(deepJade.rgb, paleJade.rgb, thickness * 0.62 + internal)
        let rim       = pow(1.0 - abs(n.z), 3.0) * 0.22
        return vec4f(mix(body, rimTint.rgb, rim), 0.94)
    }
}`

const midnightLacquerSelena = `
material MidnightLacquer {
    param lacquerBlack : color = rgb(0.002, 0.001, 0.002)
    param cinnabar     : color = rgb(0.065, 0.003, 0.002)
    param goldLeaf    : color = rgb(0.34, 0.11, 0.012)
    surface(geo) -> color {
        let n       = normalize(geo.normal)
        let cloudA  = sin(geo.uv.x * 31.0 + sin(geo.uv.y * 9.0) * 3.0)
        let cloudB  = sin(geo.uv.y * 47.0 + sin(geo.uv.x * 13.0) * 2.0)
        let cloud   = clamp((cloudA + cloudB) * 0.12 + 0.38, 0.0, 1.0)
        let flake   = smoothstep(0.992, 0.999, abs(sin(geo.uv.x * 173.0) * sin(geo.uv.y * 149.0)))
        let body    = mix(lacquerBlack.rgb, cinnabar.rgb, cloud)
        let gilded  = mix(body, goldLeaf.rgb, flake * 0.28)
        let clear   = pow(1.0 - abs(n.y), 4.0) * 0.06
        return vec4f(gilded + vec3f(clear, clear, clear), 1.0)
    }
}`

const moonPorcelainSelena = `
material MoonPorcelain {
    param porcelain : color = rgb(0.82, 0.88, 0.91)
    param cobalt    : color = rgb(0.035, 0.15, 0.42)
    param pearl     : color = rgb(0.72, 0.88, 0.96)
    surface(geo) -> color {
        let n       = normalize(geo.normal)
        let veinA   = abs(sin(geo.uv.x * 29.0 + sin(geo.uv.y * 17.0) * 1.8))
        let veinB   = abs(sin(geo.uv.y * 37.0 + sin(geo.uv.x * 11.0) * 2.1))
        let crackle = smoothstep(0.08, 0.015, veinA * veinB)
        let wash    = sin((geo.uv.x - geo.uv.y) * 8.0) * 0.06 + 0.10
        let body    = mix(porcelain.rgb, cobalt.rgb, crackle * 0.62 + wash)
        let rim     = pow(1.0 - abs(n.z), 3.0) * 0.24
        return vec4f(mix(body, pearl.rgb, rim), 1.0)
    }
}`

func checkerMaterials() map[ID]Material {
	return map[ID]Material{
		"board-material": {
			ID: "board-material", Name: "Carved Wood", Color: "#5f4032",
			Roughness: 0.5, Metalness: 0, Clearcoat: 0.24, Sheen: 0.08,
			Selena: &SelenaShader{Material: "CarvedWood", Source: carvedWoodSelena},
		},
		"board-jade-material": {
			ID: "board-jade-material", Name: "Imperial Jade", Color: "#4fa979",
			Roughness: 0.24, Metalness: 0.02, Clearcoat: 0.7, Transmission: 0.18, Iridescence: 0.08,
			Selena: &SelenaShader{Material: "ImperialJade", Source: imperialJadeSelena},
		},
		"board-steel-material": {
			ID: "board-steel-material", Name: "Brushed Steel", Color: "#68767c",
			Roughness: 0.32, Metalness: 0.92, Clearcoat: 0.38, Sheen: 0.025, Iridescence: 0.025, Anisotropy: 0.72,
		},
		"board-lacquer-material": {
			ID: "board-lacquer-material", Name: "Midnight Lacquer", Color: "#5a1519",
			Roughness: 0.16, Metalness: 0.12, Clearcoat: 0.9, Sheen: 0.14,
			Selena: &SelenaShader{Material: "MidnightLacquer", Source: midnightLacquerSelena},
		},
		"board-porcelain-material": {
			ID: "board-porcelain-material", Name: "Moon Porcelain", Color: "#cbd9df",
			Roughness: 0.28, Metalness: 0.01, Clearcoat: 0.68, Iridescence: 0.18,
			Selena: &SelenaShader{Material: "MoonPorcelain", Source: moonPorcelainSelena},
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
			ID: "player-1-material", Name: "Coral Pieces", Color: "#c8321f",
			// Keep the pieces on the lit Standard PBR path. The old constant-color
			// Selena surface treated display-range coral as linear input; ACES then
			// lifted it to a flat chalk-pink and bypassed the sphere normals. These
			// A warm, blue-restrained coral base and matte dielectric finish keep the
			// Studio key light from bleaching the bright faces toward white-pink,
			// while the sphere normals still describe each piece's round form.
			Roughness: 0.70,
		},
		"player-4-material": {
			ID: "player-4-material", Name: "Cobalt Pieces", Color: "#3d7ba1",
			Roughness: 0.36, Metalness: 0.22, Clearcoat: 0.54, Sheen: 0.1,
			Iridescence: 0.05, Emissive: 0.008,
		},
	}
}
