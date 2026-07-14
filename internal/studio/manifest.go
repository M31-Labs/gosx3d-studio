// Package studio defines the handoff contract for the first GoSX 3D Studio
// vertical slice. It deliberately describes capability state without pretending
// that a visible scaffold is an implemented authoring feature.
package studio

const ManifestSchema = "gosx3d.studio.scaffold/v1"

type Manifest struct {
	Schema          string             `json:"schema"`
	Product         string             `json:"product"`
	VisualDirection string             `json:"visualDirection"`
	Surfaces        []Surface          `json:"surfaces"`
	Capabilities    []Capability       `json:"capabilities"`
	Actions         []ActionCapability `json:"actions"`
	NextSlice       []string           `json:"nextSlice"`
}

type Surface struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
}

type Capability struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Evidence string `json:"evidence,omitempty"`
}

func DefaultManifest() Manifest {
	return Manifest{
		Schema:          ManifestSchema,
		Product:         "GoSX 3D Studio",
		VisualDirection: "Industrial Void",
		Surfaces: []Surface{
			{ID: "project", Label: "Project / Assets", Status: "scaffolded"},
			{ID: "hierarchy", Label: "Scene Hierarchy", Status: "scaffolded"},
			{ID: "viewport", Label: "Scene3D Viewport", Status: "mount-seam"},
			{ID: "inspector", Label: "Inspector", Status: "scaffolded"},
			{ID: "timeline", Label: "Timeline", Status: "scaffolded"},
			{ID: "telemetry", Label: "Telemetry", Status: "scaffolded"},
			{ID: "agent-actions", Label: "Agent Actions", Status: "contract-only"},
		},
		Capabilities: []Capability{
			{ID: "server-shell", Status: "available", Evidence: "GET / and GET /api/health"},
			{ID: "agent-manifest", Status: "available", Evidence: "GET /api/studio/manifest"},
			{ID: "scene-document", Status: "available", Evidence: "GET /api/studio/document and internal/studio document tests"},
			{ID: "scene3d-compile", Status: "available", Evidence: "GET /api/studio/scene-ir and shared SceneIR tests"},
			{ID: "scene3d-mount", Status: "planned"},
			{ID: "native-harness", Status: "available", Evidence: "go run ./cmd/studio-smoke"},
			{ID: "desktop-host", Status: "planned"},
		},
		Actions: ActionCatalog(),
		NextSlice: []string{
			"define SceneDoc v1 and stable entity IDs",
			"compile the Chinese Checkers document through shared SceneIR",
			"mount Scene3D in the viewport without editor-only scene truth",
			"connect exact picking and browser-free harness evidence",
			"expose revision-safe read and propose agent actions",
		},
	}
}
