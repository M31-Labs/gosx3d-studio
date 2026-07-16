package studio

// InitializeReport is the spec §13.1 initialize handshake: protocol identity,
// document identity, action-surface counts, and authority modes, so an agent
// can discover the contract before calling anything else.
type InitializeReport struct {
	Protocol       string                   `json:"protocol"`
	SchemaVersion  string                   `json:"schemaVersion"`
	Application    string                   `json:"application"`
	AuthorityModes []string                 `json:"authorityModes"`
	Transports     []string                 `json:"transports"`
	Document       InitializeDocument       `json:"document"`
	Actions        InitializeActionsSummary `json:"actions"`
	Endpoints      map[string]string        `json:"endpoints"`
}

type InitializeDocument struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	Revision    uint64 `json:"revision"`
	Fingerprint string `json:"fingerprint"`
	Entities    int    `json:"entities"`
}

type InitializeActionsSummary struct {
	Mutating int `json:"mutating"`
	Read     int `json:"read"`
}

// BuildInitializeReport assembles the handshake for the current document.
func BuildInitializeReport(document Document) (InitializeReport, error) {
	fingerprint, err := document.Fingerprint()
	if err != nil {
		return InitializeReport{}, err
	}
	mutating, read := 0, 0
	for _, descriptor := range ActionDescriptors() {
		if descriptor.Access == "read" {
			read++
		} else {
			mutating++
		}
	}
	return InitializeReport{
		Protocol:       "gosx.studio3d.actions/v1",
		SchemaVersion:  SceneDocSchema,
		Application:    "gosx3d-studio",
		AuthorityModes: []string{"read", "propose", "direct"},
		Transports:     []string{"http+bearer"},
		Document: InitializeDocument{
			ID: string(document.ID), Name: document.Name, Schema: document.Schema,
			Revision: document.Revision, Fingerprint: fingerprint, Entities: len(document.Entities),
		},
		Actions: InitializeActionsSummary{Mutating: mutating, Read: read},
		Endpoints: map[string]string{
			"actionsList":      "/api/studio/actions",
			"actionsPreview":   "/api/studio/actions/preview",
			"transactionsCall": "/api/studio/transactions/call",
			"undo":             "/api/studio/undo",
			"redo":             "/api/studio/redo",
			"selection":        "/api/studio/selection",
			"certification":    "/api/studio/certification/evidence",
		},
	}, nil
}
