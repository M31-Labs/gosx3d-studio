package studio

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxAssetBytes int64 = 512 << 20

type AssetImportRequest struct {
	Path             string          `json:"path"`
	Actor            string          `json:"actor"`
	Mode             TransactionMode `json:"mode"`
	ExpectedRevision uint64          `json:"expectedRevision"`
}
type AssetReimportRequest struct {
	AssetID          ID              `json:"assetId"`
	Path             string          `json:"path"`
	Actor            string          `json:"actor"`
	Mode             TransactionMode `json:"mode"`
	ExpectedRevision uint64          `json:"expectedRevision"`
}
type AssetDependencyReport struct {
	Schema   string                 `json:"schema"`
	Revision uint64                 `json:"revision"`
	Assets   []AssetDependencyEntry `json:"assets"`
}
type AssetDependencyEntry struct {
	Asset        ID   `json:"asset"`
	Entities     []ID `json:"entities"`
	Prefabs      []ID `json:"prefabs"`
	Instances    []ID `json:"instances"`
	UsedByAssets []ID `json:"usedByAssets,omitempty"`
	DirectCount  int  `json:"directCount"`
}
type AssetAudit struct {
	Schema   string            `json:"schema"`
	Revision uint64            `json:"revision"`
	Assets   []AssetAuditEntry `json:"assets"`
	Valid    bool              `json:"valid"`
}
type AssetAuditEntry struct {
	ID           ID     `json:"id"`
	Status       string `json:"status"`
	ExpectedHash string `json:"expectedHash"`
	ActualHash   string `json:"actualHash,omitempty"`
	Path         string `json:"path"`
	Bytes        int64  `json:"bytes,omitempty"`
}

func validateAssetRecord(asset AssetRecord) error {
	if asset.ID == "" || asset.Kind == "" || asset.Format == "" || asset.MediaType == "" || asset.Bytes < 0 || asset.SourceName == "" {
		return fmt.Errorf("id, kind, format, mediaType, non-negative bytes, and sourceName are required")
	}
	if len(asset.ContentHash) != 64 {
		return fmt.Errorf("contentHash must be sha256 hex")
	}
	if _, err := hex.DecodeString(asset.ContentHash); err != nil {
		return fmt.Errorf("contentHash: %w", err)
	}
	expectedID := ID("asset-sha256-" + asset.ContentHash)
	if asset.ID != expectedID {
		return fmt.Errorf("id %q does not match content hash", asset.ID)
	}
	expectedURI := "/api/studio/assets/content/" + string(asset.ID)
	if asset.URI != expectedURI {
		return fmt.Errorf("uri %q does not match asset id", asset.URI)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(asset.StorePath)))
	if strings.HasPrefix(clean, "../") || filepath.IsAbs(asset.StorePath) || !strings.HasPrefix(clean, "assets/sha256/") {
		return fmt.Errorf("storePath %q is not a safe project asset path", asset.StorePath)
	}
	if !strings.Contains(filepath.Base(clean), asset.ContentHash) {
		return fmt.Errorf("storePath does not contain content hash")
	}
	seen := map[ID]bool{}
	for _, dependency := range asset.Dependencies {
		if dependency == "" || seen[dependency] {
			return fmt.Errorf("dependencies must contain unique non-empty ids")
		}
		seen[dependency] = true
	}
	return nil
}

