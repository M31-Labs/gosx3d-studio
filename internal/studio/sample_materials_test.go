package studio

import "testing"

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

	for _, id := range []ID{"board-material", "board-jade-material", "board-steel-material", "board-lacquer-material", "board-porcelain-material"} {
		material, ok := document.Materials[id]
		if !ok {
			t.Fatalf("board finish %q missing", id)
		}
		if material.Selena != nil {
			t.Fatalf("board finish %q uses a custom shader; built-in PBR lighting and shadows must remain active", id)
		}
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
	if got := objects["board"]; got.kind != "cylinder" || got.sheen != 0.08 || got.wireframe == nil || *got.wireframe {
		t.Fatalf("compiled board = %+v, want solid sheen-bearing PBR cylinder", got)
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
			if object.MaterialKind != "standard" || object.Color != "#7d898f" || object.Metalness != 0.84 || object.Anisotropy != 0.58 || object.Wireframe == nil || *object.Wireframe {
				t.Fatalf("compiled Brushed Steel board = %+v", object)
			}
		}
	}
	if !foundSteel {
		t.Fatal("compiled Brushed Steel board missing")
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
