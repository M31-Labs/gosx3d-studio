package studio

type SourceLocation struct {
	Entity   ID     `json:"entity"`
	Kind     string `json:"kind"`
	RecordID string `json:"recordId"`
}
type CompileArtifact struct {
	DocumentFingerprint string                `json:"documentFingerprint"`
	SceneIR             SceneIR               `json:"sceneIR"`
	SourceMap           map[ID]SourceLocation `json:"sourceMap"`
}

func CompileWithSourceMap(document Document) (CompileArtifact, error) {
	ir, err := CompileIR(document)
	if err != nil {
		return CompileArtifact{}, err
	}
	fingerprint, err := document.Fingerprint()
	if err != nil {
		return CompileArtifact{}, err
	}
	sourceMap := make(map[ID]SourceLocation, len(document.Entities))
	for id, entity := range document.Entities {
		kind := "group"
		if entity.Mesh != nil {
			kind = "object"
		} else if entity.Model != nil {
			kind = "model"
		} else if entity.Light != nil {
			kind = "light"
		}
		sourceMap[id] = SourceLocation{Entity: id, Kind: kind, RecordID: string(id)}
		if entity.Prefab != nil {
			definition, err := resolvePrefabDefinition(document.Prefabs, entity.Prefab.Prefab)
			if err != nil {
				return CompileArtifact{}, err
			}
			for localID, local := range definition.Entities {
				runtimeID := ID(string(id) + "--" + string(localID))
				localKind := "group"
				if local.Mesh != nil {
					localKind = "object"
				} else if local.Model != nil {
					localKind = "model"
				} else if local.Light != nil {
					localKind = "light"
				}
				sourceMap[runtimeID] = SourceLocation{Entity: id, Kind: localKind, RecordID: string(definition.ID) + "/" + string(localID)}
			}
		}
	}
	return CompileArtifact{DocumentFingerprint: fingerprint, SceneIR: ir, SourceMap: sourceMap}, nil
}
