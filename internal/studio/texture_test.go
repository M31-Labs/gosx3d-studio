package studio

import (
	"strings"
	"testing"
)

var texID = ID("asset-sha256-" + strings.Repeat("a", 64))

func textureDocument(t *testing.T) Document {
	t.Helper()
	document := SampleDocument()
	document.Assets = map[ID]AssetRecord{
		texID: {ID: texID, Kind: "image", Format: "png", MediaType: "image/png", ContentHash: strings.Repeat("a", 64), Bytes: 4, URI: "/api/studio/assets/content/" + string(texID), StorePath: "assets/sha256/" + strings.Repeat("a", 64) + ".png", SourceName: "wood.png"},
	}
	material := document.Materials["board-material"]
	material.Textures = map[string]TextureSlot{
		"color":     {Asset: texID, ColorSpace: "srgb"},
		"roughness": {Asset: texID, ColorSpace: "linear"},
	}
	document.Materials["board-material"] = material
	return document
}

func TestMaterialTextureSlotsValidateAndLowerToEngineMaps(t *testing.T) {
	document := textureDocument(t)
	if err := document.Validate(); err != nil {
		t.Fatalf("texture slots must validate: %v", err)
	}
	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	ir := props.SceneIR()
	found := false
	for _, object := range ir.Objects {
		if object.ID == "board" {
			found = true
			if object.Texture != "/api/studio/assets/content/" + string(texID) + "" {
				t.Fatalf("board texture = %q", object.Texture)
			}
		}
	}
	if !found {
		t.Fatal("board missing")
	}
}

func TestMaterialTextureSlotsRejectBadChannelsAndMissingAssets(t *testing.T) {
	document := textureDocument(t)
	material := document.Materials["board-material"]
	material.Textures = map[string]TextureSlot{"sparkle": {Asset: texID}}
	document.Materials["board-material"] = material
	if err := document.Validate(); err == nil {
		t.Fatal("unknown texture channel must fail validation")
	}
	document = textureDocument(t)
	material = document.Materials["board-material"]
	material.Textures = map[string]TextureSlot{"color": {Asset: ID("asset-sha256-" + strings.Repeat("b", 64))}}
	document.Materials["board-material"] = material
	if err := document.Validate(); err == nil {
		t.Fatal("missing texture asset must fail validation")
	}
}

func TestTextureReferencedAssetsAreDeleteSafeAndGCReachable(t *testing.T) {
	document := textureDocument(t)
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := workspace.Execute(Transaction{ID: "delete-tex", Actor: "test", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpDeleteAsset, AssetID: texID}}}); err == nil {
		t.Fatal("deleting a texture-referenced asset must fail")
	}
	plan, err := PlanAssetGarbage(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range plan.Assets {
		if asset == texID {
			t.Fatal("GC plan must not collect texture-referenced assets")
		}
	}
}