func inspectAsset(path string, data []byte) (AssetRecord, error) {
	name := filepath.Base(path)
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	kind, format, media := "binary", extension, "application/octet-stream"
	metadata := map[string]string{}
	switch {
	case len(data) >= 12 && string(data[:4]) == "glTF":
		format, kind, media = "glb", "model", "model/gltf-binary"
		version := binary.LittleEndian.Uint32(data[4:8])
		declared := binary.LittleEndian.Uint32(data[8:12])
		if version != 2 || int(declared) != len(data) {
			return AssetRecord{}, fmt.Errorf("invalid GLB header version=%d declared=%d actual=%d", version, declared, len(data))
		}
		metadata["gltfVersion"] = "2"
		inspection, err := InspectGLTF(data, "glb")
		if err != nil {
			return AssetRecord{}, err
		}
		applyGLTFMetadata(metadata, inspection)
	case extension == "gltf":
		var root map[string]any
		if err := json.Unmarshal(data, &root); err != nil {
			return AssetRecord{}, fmt.Errorf("invalid glTF JSON: %w", err)
		}
		asset, _ := root["asset"].(map[string]any)
		version, _ := asset["version"].(string)
		if version == "" {
			return AssetRecord{}, fmt.Errorf("glTF asset.version is required")
		}
		format, kind, media = "gltf", "model", "model/gltf+json"
		metadata["gltfVersion"] = version
		inspection, err := InspectGLTF(data, "gltf")
		if err != nil {
			return AssetRecord{}, err
		}
		applyGLTFMetadata(metadata, inspection)
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		format, kind, media = "png", "image", "image/png"
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		format, kind, media = "jpg", "image", "image/jpeg"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		format, kind, media = "webp", "image", "image/webp"
	case len(data) >= 12 && string(data[4:8]) == "ftyp" && (string(data[8:12]) == "avif" || string(data[8:12]) == "avis"):
		format, kind, media = "avif", "image", "image/avif"
	case len(data) >= 12 && string(data[:12]) == "\xabKTX 20\xbb\r\n\x1a\n":
		format, kind, media = "ktx2", "image", "image/ktx2"
	case strings.HasPrefix(string(data), "#?RADIANCE") || strings.HasPrefix(string(data), "#?RGBE"):
		format, kind, media = "hdr", "image", "image/vnd.radiance"
	case len(data) >= 4 && data[0] == 0x76 && data[1] == 0x2f && data[2] == 0x31 && data[3] == 0x01:
		format, kind, media = "exr", "image", "image/x-exr"
	case len(data) >= 4 && string(data[:4]) == "RIFF" && len(data) >= 12 && string(data[8:12]) == "WAVE":
		format, kind, media = "wav", "audio", "audio/wav"
	case len(data) >= 4 && string(data[:4]) == "OggS":
		format, kind, media = "ogg", "audio", "audio/ogg"
	case len(data) >= 12 && string(data[4:8]) == "ftyp":
		format, kind, media = "mp4", "video", "video/mp4"
	case extension == "obj":
		format, kind, media = "obj", "model", "model/obj"
		vertices, faces := 0, 0
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "v ") {
				vertices++
			}
			if strings.HasPrefix(line, "f ") {
				faces++
			}
		}
		if vertices == 0 || faces == 0 {
			return AssetRecord{}, fmt.Errorf("OBJ requires vertices and faces")
		}
		metadata["vertices"] = fmt.Sprint(vertices)
		metadata["faces"] = fmt.Sprint(faces)
	case extension == "bin":
		format, kind, media = "bin", "buffer", "application/octet-stream"
	default:
		return AssetRecord{}, fmt.Errorf("unsupported or unrecognized asset %q", name)
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	id := ID("asset-sha256-" + hash)
	storePath := filepath.ToSlash(filepath.Join("assets", "sha256", hash+"."+format))
	return AssetRecord{ID: id, Kind: kind, Format: format, MediaType: media, ContentHash: hash, Bytes: int64(len(data)), URI: "/api/studio/assets/content/" + string(id), StorePath: storePath, SourceName: name, Metadata: metadata}, nil
}

type preparedAssetPayload struct {
	Asset AssetRecord
	Data  []byte
}

func inspectBinaryBuffer(path string, data []byte) AssetRecord {
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	id := ID("asset-sha256-" + hash)
	return AssetRecord{ID: id, Kind: "buffer", Format: "bin", MediaType: "application/octet-stream", ContentHash: hash, Bytes: int64(len(data)), URI: "/api/studio/assets/content/" + string(id), StorePath: filepath.ToSlash(filepath.Join("assets", "sha256", hash+".bin")), SourceName: filepath.Base(path), Metadata: map[string]string{"sourceExtension": strings.ToLower(filepath.Ext(path))}}
}

