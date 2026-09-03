package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx/scene"
	"m31labs.dev/gosx/scene/preview"
)

func TestShowcaseBoardUsesLayeredPhysicalMaterials(t *testing.T) {
	document := SampleDocument()
	board := document.Entities["board"]
	chassis := document.Entities["board-pedestal"]
	outerRim := document.Entities["board-outer-rim"]
	innerFillet := document.Entities["board-inner-fillet"]
	if board.Mesh == nil || chassis.Mesh == nil {
		t.Fatal("board face and chassis must both be meshes")
	}
	if outerRim.Mesh == nil || innerFillet.Mesh == nil || outerRim.Mesh.Geometry.Kind != "torus" || innerFillet.Mesh.Geometry.Kind != "torus" {
		t.Fatal("board edge must use typed outer-rim and inner-fillet torus layers")
	}
	if board.Mesh.Geometry.RadiusTop >= chassis.Mesh.Geometry.RadiusTop {
		t.Fatalf("board face radius %.2f must reveal chassis radius %.2f", board.Mesh.Geometry.RadiusTop, chassis.Mesh.Geometry.RadiusTop)
	}
	if board.Transform.Position.Y <= chassis.Transform.Position.Y {
		t.Fatalf("board face y %.2f must sit above chassis y %.2f", board.Transform.Position.Y, chassis.Transform.Position.Y)
	}

	socket := document.Entities["socket-000"]
	if socket.Mesh == nil || socket.Mesh.Geometry.Kind != "sphere" {
		t.Fatalf("socket geometry = %+v, want a rounded sphere well", socket.Mesh)
	}
	if socket.Transform.Scale != (Vec3{X: 1, Y: 0.28, Z: 1}) {
		t.Fatalf("socket scale = %+v, want a flattened countersunk profile", socket.Transform.Scale)
	}
	boardTop := board.Transform.Position.Y + board.Mesh.Geometry.Height/2
	socketTop := socket.Transform.Position.Y + socket.Mesh.Geometry.Radius*socket.Transform.Scale.Y
	if socketTop-boardTop > 0.015 {
		t.Fatalf("socket rises %.3f above the face, want at most a thin highlight crescent", socketTop-boardTop)
	}

	for _, id := range []ID{"board-selena-detail-material", "board-jade-material", "board-lacquer-material", "board-porcelain-material"} {
		material, ok := document.Materials[id]
		if !ok {
			t.Fatalf("board finish %q missing", id)
		}
		if material.Selena == nil {
			t.Fatalf("board finish %q is missing its portable Selena surface program", id)
		}
	}
	wood := document.Materials["board-material"]
	if wood.Color != "#76513a" || wood.Roughness != 0.38 || wood.Metalness != 0 ||
		wood.Clearcoat != 0.42 || wood.Sheen != 0.14 || wood.Anisotropy != 0.32 {
		t.Fatalf("Carved Wood physical fallback = %+v", wood)
	}
	if wood.Selena != nil {
		t.Fatal("default Carved Wood must retain portable lit Standard PBR shading")
	}
	grain := document.Materials["board-selena-detail-material"]
	for _, authoredLift := range []string{
		"rgb(0.16, 0.065, 0.025)",
		"rgb(0.66, 0.32, 0.14)",
		"float = 0.045",
		"1.0 - smoothstep(0.04, 0.34, length(holeUV))",
	} {
		if !strings.Contains(grain.Selena.Source, authoredLift) {
			t.Fatalf("Carved Grain Inlay Selena source is missing calibrated display lift %q", authoredLift)
		}
	}
	if material := document.Materials["board-steel-material"]; material.Selena != nil {
		t.Fatal("Brushed Steel must retain Standard PBR lighting under the Studio rig")
	}
	coral := document.Materials["player-1-material"]
	if coral.Selena != nil {
		t.Fatal("Coral Pieces must use lit Standard PBR instead of a flat custom shader")
	}
	if coral.Color != "#c8321f" || coral.Roughness != 0.32 || coral.Metalness != 0 ||
		coral.Clearcoat != 0.65 || coral.Sheen != 0.08 || coral.Transmission != 0 ||
		coral.Iridescence != 0 || coral.Emissive != 0 {
		t.Fatalf("Coral Pieces physical finish = %+v", coral)
	}
	playerOnePieces := 0
	for _, entity := range document.Entities {
		if entity.Mesh != nil && entity.Mesh.Material == coral.ID {
			playerOnePieces++
		}
	}
	if playerOnePieces != 10 {
		t.Fatalf("Coral Pieces references = %d, want 10", playerOnePieces)
	}

	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	objects := map[string]struct {
		kind                                         string
		scaleY, tube, sheen, iridescence, anisotropy float64
		wireframe                                    *bool
	}{}
	for _, object := range props.SceneIR().Objects {
		objects[object.ID] = struct {
			kind                                         string
			scaleY, tube, sheen, iridescence, anisotropy float64
			wireframe                                    *bool
		}{object.Kind, object.ScaleY, object.Tube, object.Sheen, object.Iridescence, object.Anisotropy, object.Wireframe}
	}
	if got := objects["board"]; got.kind != "cylinder" || got.sheen != 0.14 || got.anisotropy != 0.32 || got.wireframe == nil || *got.wireframe {
		t.Fatalf("compiled board = %+v, want solid lit anisotropic wood cylinder", got)
	}
	if got := objects["board-pedestal"]; got.anisotropy != 0.22 {
		t.Fatalf("compiled chassis anisotropy = %.2f, want 0.22", got.anisotropy)
	}
	if got := objects["socket-000"]; got.kind != "sphere" || got.scaleY != 0.28 || got.iridescence != 0.04 {
		t.Fatalf("compiled socket = %+v, want flattened physical sphere", got)
	}
	if got := objects["board-outer-rim"]; got.kind != "torus" || got.tube != 0.075 || got.anisotropy != 0.22 || got.wireframe == nil || *got.wireframe {
		t.Fatalf("compiled outer rim = %+v, want a solid machined torus", got)
	}
	if got := objects["board-inner-fillet"]; got.kind != "torus" || got.tube != 0.018 || got.wireframe == nil || *got.wireframe {
		t.Fatalf("compiled inner fillet = %+v, want a solid shadow torus", got)
	}
	foundCoral := false
	for _, object := range props.SceneIR().Objects {
		if object.ID != "piece-player-1-01" {
			continue
		}
		foundCoral = true
		if object.Kind != "sphere" || object.MaterialKind != "standard" || object.Color != "#c8321f" ||
			object.Roughness != 0.32 || object.Metalness != 0 || object.Clearcoat != 0.65 ||
			object.Sheen != 0.08 || object.Transmission != 0 || object.Iridescence != 0 ||
			object.Emissive != nil || object.Wireframe == nil || *object.Wireframe {
			t.Fatalf("compiled Coral Piece = %+v, want a solid lit coral Standard PBR sphere", object)
		}
	}
	if !foundCoral {
		t.Fatal("compiled Coral Piece missing")
	}

	// This is the material used by the recorded WebMCP handoff. Exercise the
	// exact assignment so the demo cannot silently fall back to the old piece
	// finish or lose the physical fields at the renderer boundary.
	board = document.Entities["board"]
	board.Mesh.Material = "board-steel-material"
	document.Entities["board"] = board
	props, err = Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	foundSteel := false
	for _, object := range props.SceneIR().Objects {
		if object.ID == "board" {
			foundSteel = true
			if object.MaterialKind != "standard" || object.Color != "#68767c" || object.Metalness != 0.92 || object.Anisotropy != 0.72 || object.Wireframe == nil || *object.Wireframe {
				t.Fatalf("compiled Brushed Steel board = %+v", object)
			}
		}
	}
	if !foundSteel {
		t.Fatal("compiled Brushed Steel board missing")
	}
}

