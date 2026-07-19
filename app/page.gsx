package app

func Page() Node {
	return <main class="studio-shell" aria-label="GoSX 3D Studio M1 foundation workbench">
		<header class="menu-bar">
			<a href="/" data-gosx-link class="brand" aria-label="GoSX 3D Studio home">
				<span class="brand-mark">GoSX</span>
				<span>3D Studio</span>
			</a>
			<nav class="application-menu" aria-label="Application menu">
				<button type="button" id="studio-open-project" data-revision={data.revision} data-dirty={data.project.dirty} title="Open a GoSX 3D Studio project through the native desktop dialog">Open</button>
				<form data-gosx-form method="post" action={actionPath("saveProject")} class="menu-save-form">
					<input type="hidden" name="csrf_token" value={csrf.token}></input>
					<input type="hidden" name="selection" value={data.inspector.id}></input>
					<button type="submit" title={"Save " + data.project.directory}>Save</button>
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
				<button type="button">Assets</button>
				<button type="button">Scene</button>
				<button type="button">Model</button>
				<button type="button">Rig</button>
				<button type="button">Animate</button>
				<button type="button">Simulate</button>
				<button type="button">Render</button>
				<button type="button">Agent</button>
				<button type="button">Window</button>
				<button type="button">Help</button>
			</nav>
			<div class="build-badge">
				<span class="status-dot"></span>
				M1 foundation · {data.project.state}
			</div>
		</header>
		<nav class="workspace-tabs" aria-label="Studio workspaces">
			<button type="button" class="active" aria-current="page">Layout</button>
			<button type="button">Modeling</button>
			<button type="button" disabled>Sculpt</button>
			<button type="button" disabled>Rigging</button>
			<button type="button" disabled>Animation</button>
			<button type="button" disabled>Simulation</button>
			<button type="button" disabled>Shading</button>
			<button type="button" disabled>Rendering</button>
			<button type="button">Automation</button>
		</nav>
		<div class="tool-bar" aria-label="Viewport tools">
			<div class="tool-group">
				<button type="button" class="tool active" aria-label="Select" data-gizmo-mode="">↖</button>
				<button type="button" class="tool" aria-label="Move" data-gizmo-mode="translate">✣</button>
				<button type="button" class="tool" aria-label="Rotate" data-gizmo-mode="rotate">↻</button>
				<button type="button" class="tool" aria-label="Scale" data-gizmo-mode="scale">↗</button>
			</div>
			<div class="tool-group segmented" aria-label="Transform orientation">
				<button type="button" class="active">Global</button>
				<button type="button">Pivot</button>
			</div>
			<div class="tool-spacer"></div>
			<div class="tool-group transport" aria-label="Playback">
				<button type="button" disabled>◀</button>
				<button type="button" disabled>▶</button>
				<button type="button" disabled>■</button>
			</div>
			<span class="frame-readout">Frame 0000 · 24 fps</span>
		</div>
		<div class="workbench">
			<aside class="panel project-panel" aria-labelledby="project-title">
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
					<input type="search" placeholder="Search assets…"></input>
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
					<p class="placeholder-copy">{data.assetGC.status}</p>
					<button type="submit" class="inspector-apply" disabled={!data.assetGC.available}>Checkpoint and collect</button>
				</form>
			</aside>
			<aside class="panel hierarchy-panel" aria-labelledby="hierarchy-title">
				<header class="panel-heading">
					<h2 id="hierarchy-title">Scene Hierarchy</h2>
					<span class="panel-count">{data.entityCount}</span>
				</header>
				<label class="search-field">
					<span class="sr-only">Search hierarchy</span>
					<input type="search" placeholder="Search hierarchy…"></input>
				</label>
				<ul class="tree hierarchy-tree">
					<Each of={data.hierarchy} as="item">
						<li class={item.class}>
							<a href={"/?selection=" + item.id} data-gosx-link aria-label={"Select " + item.name}>
								<code>{item.code}</code>
								<span class="entity-kind">{item.kind}</span>
								<span>{item.name}</span>
							</a>
						</li>
					</Each>
				</ul>
			</aside>
			<section class="viewport-panel" aria-labelledby="viewport-title">
				<header class="viewport-header">
					<h2 id="viewport-title">Viewport</h2>
					<div>
						<button type="button" data-camera-view="perspective">Perspective</button>
						<button type="button" data-camera-view="front">Front</button>
						<button type="button" data-camera-view="top">Top</button>
						<button type="button" data-camera-view="right">Right</button>
					</div>
				</header>
				<div class="scene-stage" data-camera-home={data.cameraHome} data-camera-focus-x={data.inspector.x} data-camera-focus-y={data.inspector.y} data-camera-focus-z={data.inspector.z}>
					<div class="runtime-readout" aria-label="Observed runtime telemetry">
						<span>SCENE IR</span>
						<strong>mounted</strong>
						<span>BACKEND</span>
						<strong>negotiated</strong>
						<span>REVISION</span>
						<strong>{data.revision}</strong>
					</div>
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
			<aside class="panel inspector-panel" aria-labelledby="inspector-title">
				<header class="panel-heading">
					<h2 id="inspector-title">Inspector</h2>
					<span class="selection-id">{data.inspector.name} · {data.inspector.id}</span>
				</header>
				<details open>
					<summary>Transform</summary>
					<form data-gosx-form method="post" action={actionPath("setTransform")} class="inspector-form">
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
					<div class="material-preview authored-material"></div>
					<dl class="properties">
						<div>
							<dt>Material</dt>
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
					<form data-gosx-form method="post" action={actionPath("assignMaterial")} class="inspector-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="target" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<label>
							Assign material
							<select name="materialId">
								<Each of={data.materials} as="material">
									<option value={material.id}>{material.name}</option>
								</Each>
							</select>
						</label>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Assign material</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("setMaterial")} class="inspector-form">
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
							Selena source (compiled before replacement; invalid source keeps the last valid material)
							<textarea name="selenaSource" rows="6" spellcheck="false">{data.inspector.selenaSource}</textarea>
						</label>
						<p class="form-status">{action.message}</p>
						<button type="submit" class="inspector-apply">Apply material</button>
					</form>
				</details>
				<details>
					<summary>Model Asset</summary>
					<form data-gosx-form method="post" action={actionPath("reimportAsset")} class="inspector-form">
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
					<form data-gosx-form method="post" action={actionPath("setSolidify")} class="inspector-form">
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
					<form data-gosx-form method="post" action={actionPath("setSubdivision")} class="inspector-form">
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
					<form data-gosx-form method="post" action={actionPath("reorderModifier")} class="inspector-form">
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
					<form data-gosx-form method="post" action={actionPath("applyModifier")} class="inspector-form">
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
					<form data-gosx-form method="post" action={actionPath("selectSubobjects")} class="inspector-form">
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
					<form data-gosx-form method="post" action={actionPath("meshOperator")} class="inspector-form">
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
					<summary>Physics</summary>
					<p class="placeholder-copy">Component surface reserved.</p>
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
					<form data-gosx-form method="post" action={actionPath("setBonePose")} class="timeline-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="armatureId" value={data.timeline.armatureId}></input>
						<input type="hidden" name="boneId" value={data.timeline.boneId}></input>
						<strong>Pose · {data.timeline.boneName}</strong>
						<label>RX <input name="rx" value={data.timeline.rx} inputmode="decimal"></input></label>
						<label>RY <input name="ry" value={data.timeline.ry} inputmode="decimal"></input></label>
						<label>RZ <input name="rz" value={data.timeline.rz} inputmode="decimal"></input></label>
						<button type="submit">Set pose</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("setAnimationKey")} class="timeline-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="clipId" value={data.timeline.clipId}></input>
						<input type="hidden" name="trackId" value={data.timeline.trackId}></input>
						<strong>Key · {data.timeline.clipName}</strong>
						<label>Time <input name="time" value={data.timeline.keyTime} inputmode="decimal"></input></label>
						<label>RX <input name="rx" value={data.timeline.rx} inputmode="decimal"></input></label>
						<label>RY <input name="ry" value={data.timeline.ry} inputmode="decimal"></input></label>
						<label>RZ <input name="rz" value={data.timeline.rz} inputmode="decimal"></input></label>
						<button type="submit">Insert key</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("solveIK")} class="timeline-form compact">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="armatureId" value={data.timeline.armatureId}></input>
						<input type="hidden" name="constraintId" value={data.timeline.constraintId}></input>
						<strong>IK · {data.timeline.constraintId}</strong>
						<button type="submit">Solve deterministically</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("simulateTicks")} class="timeline-form compact">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="simulationId" value={data.timeline.simulationId}></input>
						<strong>Sim · {data.timeline.simulationName}</strong>
						<label>Ticks <input name="ticks" value={data.timeline.ticks} inputmode="numeric"></input></label>
						<button type="submit">Advance fixed ticks</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("retargetAnimation")} class="timeline-form">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="retargetMapId" value={data.timeline.retargetMapId}></input>
						<input type="hidden" name="sourceClipId" value={data.timeline.clipId}></input>
						<strong>Retarget · {data.timeline.retargetName}</strong>
						<label>Stable ID <input name="newId" value={data.timeline.retargetClipId}></input></label>
						<label>Name <input name="name" value="Retargeted Clip"></input></label>
						<button type="submit">Retarget clip</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("setAnimationParameter")} class="timeline-form compact">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="machineId" value={data.timeline.machineId}></input>
						<input type="hidden" name="parameter" value={data.timeline.machineParameter}></input>
						<strong>{data.timeline.machineName} · {data.timeline.machineState}</strong>
						<label>{data.timeline.machineParameter} <input name="number" value={data.timeline.machineValue} inputmode="decimal"></input></label>
						<button type="submit">Set parameter</button>
					</form>
					<form data-gosx-form method="post" action={actionPath("stepAnimationMachine")} class="timeline-form compact">
						<input type="hidden" name="csrf_token" value={csrf.token}></input>
						<input type="hidden" name="selection" value={data.inspector.id}></input>
						<input type="hidden" name="expectedRevision" value={data.revision}></input>
						<input type="hidden" name="machineId" value={data.timeline.machineId}></input>
						<strong>Governed transition · Arbiter</strong>
						<label>Δ seconds <input name="deltaTime" value={data.timeline.deltaTime} inputmode="decimal"></input></label>
						<button type="submit">Evaluate + sample</button>
					</form>
				</div>
			</section>
			<aside class="agent-panel" aria-labelledby="agent-title">
				<header class="panel-heading">
					<h2 id="agent-title">Agent Actions</h2>
					<span class="runtime-label">typed surface</span>
				</header>
				<p class="placeholder-copy">{data.agent.authority}</p>
				<div class="proposal">
					<span class="overline">Latest agent proposal</span>
					<p>{data.agent.proposalSummary}</p>
					<dl>
						<div>
							<dt>Actor</dt>
							<dd>{data.agent.proposalActor}</dd>
						</div>
						<div>
							<dt>Revision</dt>
							<dd>{data.agent.proposalRevision}</dd>
						</div>
						<div>
							<dt>Affected</dt>
							<dd>{data.agent.proposalAffected}</dd>
						</div>
						<div>
							<dt>Result fingerprint</dt>
							<dd class="pending">{data.agent.proposalFingerprint}</dd>
						</div>
					</dl>
				</div>
				<div class="proposal">
					<span class="overline">Session activity</span>
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
					<p class="placeholder-copy">Proposals commit by resubmitting the same operations to /api/studio/transactions/call with the expected revision.</p>
				</div>
				<div class="agent-actions">
					<a class="primary" href="/api/studio/actions">Action catalog ↗</a>
					<a href="/api/studio/manifest">Inspect manifest ↗</a>
				</div>
			</aside>
		</div>
		<section class="telemetry-dock" aria-label="Diagnostics and telemetry">
			<nav aria-label="Diagnostic tabs">
				<button type="button">Console</button>
				<button type="button">Render Graph</button>
				<button type="button">Harness</button>
				<button type="button">Ray Trace</button>
				<button type="button">Selena</button>
				<button type="button">Performance</button>
				<button type="button" class="active">Agent Activity</button>
			</nav>
			<div class="telemetry-content">
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
							{entry.summary}
						</p>
					</Each>
				</div>
				<div class="certification-card">
					<span>Certification</span>
					<strong>{data.certification.status}</strong>
					<small>{data.certification.available} / {data.certification.total} editor dimensions available</small>
					<small>{data.certification.liveChecksPass} / {data.certification.liveChecksTotal} live evidence checks pass · release {data.certification.releaseStatus}</small>
					<ul class="certification-dimensions">
						<Each of={data.certification.dimensions} as="dimension">
							<li title={dimension.evidence}>
								<span>{dimension.id}</span>
								<strong>{dimension.status}</strong>
							</li>
						</Each>
					</ul>
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
		<script src="/studio-gizmo.js" defer></script>
		<script src="/studio-camera.js" defer></script>
		<footer class="status-bar">
			<span class="runtime-label">GoSX server</span>
			<span>Scene revision {data.revision}</span>
			<span>Saved revision {data.project.savedRevision}</span>
			<span class="author-label">Project {data.project.state}</span>
			<span>Scene3D mounted</span>
			<span>GPU telemetry unavailable</span>
			<span class="status-spacer"></span>
			<strong class="pending">CERTIFICATION {data.certification.status}</strong>
			<span>Industrial Void</span>
		</footer>
	</main>
}