func prepareAssetImport(path string, data []byte) (preparedAssetPayload, []preparedAssetPayload, error) {
	if strings.ToLower(filepath.Ext(path)) != ".gltf" {
		asset, err := inspectAsset(path, data)
		return preparedAssetPayload{Asset: asset, Data: data}, nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return preparedAssetPayload{}, nil, fmt.Errorf("invalid glTF JSON: %w", err)
	}
	type uriField struct {
		value map[string]any
		role  string
	}
	uriFields := []uriField{}
	for _, key := range []string{"buffers", "images"} {
		items, _ := root[key].([]any)
		for _, raw := range items {
			if item, ok := raw.(map[string]any); ok {
				if _, ok := item["uri"].(string); ok {
					uriFields = append(uriFields, uriField{value: item, role: key})
				}
			}
		}
	}
	baseReal, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return preparedAssetPayload{}, nil, err
	}
	byURI := map[string]preparedAssetPayload{}
	for _, field := range uriFields {
		rawURI := strings.TrimSpace(field.value["uri"].(string))
		if rawURI == "" || strings.HasPrefix(rawURI, "data:") {
			continue
		}
		parsed, err := url.Parse(rawURI)
		if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return preparedAssetPayload{}, nil, fmt.Errorf("glTF dependency URI %q must be a local relative path", rawURI)
		}
		decoded, err := url.PathUnescape(parsed.Path)
		if err != nil || filepath.IsAbs(decoded) {
			return preparedAssetPayload{}, nil, fmt.Errorf("glTF dependency URI %q is invalid", rawURI)
		}
		if strings.Contains(decoded, "\\") {
			return preparedAssetPayload{}, nil, fmt.Errorf("glTF dependency URI %q must use portable forward slashes", rawURI)
		}
		candidate := filepath.Join(filepath.Dir(path), filepath.FromSlash(decoded))
		real, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			return preparedAssetPayload{}, nil, fmt.Errorf("glTF dependency %q: %w", rawURI, err)
		}
		rel, err := filepath.Rel(baseReal, real)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return preparedAssetPayload{}, nil, fmt.Errorf("glTF dependency %q escapes source directory", rawURI)
		}
		if existing, ok := byURI[rawURI]; ok {
			field.value["uri"] = existing.Asset.URI
			continue
		}
		info, err := os.Stat(real)
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxAssetBytes {
			return preparedAssetPayload{}, nil, fmt.Errorf("glTF dependency %q must be a regular file no larger than %d bytes", rawURI, maxAssetBytes)
		}
		payload, err := os.ReadFile(real)
		if err != nil {
			return preparedAssetPayload{}, nil, err
		}
		asset, err := inspectAsset(real, payload)
		if err != nil && field.role == "buffers" {
			asset = inspectBinaryBuffer(real, payload)
			err = nil
		}
		if err != nil {
			return preparedAssetPayload{}, nil, fmt.Errorf("glTF dependency %q: %w", rawURI, err)
		}
		prepared := preparedAssetPayload{Asset: asset, Data: payload}
		byURI[rawURI] = prepared
		field.value["uri"] = asset.URI
	}
	rewritten, err := json.Marshal(root)
	if err != nil {
		return preparedAssetPayload{}, nil, err
	}
	asset, err := inspectAsset(path, rewritten)
	if err != nil {
		return preparedAssetPayload{}, nil, err
	}
	sourceSum := sha256.Sum256(data)
	asset.Metadata["sourceContentHash"] = hex.EncodeToString(sourceSum[:])
	dependencies := make([]preparedAssetPayload, 0, len(byURI))
	ids := make([]string, 0, len(byURI))
	for _, item := range byURI {
		dependencies = append(dependencies, item)
		ids = append(ids, string(item.Asset.ID))
	}
	sort.Slice(dependencies, func(i, j int) bool { return dependencies[i].Asset.ID < dependencies[j].Asset.ID })
	sort.Strings(ids)
	asset.Metadata["gltfDependencies"] = strings.Join(ids, ",")
	asset.Dependencies = make([]ID, len(ids))
	for i, id := range ids {
		asset.Dependencies[i] = ID(id)
	}
	return preparedAssetPayload{Asset: asset, Data: rewritten}, dependencies, nil
}

