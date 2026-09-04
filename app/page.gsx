package app

func Page() Node {
	return <main class="studio-shell" data-studio-demo={data.demoMode} aria-label="GoSX 3D Studio shared scene workbench">
		<a class="skip-link" href="#inspector-panel">Skip to Inspector</a>
		<a class="skip-link skip-link-agent" href="#agent-panel">Skip to Agent review</a>
		<h1 class="sr-only">GoSX 3D Studio</h1>
		<header class="menu-bar">
			<a href="/" data-gosx-link class="brand" aria-label="GoSX 3D Studio home">
				<span class="brand-mark">GoSX</span>
				<span>3D Studio</span>
			</a>
			<nav class="application-menu" aria-label="Application menu">
				<button type="button" class="desktop-only-control" id="studio-open-project" data-revision={data.revision} data-dirty={data.project.dirty} title="Open a GoSX 3D Studio project through the native desktop dialog">Open</button>
				<form data-gosx-form method="post" action={actionPath("saveProject")} class="menu-save-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="selection" value={data.inspector.id}></input>
					<button type="submit" title="Save current project">Save</button>
				</form>
				<form data-gosx-form method="post" action={actionPath("undoCommand")} class="menu-save-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="selection" value={data.inspector.id}></input>
					<input type="hidden" name="expectedRevision" value={data.revision}></input>
					<button type="submit" title="Undo the last committed command">Undo</button>
				</form>
				<form data-gosx-form method="post" action={actionPath("redoCommand")} class="menu-save-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="selection" value={data.inspector.id}></input>
					<input type="hidden" name="expectedRevision" value={data.revision}></input>
					<button type="submit" title="Redo the last undone command">Redo</button>
				</form>
				<button type="button" data-project-panel-toggle aria-controls="project-panel" aria-expanded="true" title="Hide Project / Assets panel">Assets</button>
				<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Scene</button>
				<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Model</button>
				<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Rig</button>
				<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Animate</button>
				<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Simulate</button>
				<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Render</button>
				<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Agent</button>
				<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Window</button>
				<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Help</button>
			</nav>
			<div class="build-badge">
				<span class="status-dot"></span>
				Scene rev {data.revision} · {data.project.state}
			</div>
		</header>
		<nav class="workspace-tabs" aria-label="Studio workspaces">
			<button type="button" class="active" aria-current="page">Layout</button>
			<span class="demo-mode-label">PUBLIC DEMO · AGENT PREP · HUMAN APPROVAL</span>
			<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Modeling</button>
			<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Sculpt</button>
			<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Rigging</button>
			<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Animation</button>
			<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Simulation</button>
			<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Shading</button>
			<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Rendering</button>
			<button type="button" disabled aria-disabled="true" title="Unavailable in this public build">Automation</button>
		</nav>
		<div class="tool-bar" role="toolbar" aria-label="Viewport tools">
			<div class="tool-group" role="group" aria-label="Transform tools">
				<button type="button" class="tool active" aria-label="Select" title="Select" aria-pressed="true" data-gizmo-mode="select">↖</button>
				<button type="button" class="tool" aria-label="Move" title="Move selected object" aria-pressed="false" data-gizmo-mode="translate">✣</button>
				<button type="button" class="tool" aria-label="Rotate" title="Rotate selected object" aria-pressed="false" data-gizmo-mode="rotate">↻</button>
				<button type="button" class="tool" aria-label="Scale" title="Scale selected object" aria-pressed="false" data-gizmo-mode="scale">↗</button>
			</div>
			<div class="tool-group segmented" role="group" aria-label="Transform orientation">
				<button type="button" class="active" aria-pressed="true">Global</button>
				<button type="button" disabled aria-disabled="true" aria-pressed="false" title="Pivot orientation is unavailable in this public build">Pivot</button>
			</div>
			<div class="tool-spacer"></div>
			<div class="tool-group transport" role="group" aria-label="Playback">
				<form data-gosx-form method="post" action={actionPath("playOp")} class="menu-save-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="selection" value={data.inspector.id}></input>
					<input type="hidden" name="op" value="enter"></input>
					<input type="hidden" name="simulationId" value={data.timeline.simulationId}></input>
					<button type="submit" aria-label="Enter play mode" title="Enter play mode (clones the document)" disabled={data.play.isActive}>▶</button>
				</form>
				<form data-gosx-form method="post" action={actionPath("playOp")} class="menu-save-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="selection" value={data.inspector.id}></input>
					<input type="hidden" name="op" value="step"></input>
					<input type="hidden" name="ticks" value="60"></input>
					<button type="submit" aria-label="Step 60 fixed ticks" title="Enter play mode before stepping fixed ticks" disabled={!data.play.isActive}>⏭</button>
				</form>
				<form data-gosx-form method="post" action={actionPath("playOp")} class="menu-save-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="selection" value={data.inspector.id}></input>
					<input type="hidden" name="op" value="exit"></input>
					<button type="submit" aria-label="Exit play mode" title="Enter play mode before exiting" disabled={!data.play.isActive}>■</button>
				</form>
			</div>
			<span class="frame-readout">play={data.play.active} · tick {data.play.tick} · {data.play.diffs} runtime diffs</span>
		</div>
		<div class="workbench">
			<aside id="project-panel" class="panel project-panel" aria-labelledby="project-title">
				<header class="panel-heading">
					<h2 id="project-title">Project / Assets</h2>
					<span class="panel-count">{data.assetCount}</span>
				</header>
				<form id="studio-asset-import" data-gosx-form method="post" action={actionPath("importAsset")} class="inspector-form asset-import-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="expectedRevision" value={data.revision}></input>
					<input type="hidden" name="selection" value={data.inspector.id}></input>
					<label>
						Project-local asset file
						<input id="studio-asset-path" name="path" placeholder="/path/to/model.glb" aria-describedby="studio-asset-dialog-status"></input>
					</label>
					<div class="asset-file-controls">
						<button id="studio-choose-asset" type="button" class="inspector-apply">Choose file…</button>
						<span id="studio-asset-dialog-status" class="placeholder-copy" aria-live="polite">Desktop chooser or trusted local path</span>
					</div>
					<p class="form-status">{action.message}</p>
					<button type="submit" class="inspector-apply">Inspect and import</button>
				</form>
				<label class="search-field">
					<span class="sr-only">Search assets</span>
					<input type="search" placeholder="Asset search unavailable" disabled aria-disabled="true" title="Asset search is unavailable in this public build"></input>
				</label>
				<ul class="tree asset-tree">
					<li class="expanded">
						<span>▾</span>
						<strong>{data.projectName}</strong>
					</li>
					<li class="depth-1 expanded">
						<span>▾</span>
						Assets
						<small>{data.assetCount}</small>
					</li>
					<Each of={data.assets} as="asset">
						<li class="depth-2 asset-record" title={asset.id}>
							<code>{asset.shortId}</code>
							<span>{asset.name}</span>
							<small>{asset.format} · {asset.bytes}</small>
							<small>{asset.dependencies}</small>
						</li>
					</Each>
				</ul>
				<form data-gosx-form method="post" action={actionPath("collectAssets")} class="inspector-form asset-gc-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="expectedRevision" value={data.revision}></input>
					<input type="hidden" name="confirmPlan" value={data.assetGC.fingerprint}></input>
					<input type="hidden" name="selection" value={data.inspector.id}></input>
					<strong>Unused assets · {data.assetGC.count}</strong>
					<small>{data.assetGC.bytes} reclaimable</small>
					<small>Unreferenced payloads · {data.assetGC.orphans} · {data.assetGC.orphanBytes}</small>
					<p class="placeholder-copy">{data.assetGC.status}</p>
					<button type="submit" class="inspector-apply" disabled={!data.assetGC.available}>Checkpoint and collect</button>
				</form>
			</aside>
			<aside class="panel hierarchy-panel" aria-labelledby="hierarchy-title">
				<header class="panel-heading">
					<h2 id="hierarchy-title">Scene Hierarchy</h2>
					<span class="panel-count" data-hierarchy-filter-count>{data.entityCount}</span>
				</header>
				<label class="search-field">
					<span class="sr-only">Search hierarchy</span>
					<input id="studio-hierarchy-search" type="search" placeholder="Search name, ID, or type…" autocomplete="off" data-hierarchy-filter aria-controls="studio-hierarchy-tree" aria-describedby="studio-hierarchy-search-status"></input>
				</label>
				<span id="studio-hierarchy-search-status" class="sr-only" aria-live="polite">{data.entityCount} scene entities</span>
				<ul id="studio-hierarchy-tree" class="tree hierarchy-tree" role="tree" aria-label="Scene entities">
					<Each of={data.hierarchy} as="item">
						<li class={item.class} data-hierarchy-row data-entity-name={item.name} data-hierarchy-id={item.id} data-entity-type={item.kind} role="none">
							<a href={"/?selection=" + item.id} data-gosx-link data-entity-id={item.id} role="treeitem" aria-level={item.level} aria-selected={item.selected} tabindex={item.tabIndex} aria-label={"Select " + item.name} title={item.name + " · " + item.id}>
								<code>{item.code}</code>
								<span class="entity-kind">{item.kind}</span>
								<span title={item.name}>{item.name}</span>
							</a>
						</li>
					</Each>
					<li data-hierarchy-empty role="status" hidden>No scene entities match this search.</li>
				</ul>
				<form data-gosx-form data-selection-bound="true" data-gosx-key={"entity-op-" + data.inspector.id} method="post" action={actionPath("entityOp")} class="inspector-form hierarchy-ops-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="target" value={data.inspector.id}></input>
					<input type="hidden" name="expectedRevision" value={data.revision}></input>
					<strong>Selected: {data.inspector.name}</strong>
					<label>
						Operation
						<select name="op">
							<option value="rename">rename</option>
							<option value="duplicate">duplicate</option>
							<option value="reparent">reparent</option>
							<option value="delete">delete</option>
						</select>
					</label>
					<label>
						New name (rename)
						<input name="name" value={data.inspector.name}></input>
					</label>
					<label>
						New parent (reparent)
						<select name="parent">
							<Each of={data.hierarchy} as="candidate">
								<option value={candidate.id}>{candidate.name}</option>
							</Each>
						</select>
					</label>
					<p class="form-status">{action.message}</p>
					<button type="submit" class="inspector-apply">Apply entity operation</button>
				</form>
			</aside>
			<section class="viewport-panel" aria-labelledby="viewport-title">
				<header class="viewport-header">
					<h2 id="viewport-title">Viewport</h2>
					<div role="toolbar" aria-label="Camera views">
						<button type="button" class="active" aria-pressed="true" data-camera-view="perspective">Perspective</button>
						<button type="button" aria-pressed="false" data-camera-view="front">Front</button>
						<button type="button" aria-pressed="false" data-camera-view="top">Top</button>
						<button type="button" aria-pressed="false" data-camera-view="right">Right</button>
					</div>
				</header>
				<div class="scene-stage" data-selection-id={data.inspector.id} data-camera-home={data.cameraHome} data-camera-focus-x={data.inspector.x} data-camera-focus-y={data.inspector.y} data-camera-focus-z={data.inspector.z}>
					<aside class="viewport-preview-card" data-webmcp-preview-badge hidden role="status" aria-live="polite" aria-label="Agent preview · not committed; awaiting human approval">
						<header>
							<span class="viewport-preview-title"><span aria-hidden="true"></span><strong>Agent preview</strong></span>
							<small>not committed</small>
						</header>
						<ul class="viewport-preview-changes" data-webmcp-preview-changes hidden aria-label="Previewed scene changes"></ul>
						<p data-webmcp-preview-revision>Canonical revision unchanged · human Apply only</p>
					</aside>
					<div class="viewport-approval-outcome" data-webmcp-approval-outcome hidden role="status" aria-live="polite">
						<span class="viewport-approval-check" aria-hidden="true">✓</span>
						<span>
							<strong>Human approved</strong>
							<small data-webmcp-approval-outcome-copy>Canonical scene updated</small>
						</span>
					</div>
					<div class="runtime-readout" aria-label="Observed runtime telemetry">
						<span>SCENE IR</span>
						<strong>mounted</strong>
						<span>BACKEND</span>
						<strong data-scene-runtime-backend>negotiating</strong>
						<span>REVISION</span>
						<strong>{data.revision}</strong>
					</div>
					<section class="judge-value-card" aria-label="Why agent-assisted scene editing is useful">
						<span class="overline">Agent-assisted scene editing</span>
						<strong>Find 1 object in 150. Stage 2 exact edits. Keep the only Apply.</strong>
						<p>A browser agent handles precise scene busywork in the live viewport. You inspect the reversible result and decide.</p>
						<ul aria-label="Demo capabilities">
							<li>3 action types</li>
							<li>12 edits max</li>
							<li>0 commit tools</li>
						</ul>
					</section>
					<Scene3D class="studio-scene" {...data.scene}>
						<div class="viewport-empty-state">
							<span class="overline">Scene3D unavailable</span>
							<h1>{data.appName}</h1>
							<p>The canonical document remains available to the native harness and action API.</p>
						</div>
					</Scene3D>
					<div class="axis-widget" aria-hidden="true">
						<span class="x">X</span>
						<span class="y">Y</span>
						<span class="z">Z</span>
					</div>
				</div>
			</section>
			<aside id="inspector-panel" class="panel inspector-panel" aria-labelledby="inspector-title" tabindex="-1">
				<header class="panel-heading">
					<h2 id="inspector-title">Inspector</h2>
					<span class="selection-id" title={data.inspector.name + " · " + data.inspector.id}>{data.inspector.name} · {data.inspector.id}</span>
				</header>
				<details open>
					<summary>Transform</summary>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"transform-" + data.inspector.id} method="post" action={actionPath("setTransform")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
					<div class="property-grid">
						<span>Location</span>
						<label>
							X
							<input name="x" value={data.inspector.x} inputmode="decimal"></input>
						</label>
						<label>
							Y
							<input name="y" value={data.inspector.y} inputmode="decimal"></input>
						</label>
						<label>
							Z
							<input name="z" value={data.inspector.z} inputmode="decimal"></input>
						</label>
					</div>
					<div class="property-grid">
						<span>Rotation</span>
						<label>
							X
							<input name="rx" value={data.inspector.rx} inputmode="decimal"></input>
						</label>
						<label>
							Y
							<input name="ry" value={data.inspector.ry} inputmode="decimal"></input>
						</label>
						<label>
							Z
							<input name="rz" value={data.inspector.rz} inputmode="decimal"></input>
						</label>
					</div>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Apply transform</button>
					</form>
				</details>
				<details open>
					<summary>Material</summary>
					<div class="material-preview" style={"--material-preview-color: " + data.inspector.materialColor}></div>
					<dl class="properties">
						<div>
							<dt>Canonical material</dt>
							<dd>{data.inspector.material}</dd>
						</div>
						<div>
							<dt>Surface</dt>
							<dd>{data.inspector.kind}</dd>
						</div>
						<div>
							<dt>Shader</dt>
							<dd class="runtime-label">{data.inspector.shader}</dd>
						</div>
						<div>
							<dt>Transmission</dt>
							<dd class="authored">{data.inspector.transmission}</dd>
						</div>
						<div>
							<dt>Roughness</dt>
							<dd>{data.inspector.roughness}</dd>
						</div>
					</dl>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"assign-material-" + data.inspector.id} method="post" action={actionPath("assignMaterial")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Assign material
							<select name="materialId">
								<Each of={data.materials} as="material">
									<option value={material.id} selected={material.id == data.inspector.materialId}>{material.name}</option>
								</Each>
							</select>
						</label>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Assign material</button>
					</form>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"material-" + data.inspector.id + "-" + data.inspector.materialId} method="post" action={actionPath("setMaterial")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="materialId" value={data.inspector.materialId}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<div class="vector-inputs">
							<label>
								Color
								<input name="color" value={data.inspector.materialColor}></input>
							</label>
							<label>
								Roughness
								<input name="roughness" value={data.inspector.roughness} inputmode="decimal"></input>
							</label>
							<label>
								Metalness
								<input name="metalness" value={data.inspector.metalness} inputmode="decimal"></input>
							</label>
						</div>
						<div class="vector-inputs">
							<label>
								Clearcoat
								<input name="clearcoat" value={data.inspector.clearcoat} inputmode="decimal"></input>
							</label>
							<label>
								Transmission
								<input name="transmission" value={data.inspector.transmission} inputmode="decimal"></input>
							</label>
							<label>
								Emissive
								<input name="emissive" value={data.inspector.emissive} inputmode="decimal"></input>
							</label>
						</div>
						<label>
							Base color texture (image assets)
							<select name="texture-color">
								<option value="" selected={data.inspector.textureColor == ""}>none</option>
								<Each of={data.imageAssets} as="image">
									<option value={image.id} selected={image.id == data.inspector.textureColor}>{image.name}</option>
								</Each>
							</select>
						</label>
						<label>
							Normal map
							<select name="texture-normal">
								<option value="" selected={data.inspector.textureNormal == ""}>none</option>
								<Each of={data.imageAssets} as="image">
									<option value={image.id} selected={image.id == data.inspector.textureNormal}>{image.name}</option>
								</Each>
							</select>
						</label>
						<label>
							Roughness map
							<select name="texture-roughness">
								<option value="" selected={data.inspector.textureRoughness == ""}>none</option>
								<Each of={data.imageAssets} as="image">
									<option value={image.id} selected={image.id == data.inspector.textureRoughness}>{image.name}</option>
								</Each>
							</select>
						</label>
						<label>
							Selena source (compiled before replacement; invalid source keeps the last valid material)
							<textarea name="selenaSource" rows="6" spellcheck="false">{data.inspector.selenaSource}</textarea>
						</label>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Apply material</button>
					</form>
				</details>
				<details>
					<summary>Model Asset</summary>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"reimport-" + data.inspector.id} method="post" action={actionPath("reimportAsset")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Content-addressed asset ID
							<input name="assetId" value={data.inspector.assetId} readonly></input>
						</label>
						<label>
							Replacement project-local file
							<input name="path" placeholder={data.inspector.assetName}></input>
						</label>
						<p class="placeholder-copy">Reimport creates a new content identity and migrates every SceneDoc and prefab model reference atomically.</p>
						<button type="submit" class="inspector-apply">Reimport and retarget</button>
					</form>
				</details>
				<details>
					<summary>Modifiers</summary>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"solidify-" + data.inspector.id} method="post" action={actionPath("setSolidify")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Stable modifier ID
							<input name="modifierId" value={data.inspector.modifierId}></input>
						</label>
						<label>
							Solidify thickness
							<input name="thickness" value={data.inspector.thickness} inputmode="decimal"></input>
						</label>
						<p class="placeholder-copy">{data.inspector.modifierStatus}</p>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Set solidify</button>
					</form>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"subdivision-" + data.inspector.id} method="post" action={actionPath("setSubdivision")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Subdivision modifier ID
							<input name="modifierId" value={data.inspector.subdivisionId}></input>
						</label>
						<label>
							Subdivision levels
							<input name="levels" value={data.inspector.subdivisionLevels} inputmode="numeric"></input>
						</label>
						<button type="submit" class="inspector-apply">Set subdivision</button>
					</form>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"modifier-order-" + data.inspector.id} method="post" action={actionPath("reorderModifier")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Modifier ID
							<input name="modifierId" value={data.inspector.activeModifierId}></input>
						</label>
						<label>
							Stack index
							<input name="modifierIndex" value="0" inputmode="numeric"></input>
						</label>
						<button type="submit" class="inspector-apply">Move modifier</button>
					</form>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"modifier-apply-" + data.inspector.id} method="post" action={actionPath("applyModifier")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Bake through modifier ID
							<input name="modifierId" value={data.inspector.activeModifierId}></input>
						</label>
						<button type="submit" class="inspector-apply">Apply through modifier</button>
					</form>
				</details>
				<details>
					<summary>Modeling</summary>
					<p class="placeholder-copy">{data.modeling.status}</p>
					<dl class="properties">
						<div>
							<dt>Vertices</dt>
							<dd>{data.modeling.vertices}</dd>
						</div>
						<div>
							<dt>Edges</dt>
							<dd>{data.modeling.edges}</dd>
						</div>
						<div>
							<dt>Faces</dt>
							<dd>{data.modeling.faces}</dd>
						</div>
						<div>
							<dt>Sub-object selection</dt>
							<dd class="authored">{data.modeling.selectionSummary}</dd>
						</div>
					</dl>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"subobjects-" + data.inspector.id} method="post" action={actionPath("selectSubobjects")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Mode
							<select name="mode">
								<option value="face">face</option>
								<option value="vertex">vertex</option>
								<option value="edge">edge</option>
							</select>
						</label>
						<label>
							IDs (comma separated; first face is {data.modeling.firstFace})
							<textarea name="ids" rows="2" spellcheck="false"></textarea>
						</label>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Select sub-objects</button>
					</form>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"mesh-operator-" + data.inspector.id} method="post" action={actionPath("meshOperator")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Operator
							<select name="operator">
								<option value="extrude-faces">extrude-faces</option>
								<option value="inset-faces">inset-faces</option>
								<option value="triangulate-faces">triangulate-faces</option>
								<option value="weld-vertices">weld-vertices</option>
								<option value="dissolve-edges">dissolve-edges</option>
								<option value="bevel-edges">bevel-edges</option>
								<option value="recalculate-normals">recalculate-normals</option>
								<option value="project-planar-uv">project-planar-uv</option>
							</select>
						</label>
						<label>
							IDs (blank uses the current sub-object selection)
							<textarea name="ids" rows="2" spellcheck="false"></textarea>
						</label>
						<div class="vector-inputs">
							<label>
								Distance
								<input name="distance" value="0.25" inputmode="decimal"></input>
							</label>
							<label>
								Amount
								<input name="amount" value="0.25" inputmode="decimal"></input>
							</label>
							<label>
								Tolerance
								<input name="tolerance" value="0.0001" inputmode="decimal"></input>
							</label>
						</div>
						<label>
							Planar projection axis (xz, xy, or yz)
							<input name="projection" value="xz"></input>
						</label>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Run operator</button>
					</form>
				</details>
				<details>
					<summary>Prefabs</summary>
					<dl class="properties">
						<Each of={data.prefabs} as="prefab">
							<div>
								<dt>{prefab.name}</dt>
								<dd>{prefab.kind} · {prefab.entities} entities</dd>
							</div>
						</Each>
					</dl>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"prefab-op-" + data.inspector.id} method="post" action={actionPath("prefabOp")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Operation
							<select name="op">
								<option value="capture">capture selection as prefab</option>
								<option value="instantiate">instantiate</option>
								<option value="override">set instance override</option>
								<option value="delete">delete definition</option>
							</select>
						</label>
						<div class="vector-inputs">
							<label>
								Prefab ID
								<input name="prefabId" placeholder="my-prefab"></input>
							</label>
							<label>
								Name (capture)
								<input name="name" placeholder="My prefab"></input>
							</label>
						</div>
						<label>
							Parent (instantiate)
							<select name="parent">
								<Each of={data.hierarchy} as="candidate">
									<option value={candidate.id}>{candidate.name}</option>
								</Each>
							</select>
						</label>
						<div class="vector-inputs">
							<label>
								Local entity (override)
								<input name="prefabEntity" placeholder="local-id"></input>
							</label>
							<label>
								Material (override)
								<select name="material">
									<option value="">keep</option>
									<Each of={data.materials} as="material">
										<option value={material.id}>{material.name}</option>
									</Each>
								</select>
							</label>
						</div>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Apply prefab operation</button>
					</form>
				</details>
				<details>
					<summary>Physics</summary>
					<form data-gosx-form data-selection-bound="true" data-gosx-key={"physics-" + data.inspector.id} method="post" action={actionPath("setPhysics")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<div class="vector-inputs">
							<label>
								Body
								<select name="kind">
									<option value="dynamic">dynamic</option>
									<option value="static">static</option>
									<option value="kinematic">kinematic</option>
								</select>
							</label>
							<label>
								Mass
								<input name="mass" value="1" inputmode="decimal"></input>
							</label>
							<label>
								Gravity scale
								<input name="gravityScale" value="1" inputmode="decimal"></input>
							</label>
						</div>
						<div class="vector-inputs">
							<label>
								Friction
								<input name="friction" value="0.4" inputmode="decimal"></input>
							</label>
							<label>
								Restitution
								<input name="restitution" value="0.2" inputmode="decimal"></input>
							</label>
							<label>
								Collider
								<select name="colliderKind">
									<option value="box">box</option>
									<option value="sphere">sphere</option>
									<option value="capsule">capsule</option>
									<option value="plane">plane</option>
								</select>
							</label>
						</div>
						<div class="vector-inputs">
							<label>
								Radius
								<input name="radius" value="0.5" inputmode="decimal"></input>
							</label>
							<label>
								Half height (capsule)
								<input name="halfHeight" value="0.5" inputmode="decimal"></input>
							</label>
							<label>
								Sensor
								<input type="checkbox" name="sensor" value="true"></input>
							</label>
						</div>
						<div class="vector-inputs">
							<label>
								Extent X (box)
								<input name="extentX" value="0.5" inputmode="decimal"></input>
							</label>
							<label>
								Extent Y
								<input name="extentY" value="0.5" inputmode="decimal"></input>
							</label>
							<label>
								Extent Z
								<input name="extentZ" value="0.5" inputmode="decimal"></input>
							</label>
						</div>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Apply physics body</button>
					</form>
				</details>
				<details>
					<summary>Metadata</summary>
					<p class="placeholder-copy">Stable ID {data.inspector.id} · revision {data.revision}</p>
				</details>
			</aside>
			<section class="timeline-panel" aria-labelledby="timeline-title">
				<header class="panel-heading">
					<h2 id="timeline-title">Timeline</h2>
					<span class="panel-note">{data.timeline.state}</span>
				</header>
				<div class="timeline-ruler">
					<span>0.000</span>
					<span>Clip</span>
					<span>{data.timeline.clipName}</span>
					<span>{data.timeline.duration}s</span>
					<span>{data.timeline.tickRate} Hz</span>
					<span>rev {data.revision}</span>
				</div>
				<div class="timeline-tools">
					<form data-gosx-form method="post" action={actionPath("setBonePose")} class="timeline-form" title="Requires an armature with at least one bone">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="armatureId" value={data.timeline.armatureId}></input>
						<input type="hidden" name="boneId" value={data.timeline.boneId}></input>
						<strong>Pose · {data.timeline.boneName}</strong>
						<label>RX <input name="rx" value={data.timeline.rx} inputmode="decimal" disabled={!data.timeline.boneAvailable}></input></label>
						<label>RY <input name="ry" value={data.timeline.ry} inputmode="decimal" disabled={!data.timeline.boneAvailable}></input></label>
						<label>RZ <input name="rz" value={data.timeline.rz} inputmode="decimal" disabled={!data.timeline.boneAvailable}></input></label>
						<button type="submit" disabled={!data.timeline.boneAvailable}>Set pose</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("setAnimationKey")} class="timeline-form" title="Requires an animation clip with a track">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="clipId" value={data.timeline.clipId}></input>
						<input type="hidden" name="trackId" value={data.timeline.trackId}></input>
						<strong>Key · {data.timeline.clipName}</strong>
						<label>Time <input name="time" value={data.timeline.keyTime} inputmode="decimal" disabled={!data.timeline.clipAvailable}></input></label>
						<label>RX <input name="rx" value={data.timeline.rx} inputmode="decimal" disabled={!data.timeline.clipAvailable}></input></label>
						<label>RY <input name="ry" value={data.timeline.ry} inputmode="decimal" disabled={!data.timeline.clipAvailable}></input></label>
						<label>RZ <input name="rz" value={data.timeline.rz} inputmode="decimal" disabled={!data.timeline.clipAvailable}></input></label>
						<button type="submit" disabled={!data.timeline.clipAvailable}>Insert key</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("solveIK")} class="timeline-form compact" title="Requires an armature IK constraint">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="armatureId" value={data.timeline.armatureId}></input>
						<input type="hidden" name="constraintId" value={data.timeline.constraintId}></input>
						<strong>IK · {data.timeline.constraintId}</strong>
						<button type="submit" disabled={!data.timeline.ikAvailable}>Solve deterministically</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("simulateTicks")} class="timeline-form compact" title="Requires a simulation profile">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="simulationId" value={data.timeline.simulationId}></input>
						<strong>Sim · {data.timeline.simulationName}</strong>
						<label>Ticks <input name="ticks" value={data.timeline.ticks} inputmode="numeric" disabled={!data.timeline.simulationAvailable}></input></label>
						<button type="submit" disabled={!data.timeline.simulationAvailable}>Advance fixed ticks</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("retargetAnimation")} class="timeline-form" title="Requires both a retarget map and source clip">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="retargetMapId" value={data.timeline.retargetMapId}></input>
						<input type="hidden" name="sourceClipId" value={data.timeline.clipId}></input>
						<strong>Retarget · {data.timeline.retargetName}</strong>
						<label>Stable ID <input name="newId" value={data.timeline.retargetClipId} disabled={!data.timeline.retargetAvailable}></input></label>
						<label>Name <input name="name" value="Retargeted Clip" disabled={!data.timeline.retargetAvailable}></input></label>
						<button type="submit" disabled={!data.timeline.retargetAvailable}>Retarget clip</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("setAnimationParameter")} class="timeline-form compact" title="Requires a state machine parameter">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="machineId" value={data.timeline.machineId}></input>
						<input type="hidden" name="parameter" value={data.timeline.machineParameter}></input>
						<strong>{data.timeline.machineName} · {data.timeline.machineState}</strong>
						<label>{data.timeline.machineParameterLabel} <input name="number" value={data.timeline.machineValue} inputmode="decimal" disabled={!data.timeline.machineParameterAvailable}></input></label>
						<button type="submit" disabled={!data.timeline.machineParameterAvailable}>Set parameter</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("stepAnimationMachine")} class="timeline-form compact" title="Requires an animation state machine">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="machineId" value={data.timeline.machineId}></input>
						<strong>Governed transition</strong>
						<label>Δ seconds <input name="deltaTime" value={data.timeline.deltaTime} inputmode="decimal" disabled={!data.timeline.machineAvailable}></input></label>
						<button type="submit" disabled={!data.timeline.machineAvailable}>Evaluate + sample</button>
					</form>
				</div>
			</section>
			<aside id="agent-panel" class="agent-panel" aria-labelledby="agent-title" tabindex="-1">
				<header class="panel-heading">
					<h2 id="agent-title">Agent Collaboration</h2>
					<span class="runtime-label">0 commit tools</span>
				</header>
				<div class="webmcp-status-region" role="status" aria-live="polite" aria-atomic="true">
					<div class="webmcp-status-banner" data-webmcp-status-panel data-state="detecting">
						<span class="webmcp-status-dot" aria-hidden="true"></span>
						<span>
							<strong>WebMCP</strong>
							<small data-webmcp-status-label>Detecting browser support</small>
						</span>
						<code data-webmcp-tool-count>0 tools</code>
					</div>
					<p class="webmcp-status-copy" data-webmcp-status-message>The complete human editing surface remains available while WebMCP initializes.</p>
				</div>
				<details class="webmcp-tool-disclosure" data-webmcp-idle-only>
					<summary>
						<span>Native WebMCP</span>
						<strong>4 registered tools</strong>
					</summary>
					<ul aria-label="Registered WebMCP tools">
						<li><code>scene_get_state</code><span>inspect canonical state</span></li>
						<li><code>scene_find_objects</code><span>find stable object IDs</span></li>
						<li><code>scene_focus_object</code><span>focus the visible Studio UI</span></li>
						<li><code>scene_preview_actions</code><span>stage a reversible preview</span></li>
					</ul>
					<a href="https://github.com/M31-Labs/gosx3d-studio/blob/main/public/studio-webmcp.js" target="_blank" rel="noreferrer">View registerTool source ↗</a>
				</details>
				<div class="webmcp-authority-cue" role="note" aria-label="Authority boundary: agents can inspect, focus, and stage proposals. A person must review and apply changes.">
					<span class="webmcp-agent-authority">
						<strong>Agent</strong>
						<small>inspect · focus · stage</small>
					</span>
					<span class="webmcp-authority-boundary" aria-hidden="true">no auto-commit</span>
					<span class="webmcp-human-authority">
						<strong>Human</strong>
						<small>review · apply</small>
					</span>
				</div>
				<ol class="webmcp-flow" aria-label="WebMCP collaboration flow">
					<li data-webmcp-flow-tool="scene_get_state" data-label="Inspect" data-state="idle" aria-label="Inspect: idle"><span aria-hidden="true">1</span><strong aria-hidden="true">Inspect</strong></li>
					<li data-webmcp-flow-tool="scene_find_objects" data-label="Find" data-state="idle" aria-label="Find: idle"><span aria-hidden="true">2</span><strong aria-hidden="true">Find</strong></li>
					<li data-webmcp-flow-tool="scene_focus_object" data-label="Focus" data-state="idle" aria-label="Focus: idle"><span aria-hidden="true">3</span><strong aria-hidden="true">Focus</strong></li>
					<li data-webmcp-flow-tool="scene_preview_actions" data-label="Stage" data-state="idle" aria-label="Stage: idle"><span aria-hidden="true">4</span><strong aria-hidden="true">Stage</strong></li>
				</ol>
				<section class="webmcp-demo-mission" data-webmcp-idle-only aria-labelledby="webmcp-mission-title">
					<span class="overline" id="webmcp-mission-title">Try it in 30 seconds</span>
					<strong class="webmcp-mission-heading">Ask for one precise scene edit.</strong>
					<p data-webmcp-demo-prompt>Inspect the current scene, find and focus the object named Board, then stage—without committing—a proposal that renames it Launch Board and assigns the Brushed Steel material. Explain the revision boundary.</p>
					<button type="button" data-webmcp-copy-prompt disabled>Copy demo prompt</button>
					<small class="webmcp-mission-steps">1 Copy task · 2 Watch four calls · 3 Review and decide</small>
					<small class="webmcp-chrome-help">Chrome 149+: DevTools → Application → WebMCP</small>
					<small data-webmcp-copy-status aria-live="polite">Paste it into the browser agent.</small>
				</section>
				<section class="webmcp-trace-shell" aria-labelledby="webmcp-trace-title">
					<header>
						<span class="overline" id="webmcp-trace-title">WebMCP tool receipts</span>
						<small>this browser session</small>
					</header>
					<div class="webmcp-trace" data-webmcp-trace role="log" aria-live="polite" aria-relevant="additions text">
						<p class="webmcp-trace-empty">Typed-call receipts will appear here.</p>
					</div>
				</section>
				<div class="studio-demo-reset" data-studio-demo-panel data-webmcp-idle-only hidden>
					<span>
						<strong>Shared public demo</strong>
						<small>One ephemeral scene shared across visitors. Reset clears current edits and staged proposals.</small>
						<small class="studio-demo-state" data-studio-demo-state aria-live="polite">Checking showcase baseline…</small>
					</span>
					<button type="button" data-studio-demo-reset data-revision={data.revision}>Reset shared scene</button>
				</div>
				<p class="placeholder-copy" data-webmcp-idle-only>{data.agent.authority}</p>
				<div class="proposal" data-webmcp-proposal data-revision={data.revision}>
					<span class="overline">Latest staged proposal</span>
					<p data-webmcp-proposal-summary>{data.agent.proposalSummary}</p>
					<p data-webmcp-proposal-rationale hidden></p>
					<ul class="webmcp-change-list" data-webmcp-proposal-changes hidden></ul>
					<dl>
						<div>
							<dt>Actor</dt>
							<dd data-webmcp-proposal-actor>{data.agent.proposalActor}</dd>
						</div>
						<div>
							<dt>Review checks</dt>
							<dd data-webmcp-proposal-policy>Waiting for proposal</dd>
						</div>
						<div>
							<dt>Revision boundary</dt>
							<dd data-webmcp-proposal-revision>{data.agent.proposalRevision}</dd>
						</div>
						<div>
							<dt>Review window</dt>
							<dd data-webmcp-proposal-expiry>not staged</dd>
						</div>
						<div>
							<dt>Affected</dt>
							<dd data-webmcp-proposal-affected>{data.agent.proposalAffected}</dd>
						</div>
						<div>
							<dt>Result fingerprint</dt>
							<dd class="pending" data-webmcp-proposal-fingerprint>{data.agent.proposalFingerprint}</dd>
						</div>
					</dl>
					<details class="webmcp-policy-details">
						<summary>What was checked</summary>
						<p data-webmcp-proposal-policy-reasons>Stage a proposal to see the safety checks for each change.</p>
					</details>
					<div class="webmcp-review-actions" data-webmcp-review-actions hidden>
						<small class="webmcp-review-gate" data-webmcp-review-gate>Human-only approval · creates the next revision</small>
						<button type="button" data-webmcp-discard>Discard</button>
						<button type="button" class="primary" data-webmcp-commit>Apply staged changes</button>
					</div>
				</div>
				<div class="proposal" data-webmcp-idle-only>
					<span class="overline">Shared workspace activity</span>
					<dl>
						<div>
							<dt>Agent transactions</dt>
							<dd class="runtime-label">{data.agent.agentCount}</dd>
						</div>
						<div>
							<dt>Human transactions</dt>
							<dd class="authored">{data.agent.humanCount}</dd>
						</div>
					</dl>
					<p class="placeholder-copy">Agent proposals and visible UI approvals share one canonical, revision-safe workspace.</p>
				</div>
				<div class="agent-actions">
					<a class="primary" href="/api/studio/actions">Action catalog ↗</a>
					<a href="/api/studio/manifest">Inspect manifest ↗</a>
				</div>
			</aside>
		</div>
		<section class="telemetry-dock" aria-label="Diagnostics and telemetry">
			<nav role="tablist" aria-label="Diagnostic tabs">
				<button type="button" role="tab" disabled aria-disabled="true" aria-selected="false" title="Unavailable in this public build">Console</button>
				<button type="button" role="tab" disabled aria-disabled="true" aria-selected="false" title="Unavailable in this public build">Render Graph</button>
				<button type="button" role="tab" disabled aria-disabled="true" aria-selected="false" title="Unavailable in this public build">Harness</button>
				<button type="button" role="tab" disabled aria-disabled="true" aria-selected="false" title="Unavailable in this public build">Ray Trace</button>
				<button type="button" role="tab" disabled aria-disabled="true" aria-selected="false" title="Unavailable in this public build">Selena</button>
				<button type="button" role="tab" disabled aria-disabled="true" aria-selected="false" title="Unavailable in this public build">Performance</button>
				<button id="agent-activity-tab" type="button" class="active" role="tab" aria-selected="true" aria-controls="agent-activity-panel">Agent Activity</button>
				<a class={"telemetry-proof certification-state certification-state-" + data.certification.certState} href="/api/studio/certification/evidence" title="Open current deterministic evidence">Evidence {data.certification.liveChecksPass}/{data.certification.liveChecksTotal} · {data.certification.certState} · rev {data.certification.certRevision} ↗</a>
			</nav>
			<div id="agent-activity-panel" class="telemetry-content" role="tabpanel" aria-labelledby="agent-activity-tab">
				<div class="console-lines">
					<p>
						<time>rev {data.revision}</time>
						<span class="runtime-label">SYSTEM</span>
						{data.historySummary}
					</p>
					<Each of={data.history} as="entry">
						<p>
							<time>rev {entry.afterRevision}</time>
							<span class="author-label">{entry.actorLabel}</span>
							<span title={entry.transactionID} data-transaction-id={entry.transactionID}>{entry.summary}</span>
						</p>
					</Each>
				</div>
				<div class="certification-card">
					<span>Deterministic evidence</span>
					<strong>{data.certification.liveChecksPass}/{data.certification.liveChecksTotal} PASS</strong>
					<small>Live invariants for the current canonical scene</small>
					<small class={"certification-state certification-state-" + data.certification.certState}>evidence {data.certification.certState} · revision {data.certification.certRevision}</small>
					<details class="certification-details">
						<summary>Capability evidence · {data.certification.available} complete · {data.certification.total} tracked</summary>
						<ul class="certification-dimensions">
							<Each of={data.certification.dimensions} as="dimension">
								<li title={dimension.evidence}>
									<span>{dimension.id}</span>
									<strong>{dimension.status}</strong>
								</li>
							</Each>
						</ul>
					</details>
				</div>
				<div class="manifest-link">
					<span>Agent-readable contract</span>
					<a href="/api/studio/manifest">/api/studio/manifest ↗</a>
					<a href="/api/studio/certification/evidence">Current deterministic evidence ↗</a>
				</div>
			</div>
		</section>
		<script src="/studio-selection.js" defer></script>
		<script src="/studio-project.js" defer></script>
		<script src="/studio-certification.js" defer></script>
		<script src="/studio-gizmo.js" defer></script>
		<script src="/studio-camera.js" defer></script>
		<script src="/studio-interactions.js" defer></script>
		<script src="/studio-webmcp-ui.js" defer></script>
		<script src="/studio-webmcp.js" defer></script>
		<footer class="status-bar">
			<span class="runtime-label">GoSX server</span>
			<span>Scene revision {data.revision}</span>
			<span>Saved revision {data.project.savedRevision}</span>
			<span class="author-label">Project {data.project.state}</span>
			<span data-scene-runtime-status>Scene3D initializing</span>
			<span>{data.entityCount} scene entities</span>
			<span class="status-spacer"></span>
			<strong class="runtime-label">REV {data.revision} CURRENT</strong>
			<span>Industrial Void</span>
		</footer>
	</main>
}
