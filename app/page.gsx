package app

func Page() Node {
	return <main class="studio-shell" aria-label="GoSX 3D Studio scaffold">
		<header class="menu-bar">
			<a href="/" data-gosx-link class="brand" aria-label="GoSX 3D Studio home">
				<span class="brand-mark">GoSX</span>
				<span>3D Studio</span>
			</a>
			<nav class="application-menu" aria-label="Application menu">
				<button type="button">File</button>
				<button type="button">Edit</button>
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
				Scaffold
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
				<button type="button" class="tool active" aria-label="Select">↖</button>
				<button type="button" class="tool" aria-label="Move">✣</button>
				<button type="button" class="tool" aria-label="Rotate">↻</button>
				<button type="button" class="tool" aria-label="Scale">↗</button>
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
					<button type="button" aria-label="Add asset">＋</button>
				</header>
				<label class="search-field">
					<span class="sr-only">Search assets</span>
					<input type="search" placeholder="Search assets…"></input>
				</label>
				<ul class="tree asset-tree">
					<li class="expanded">
						<span>▾</span>
						<strong>Chinese Checkers</strong>
					</li>
					<li class="depth-1 expanded">
						<span>▾</span>
						Scenes
					</li>
					<li class="depth-2 selected">
						<span>◇</span>
						main.scene
					</li>
					<li class="depth-1">
						<span>▸</span>
						Models
						<small>3</small>
					</li>
					<li class="depth-1">
						<span>▸</span>
						Materials
						<small>6</small>
					</li>
					<li class="depth-1">
						<span>▸</span>
						Textures
						<small>4</small>
					</li>
					<li class="depth-1">
						<span>▸</span>
						Rigs
						<small>0</small>
					</li>
					<li class="depth-1">
						<span>▸</span>
						Animations
						<small>0</small>
					</li>
				</ul>
			</aside>
			<aside class="panel hierarchy-panel" aria-labelledby="hierarchy-title">
				<header class="panel-heading">
					<h2 id="hierarchy-title">Scene Hierarchy</h2>
					<span class="panel-count">12</span>
				</header>
				<label class="search-field">
					<span class="sr-only">Search hierarchy</span>
					<input type="search" placeholder="Search hierarchy…"></input>
				</label>
				<ul class="tree hierarchy-tree">
					<li>
						<code>0000</code>
						<strong>Scene Root</strong>
					</li>
					<li class="depth-1">
						<code>0100</code>
						Environment
					</li>
					<li class="depth-1">
						<code>0110</code>
						Camera Main
					</li>
					<li class="depth-1">
						<code>0200</code>
						Lighting
					</li>
					<li class="depth-1 expanded">
						<code>0300</code>
						<span>▾</span>
						Board Group
					</li>
					<li class="depth-2">
						<code>0310</code>
						Board Mesh
					</li>
					<li class="depth-2 selected">
						<code>0421</code>
						<span class="material-dot jade"></span>
						Piece 01
					</li>
					<li class="depth-2">
						<code>0422</code>
						<span class="material-dot jade"></span>
						Piece 02
					</li>
					<li class="depth-2">
						<code>0521</code>
						<span class="material-dot blue"></span>
						Piece 11
					</li>
					<li class="depth-2">
						<code>0621</code>
						<span class="material-dot gold"></span>
						Piece 21
					</li>
				</ul>
			</aside>
			<section class="viewport-panel" aria-labelledby="viewport-title">
				<header class="viewport-header">
					<h2 id="viewport-title">Viewport</h2>
					<div>
						<button type="button">Perspective⌄</button>
						<button type="button">Lit⌄</button>
					</div>
				</header>
				<div class="scene-stage">
					<div class="runtime-readout" aria-label="Runtime scaffold telemetry">
						<span>SCENE IR</span>
						<strong>scaffold</strong>
						<span>BACKEND</span>
						<strong>unmounted</strong>
						<span>REVISION</span>
						<strong>0001</strong>
					</div>
					<div class="board-placeholder" aria-label="Scene3D viewport placeholder">
						<div class="board-grid"></div>
						<div class="piece p1 jade"></div>
						<div class="piece p2 jade"></div>
						<div class="piece p3 orange"></div>
						<div class="piece p4 blue"></div>
						<div class="piece selected-piece jade">
							<span class="gizmo-axis axis-x"></span>
							<span class="gizmo-axis axis-y"></span>
						</div>
					</div>
					<div class="viewport-empty-state">
						<span class="overline">Scene3D mount seam</span>
						<h1>{data.appName}</h1>
						<p>
							The shell is live. SceneDoc → SceneIR → native evidence is the first implementation slice.
						</p>
					</div>
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
					<span class="selection-id">Piece 01 · 0421</span>
				</header>
				<details open>
					<summary>Transform</summary>
					<div class="property-grid">
						<span>Location</span>
						<label>
							X
							<input value="2.400" readonly></input>
						</label>
						<label>
							Y
							<input value="0.420" readonly></input>
						</label>
						<label>
							Z
							<input value="-1.200" readonly></input>
						</label>
					</div>
					<div class="property-grid">
						<span>Rotation</span>
						<label>
							X
							<input value="0.000" readonly></input>
						</label>
						<label>
							Y
							<input value="0.000" readonly></input>
						</label>
						<label>
							Z
							<input value="0.000" readonly></input>
						</label>
					</div>
				</details>
				<details open>
					<summary>Selena Material</summary>
					<div class="material-preview jade"></div>
					<dl class="properties">
						<div>
							<dt>Material</dt>
							<dd>jade.selena</dd>
						</div>
						<div>
							<dt>Surface</dt>
							<dd>PBR / Selena</dd>
						</div>
						<div>
							<dt>Transmission</dt>
							<dd class="authored">0.22</dd>
						</div>
						<div>
							<dt>Roughness</dt>
							<dd>0.28</dd>
						</div>
					</dl>
				</details>
				<details>
					<summary>Physics</summary>
					<p class="placeholder-copy">Component surface reserved.</p>
				</details>
				<details>
					<summary>Metadata</summary>
					<p class="placeholder-copy">Stable ID · revision 0001</p>
				</details>
			</aside>
			<section class="timeline-panel" aria-labelledby="timeline-title">
				<header class="panel-heading">
					<h2 id="timeline-title">Timeline</h2>
					<span class="panel-note">Scaffold tracks</span>
				</header>
				<div class="timeline-ruler">
					<span>0</span>
					<span>24</span>
					<span>48</span>
					<span>72</span>
					<span>96</span>
					<span>120</span>
				</div>
				<div class="track">
					<strong>Transform</strong>
					<div class="track-line">
						<i></i>
						<i></i>
						<i></i>
					</div>
				</div>
				<div class="track">
					<strong>Agent Events</strong>
					<div class="track-line runtime">
						<i></i>
						<i></i>
					</div>
				</div>
			</section>
			<aside class="agent-panel" aria-labelledby="agent-title">
				<header class="panel-heading">
					<h2 id="agent-title">Agent Actions</h2>
					<span class="runtime-label">typed surface</span>
				</header>
				<div class="authority-tabs">
					<button type="button">Read</button>
					<button type="button" class="active">Propose</button>
					<button type="button" disabled>Trusted Direct</button>
				</div>
				<div class="proposal">
					<span class="overline">Proposed transaction</span>
					<p>
						Assign jade material to selected pieces, preserve roughness, then certify native output.
					</p>
					<dl>
						<div>
							<dt>Revision</dt>
							<dd>0001</dd>
						</div>
						<div>
							<dt>Entities</dt>
							<dd>10 pieces</dd>
						</div>
						<div>
							<dt>Evidence</dt>
							<dd class="pending">Not run</dd>
						</div>
					</dl>
				</div>
				<div class="agent-actions">
					<button type="button" class="primary" disabled>Apply atomically</button>
					<button type="button">Inspect manifest</button>
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
						<time>00:00:01</time>
						<span class="runtime-label">SYSTEM</span>
						Studio scaffold manifest loaded.
					</p>
					<p>
						<time>00:00:02</time>
						<span class="author-label">AUTHOR</span>
						Scene3D mount awaiting first vertical slice.
					</p>
				</div>
				<div class="certification-card">
					<span>Certification</span>
					<strong>NOT RUN</strong>
					<small>Honest scaffold state</small>
				</div>
				<div class="manifest-link">
					<span>Agent-readable contract</span>
					<a href="/api/studio/manifest">/api/studio/manifest ↗</a>
				</div>
			</div>
		</section>
		<footer class="status-bar">
			<span class="runtime-label">GoSX server</span>
			<span>Scene revision 0001</span>
			<span>Renderer unmounted</span>
			<span>0 draw calls</span>
			<span>0 MB GPU</span>
			<span class="status-spacer"></span>
			<strong class="pending">UNCERTIFIED</strong>
			<span>Industrial Void</span>
		</footer>
	</main>
}