func TestShowcaseCoralRendersSaturatedUnderStudioLighting(t *testing.T) {
	document := SampleDocument()
	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	result, err := preview.Render(props, preview.Options{
		Width: 720, Height: 480, Background: document.Environment.Background,
		DisablePostFX: true, MaxSegments: 24,
	})
	if err != nil {
		t.Fatal(err)
	}
	// The previous constant-output shader yielded zero pixels at this
	// saturation threshold and made the coral army read as white-pink. Keep
	// enough deeply red-orange pixels in the deterministic rendered frame that
	// future material or lighting edits cannot silently reintroduce that washout.
	saturatedCoralPixels := 0
	for y := result.Image.Bounds().Min.Y; y < result.Image.Bounds().Max.Y; y++ {
		for x := result.Image.Bounds().Min.X; x < result.Image.Bounds().Max.X; x++ {
			pixel := result.Image.RGBAAt(x, y)
			if int(pixel.R)-int(pixel.G) >= 70 && int(pixel.R)-int(pixel.B) >= 80 &&
				pixel.R >= 120 && pixel.G >= 30 {
				saturatedCoralPixels++
			}
		}
	}
	if saturatedCoralPixels < 600 {
		t.Fatalf("saturated rendered coral pixels = %d, want at least 600", saturatedCoralPixels)
	}
}

func TestAdvancedPhysicalMaterialRangesAreValidated(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Material)
	}{
		{"sheen above one", func(material *Material) { material.Sheen = 1.01 }},
		{"iridescence below zero", func(material *Material) { material.Iridescence = -0.01 }},
		{"anisotropy above one", func(material *Material) { material.Anisotropy = 1.01 }},
		{"anisotropy below negative one", func(material *Material) { material.Anisotropy = -1.01 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := SampleDocument()
			material := document.Materials["board-material"]
			test.mutate(&material)
			document.Materials[material.ID] = material
			if err := document.Validate(); err == nil {
				t.Fatal("out-of-range physical material value must fail validation")
			}
		})
	}
}

func TestShowcaseBoardFinishChangeIsLiveSceneCommandDiffable(t *testing.T) {
	document := SampleDocument()
	before, err := CompileViewport(document)
	if err != nil {
		t.Fatal(err)
	}
	board := document.Entities["board"]
	board.Mesh.Material = "board-steel-material"
	document.Entities[board.ID] = board
	after, err := CompileViewport(document)
	if err != nil {
		t.Fatal(err)
	}
	diff := scene.DiffScene(before.SceneIR(), after.SceneIR(), scene.DiffOptions{})
	if len(diff.RemountFields) != 0 {
		t.Fatalf("board finish change forces viewport remount through %v", diff.RemountFields)
	}
	if len(diff.Commands) == 0 {
		t.Fatal("board finish change produced no live Scene3D commands")
	}
}