func applyGLTFMetadata(metadata map[string]string, inspection GLTFInspection) {
	metadata["gltfMatrixSchema"] = GLTFMatrixSchema
	metadata["gltfInspectionSchema"] = inspection.Schema
	metadata["gltfCompatibilityFingerprint"] = inspection.Fingerprint
	metadata["gltfExtensionsUsed"] = strings.Join(inspection.ExtensionsUsed, ",")
	metadata["gltfExtensionsRequired"] = strings.Join(inspection.ExtensionsRequired, ",")
	for _, target := range inspection.Targets {
		metadata["gltfTarget."+target.Target] = target.Status
	}
}

func (w *Workspace) ImportAsset(request AssetImportRequest) (Receipt, Document, AssetRecord, error) {
	info, err := os.Stat(request.Path)
	if err != nil {
		return Receipt{}, Document{}, AssetRecord{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxAssetBytes {
		return Receipt{}, Document{}, AssetRecord{}, fmt.Errorf("asset must be a regular file no larger than %d bytes", maxAssetBytes)
	}
	data, err := os.ReadFile(request.Path)
	if err != nil {
		return Receipt{}, Document{}, AssetRecord{}, err
	}
	prepared, dependencies, err := prepareAssetImport(request.Path, data)
	if err != nil {
		return Receipt{}, Document{}, AssetRecord{}, err
	}
	asset := prepared.Asset
	operations := make([]Operation, 0, len(dependencies)+1)
	for i := range dependencies {
		dependency := dependencies[i].Asset
		operations = append(operations, Operation{Kind: OpRegisterAsset, Asset: &dependency})
	}
	operations = append(operations, Operation{Kind: OpRegisterAsset, Asset: &asset})
	transaction := Transaction{ID: "import-asset:" + asset.ContentHash[:16], Actor: request.Actor, Mode: request.Mode, ExpectedRevision: request.ExpectedRevision, Operations: operations}
	if request.Mode == ModeDirect {
		transaction.Mode = ModePropose
		if _, _, err := w.Execute(transaction); err != nil {
			return Receipt{}, Document{}, AssetRecord{}, err
		}
		w.mu.RLock()
		projectDir := w.projectDir
		w.mu.RUnlock()
		if projectDir == "" {
			return Receipt{}, Document{}, AssetRecord{}, fmt.Errorf("direct asset import requires a persisted project")
		}
		for _, dependency := range dependencies {
			if err := storeAssetPayload(projectDir, dependency.Asset, dependency.Data); err != nil {
				return Receipt{}, Document{}, AssetRecord{}, err
			}
		}
		if err := storeAssetPayload(projectDir, asset, prepared.Data); err != nil {
			return Receipt{}, Document{}, AssetRecord{}, err
		}
		transaction.Mode = ModeDirect
	}
	receipt, document, err := w.Execute(transaction)
	return receipt, document, asset, err
}

func (w *Workspace) ReimportAsset(request AssetReimportRequest) (Receipt, Document, AssetRecord, error) {
	if request.AssetID == "" {
		return Receipt{}, Document{}, AssetRecord{}, fmt.Errorf("reimport requires assetId")
	}
	info, err := os.Stat(request.Path)
	if err != nil {
		return Receipt{}, Document{}, AssetRecord{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxAssetBytes {
		return Receipt{}, Document{}, AssetRecord{}, fmt.Errorf("asset must be a regular file no larger than %d bytes", maxAssetBytes)
	}
	data, err := os.ReadFile(request.Path)
	if err != nil {
		return Receipt{}, Document{}, AssetRecord{}, err
	}
	prepared, dependencies, err := prepareAssetImport(request.Path, data)
	if err != nil {
		return Receipt{}, Document{}, AssetRecord{}, err
	}
	asset := prepared.Asset
	operations := make([]Operation, 0, len(dependencies)+1)
	for i := range dependencies {
		dependency := dependencies[i].Asset
		operations = append(operations, Operation{Kind: OpRegisterAsset, Asset: &dependency})
	}
	operations = append(operations, Operation{Kind: OpReimportAsset, AssetID: request.AssetID, Asset: &asset})
	transaction := Transaction{
		ID: fmt.Sprintf("reimport-asset:%s:%s", request.AssetID, asset.ContentHash[:16]), Actor: request.Actor,
		Mode: request.Mode, ExpectedRevision: request.ExpectedRevision,
		Operations: operations,
	}
	if request.Mode == ModeDirect {
		transaction.Mode = ModePropose
		if _, _, err := w.Execute(transaction); err != nil {
			return Receipt{}, Document{}, AssetRecord{}, err
		}
		w.mu.RLock()
		projectDir := w.projectDir
		w.mu.RUnlock()
		if projectDir == "" {
			return Receipt{}, Document{}, AssetRecord{}, fmt.Errorf("direct asset reimport requires a persisted project")
		}
		for _, dependency := range dependencies {
			if err := storeAssetPayload(projectDir, dependency.Asset, dependency.Data); err != nil {
				return Receipt{}, Document{}, AssetRecord{}, err
			}
		}
		if err := storeAssetPayload(projectDir, asset, prepared.Data); err != nil {
			return Receipt{}, Document{}, AssetRecord{}, err
		}
		transaction.Mode = ModeDirect
	}
	receipt, document, err := w.Execute(transaction)
	return receipt, document, asset, err
}

func storeAssetPayload(projectDir string, asset AssetRecord, data []byte) error {
	path := filepath.Join(projectDir, filepath.FromSlash(asset.StorePath))
	if existing, err := os.ReadFile(path); err == nil {
		sum := sha256.Sum256(existing)
		if hex.EncodeToString(sum[:]) == asset.ContentHash {
			return nil
		}
		return fmt.Errorf("content-addressed asset path contains different bytes")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".asset-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func (w *Workspace) AssetContentPath(id ID) (string, AssetRecord, error) {
	w.mu.RLock()
	asset, ok := w.doc.Assets[id]
	projectDir := w.projectDir
	w.mu.RUnlock()
	if !ok {
		return "", AssetRecord{}, fmt.Errorf("asset %q does not exist", id)
	}
	if projectDir == "" {
		return "", AssetRecord{}, fmt.Errorf("workspace has no project directory")
	}
	path := filepath.Join(projectDir, filepath.FromSlash(asset.StorePath))
	return path, asset, nil
}

func (w *Workspace) AuditAssets() AssetAudit {
	w.mu.RLock()
	document, _ := w.doc.Clone()
	projectDir := w.projectDir
	w.mu.RUnlock()
	report := AssetAudit{Schema: "gosx3d.studio.asset-audit/v1", Revision: document.Revision, Valid: true, Assets: []AssetAuditEntry{}}
	ids := make([]string, 0, len(document.Assets))
	for id := range document.Assets {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, value := range ids {
		asset := document.Assets[ID(value)]
		entry := AssetAuditEntry{ID: asset.ID, ExpectedHash: asset.ContentHash, Path: asset.StorePath, Status: "missing"}
		data, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(asset.StorePath)))
		if err == nil {
			sum := sha256.Sum256(data)
			entry.ActualHash = hex.EncodeToString(sum[:])
			entry.Bytes = int64(len(data))
			if entry.ActualHash == asset.ContentHash && entry.Bytes == asset.Bytes {
				entry.Status = "ok"
			} else {
				entry.Status = "mismatch"
			}
		}
		if entry.Status != "ok" {
			report.Valid = false
		}
		report.Assets = append(report.Assets, entry)
	}
	return report
}

func (w *Workspace) AssetDependencies() AssetDependencyReport {
	w.mu.RLock()
	document, _ := w.doc.Clone()
	w.mu.RUnlock()
	report := AssetDependencyReport{Schema: "gosx3d.studio.asset-dependencies/v1", Revision: document.Revision, Assets: []AssetDependencyEntry{}}
	assetIDs := make([]string, 0, len(document.Assets))
	for id := range document.Assets {
		assetIDs = append(assetIDs, string(id))
	}
	sort.Strings(assetIDs)
	for _, value := range assetIDs {
		id := ID(value)
		entry := AssetDependencyEntry{Asset: id, Entities: []ID{}, Prefabs: []ID{}, Instances: []ID{}}
		prefabUses := map[ID]bool{}
		for entityID, entity := range document.Entities {
			if entity.Model != nil && entity.Model.Asset == id {
				entry.Entities = append(entry.Entities, entityID)
			}
		}
		for assetID, asset := range document.Assets {
			for _, dependency := range asset.Dependencies {
				if dependency == id {
					entry.UsedByAssets = append(entry.UsedByAssets, assetID)
					break
				}
			}
		}
		for prefabID, prefab := range document.Prefabs {
			for _, entity := range prefab.Entities {
				if entity.Model != nil && entity.Model.Asset == id {
					prefabUses[prefabID] = true
					break
				}
			}
		}
		for prefabID := range prefabUses {
			entry.Prefabs = append(entry.Prefabs, prefabID)
		}
		for entityID, entity := range document.Entities {
			if entity.Prefab != nil && prefabUses[entity.Prefab.Prefab] {
				entry.Instances = append(entry.Instances, entityID)
			}
		}
		sortIDs(entry.Entities)
		sortIDs(entry.Prefabs)
		sortIDs(entry.Instances)
		sortIDs(entry.UsedByAssets)
		entry.DirectCount = len(entry.Entities) + len(entry.Prefabs) + len(entry.UsedByAssets)
		report.Assets = append(report.Assets, entry)
	}
	return report
}

func sortIDs(values []ID) {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
}

func applyRegisterAsset(document *Document, operation Operation) ([]ID, error) {
	if operation.Asset == nil {
		return nil, fmt.Errorf("register-asset requires asset")
	}
	asset := *operation.Asset
	if err := validateAssetRecord(asset); err != nil {
		return nil, err
	}
	if existing, ok := document.Assets[asset.ID]; ok && existing.ContentHash != asset.ContentHash {
		return nil, fmt.Errorf("asset id collision")
	}
	if document.Assets == nil {
		document.Assets = map[ID]AssetRecord{}
	}
	document.Assets[asset.ID] = asset
	return []ID{asset.ID}, nil
}
func applyDeleteAsset(document *Document, operation Operation) ([]ID, error) {
	if operation.AssetID == "" {
		return nil, fmt.Errorf("delete-asset requires assetId")
	}
	if _, ok := document.Assets[operation.AssetID]; !ok {
		return nil, fmt.Errorf("asset %q does not exist", operation.AssetID)
	}
	for _, entity := range document.Entities {
		if entity.Model != nil && entity.Model.Asset == operation.AssetID {
			return nil, fmt.Errorf("asset %q is referenced by model %q", operation.AssetID, entity.ID)
		}
	}
	for _, prefab := range document.Prefabs {
		for _, entity := range prefab.Entities {
			if entity.Model != nil && entity.Model.Asset == operation.AssetID {
				return nil, fmt.Errorf("asset %q is referenced by prefab %q model %q", operation.AssetID, prefab.ID, entity.ID)
			}
		}
	}
	for _, asset := range document.Assets {
		for _, dependency := range asset.Dependencies {
			if dependency == operation.AssetID {
				return nil, fmt.Errorf("asset %q is referenced by asset %q", operation.AssetID, asset.ID)
			}
		}
	}
	delete(document.Assets, operation.AssetID)
	return []ID{operation.AssetID}, nil
}

func applyReimportAsset(document *Document, operation Operation) ([]ID, error) {
	if operation.AssetID == "" || operation.Asset == nil {
		return nil, fmt.Errorf("reimport-asset requires assetId and asset")
	}
	if _, ok := document.Assets[operation.AssetID]; !ok {
		return nil, fmt.Errorf("asset %q does not exist", operation.AssetID)
	}
	asset := *operation.Asset
	if err := validateAssetRecord(asset); err != nil {
		return nil, err
	}
	affected := []ID{operation.AssetID, asset.ID}
	document.Assets[asset.ID] = asset
	for id, entity := range document.Entities {
		if entity.Model != nil && entity.Model.Asset == operation.AssetID {
			entity.Model.Asset = asset.ID
			document.Entities[id] = entity
			affected = append(affected, id)
		}
	}
	for id, prefab := range document.Prefabs {
		changed := false
		for localID, entity := range prefab.Entities {
			if entity.Model != nil && entity.Model.Asset == operation.AssetID {
				entity.Model.Asset = asset.ID
				prefab.Entities[localID] = entity
				changed = true
			}
		}
		if changed {
			document.Prefabs[id] = prefab
			affected = append(affected, id)
		}
	}
	if operation.AssetID != asset.ID {
		delete(document.Assets, operation.AssetID)
	}
	return uniqueIDs(affected), nil
}
