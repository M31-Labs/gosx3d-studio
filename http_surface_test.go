package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"m31labs.dev/gosx3d-studio/internal/studio"
)

// These exercise the router main() actually serves. The package sat at 8.8%
// coverage across 39 routes while its tests asserted on helpers the routes
// might not have called — which is how two mutating routes reached production
// without an authority check and nothing failed. route_authority_test.go holds
// the declared table against the source; this holds the running server against
// the table.

const testActionToken = "test-action-token"

func newTestStudio(t *testing.T) (http.Handler, *studio.Workspace) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(thisFile)

	workspace, err := studio.OpenWorkspace(t.TempDir(), studio.SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	app, err := buildStudioApp(studioConfig{
		root:          root,
		appName:       "GoSX 3D Studio (test)",
		workspace:     workspace,
		actionToken:   testActionToken,
		desktopHost:   false,
		sessionSecret: "a-private-secret-for-tests-only-0123456789",
	})
	if err != nil {
		t.Fatal(err)
	}
	return app.Build(), workspace
}

func doRequest(t *testing.T, handler http.Handler, method, path string, body any, authorize bool) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if authorize {
		request.Header.Set("Authorization", "Bearer "+testActionToken)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

// Every route declaring token authority must actually refuse an anonymous
// caller at the running server, not merely call a helper that would have.
func TestTokenRoutesRefuseAnonymousCallers(t *testing.T) {
	handler, _ := newTestStudio(t)
	checked := 0
	for pattern, authority := range studioRouteAuthority {
		if authority != authorityToken {
			continue
		}
		method, path, _ := strings.Cut(pattern, " ")
		response := doRequest(t, handler, method, path, map[string]any{}, false)
		if response.Code != http.StatusUnauthorized && response.Code != http.StatusForbidden {
			t.Errorf("%s answered an anonymous caller with %d, want 401 or 403", pattern, response.Code)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no token-authority routes were exercised")
	}
	t.Logf("refused anonymous access on %d token-authority routes", checked)
}

// Read routes carry no credential, so they must answer without one. A read
// route that started refusing anonymous callers would break every agent that
// discovers the surface before authenticating.
func TestReadRoutesAnswerWithoutCredentials(t *testing.T) {
	handler, _ := newTestStudio(t)
	checked := 0
	for pattern, authority := range studioRouteAuthority {
		if authority != authorityRead {
			continue
		}
		method, path, _ := strings.Cut(pattern, " ")
		response := doRequest(t, handler, method, path, nil, false)
		if response.Code == http.StatusUnauthorized || response.Code == http.StatusForbidden {
			t.Errorf("%s refused an anonymous read with %d", pattern, response.Code)
		}
		checked++
	}
	t.Logf("answered anonymous reads on %d read-authority routes", checked)
}

func TestRenderedAutomationStatusUsesResolvedStudioConfig(t *testing.T) {
	t.Setenv("GOSX_ENV", "development")
	for _, test := range []struct {
		name        string
		configured  string
		environment string
		want        string
		dontWant    string
	}{
		{
			name:       "private token enables automation",
			configured: "private-render-test-token",
			want:       "bearer automation enabled",
			dontWant:   "automation API disabled",
		},
		{
			name:        "empty token disables automation",
			environment: "environment-must-not-enable-automation",
			want:        "automation API disabled",
			dontWant:    "bearer automation enabled",
		},
		{
			name:        "published placeholder disables automation",
			configured:  "replace-with-a-local-action-token",
			environment: "environment-must-not-enable-automation",
			want:        "automation API disabled",
			dontWant:    "bearer automation enabled",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("STUDIO_ACTION_TOKEN", test.environment)
			workspace, err := studio.OpenWorkspace(t.TempDir(), studio.SampleDocument())
			if err != nil {
				t.Fatal(err)
			}
			app, err := buildStudioApp(studioConfig{
				root:          ".",
				appName:       "GoSX 3D Studio (automation status test)",
				workspace:     workspace,
				actionToken:   test.configured,
				sessionSecret: "a-private-secret-for-tests-only-0123456789",
			})
			if err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Accept", "text/html")
			response := httptest.NewRecorder()
			app.Build().ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("render page = %d: %s", response.Code, response.Body.String())
			}
			body := response.Body.String()
			if !strings.Contains(body, test.want) {
				t.Fatalf("rendered page omitted %q", test.want)
			}
			if strings.Contains(body, test.dontWant) {
				t.Fatalf("rendered page contradicted config with %q", test.dontWant)
			}
			if test.configured != "" && strings.Contains(body, test.configured) {
				t.Fatal("rendered page leaked the configured action token")
			}
			if test.environment != "" && strings.Contains(body, test.environment) {
				t.Fatal("rendered page leaked the unrelated environment token")
			}
		})
	}
}

func TestTransactionRoutesEnforceRevisionAndReportConflict(t *testing.T) {
	handler, workspace := newTestStudio(t)
	document, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	target, _ := studio.FirstPickTarget(document)

	transaction := map[string]any{
		"id": "http-rename", "actor": "agent://http-test", "mode": "direct",
		"expectedRevision": document.Revision,
		"operations":       []map[string]any{{"kind": "rename-entity", "target": string(target), "name": "Renamed Over HTTP"}},
	}
	response := doRequest(t, handler, http.MethodPost, "/api/studio/transactions/call", transaction, true)
	if response.Code != http.StatusOK {
		t.Fatalf("authorized transaction = %d: %s", response.Code, response.Body.String())
	}

	// Replaying the same revision must conflict, not silently apply twice.
	conflict := doRequest(t, handler, http.MethodPost, "/api/studio/transactions/call", transaction, true)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("stale revision = %d, want 409: %s", conflict.Code, conflict.Body.String())
	}

	after, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if after.Entities[target].Name != "Renamed Over HTTP" {
		t.Fatalf("entity name = %q", after.Entities[target].Name)
	}
	if after.Revision != document.Revision+1 {
		t.Fatalf("revision = %d, want exactly one commit", after.Revision)
	}
}

func TestMalformedTransactionIsRejectedAsBadRequest(t *testing.T) {
	handler, _ := newTestStudio(t)
	for name, body := range map[string]string{
		"not json":       "{not json",
		"unknown field":  `{"id":"x","mode":"direct","expectedRevision":1,"operations":[],"surprise":true}`,
		"wrong types":    `{"id":123,"mode":"direct"}`,
		"empty document": ``,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/studio/transactions/call", strings.NewReader(body))
			request.Header.Set("Accept", "application/json")
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+testActionToken)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

// Every handler bounds its body. An unbounded decoder lets one request stream
// arbitrary bytes into the process.
func TestOversizedBodiesAreRejected(t *testing.T) {
	handler, _ := newTestStudio(t)
	oversized := strings.Repeat("a", 2<<20)
	for _, path := range []string{
		"/api/studio/undo",
		"/api/studio/redo",
		"/api/studio/selection",
		"/api/studio/project/open",
	} {
		t.Run(path, func(t *testing.T) {
			body := fmt.Sprintf(`{"actor":%q}`, oversized)
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Accept", "application/json")
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+testActionToken)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code == http.StatusOK {
				t.Fatalf("%s accepted a %d byte body", path, len(body))
			}
		})
	}
}

// The native project dialog has no meaning without a desktop host, so a
// server-only process must not accept a request claiming to come from one.
func TestDesktopOnlyRouteIsClosedWithoutADesktopHost(t *testing.T) {
	handler, _ := newTestStudio(t) // built with desktopHost: false
	request := httptest.NewRequest(http.MethodPost, "/api/studio/project/open-from-desktop", strings.NewReader(`{"path":"/tmp/scene.scene3d","expectedRevision":1}`))
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-GoSX-Desktop-Intent", "native-dialog")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code == http.StatusOK {
		t.Fatal("a server-only process accepted a native-dialog project switch")
	}
}

func TestHealthAndDiscoveryRoutesReturnUsableJSON(t *testing.T) {
	handler, _ := newTestStudio(t)
	for path, key := range map[string]string{
		"/api/health":               "ok",
		"/api/studio/manifest":      "schema",
		"/api/studio/certification": "schema",
		"/api/studio/initialize":    "protocol",
	} {
		t.Run(path, func(t *testing.T) {
			response := doRequest(t, handler, http.MethodGet, path, nil, false)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", response.Code, response.Body.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode: %v (body %s)", err, response.Body.String())
			}
			if _, ok := payload[key]; !ok {
				t.Fatalf("response has no %q field: %v", key, payload)
			}
		})
	}
}

// The evidence suite costs about a second of CPU per run and this route
// carries read authority, so it answers without a credential. A repeat request
// at the same document must be served from cache rather than buying that
// second again.
func TestCertificationEvidenceIsCachedByDocument(t *testing.T) {
	if testing.Short() {
		t.Skip("evidence caching skipped in short mode")
	}
	handler, workspace := newTestStudio(t)

	// The cache lives in internal/studio and an earlier test may have warmed
	// it, so commit an edit first: that guarantees the next request is a miss
	// no matter what ran before.
	commitEdit := func(id, name string) {
		t.Helper()
		document, err := workspace.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		target, _ := studio.FirstPickTarget(document)
		response := doRequest(t, handler, http.MethodPost, "/api/studio/transactions/call", map[string]any{
			"id": id, "actor": "agent://http-test", "mode": "direct",
			"expectedRevision": document.Revision,
			"operations":       []map[string]any{{"kind": "rename-entity", "target": string(target), "name": name}},
		}, true)
		if response.Code != http.StatusOK {
			t.Fatalf("edit %s = %d: %s", id, response.Code, response.Body.String())
		}
	}

	commitEdit("evidence-cache-warm", "Before")
	start := time.Now()
	cold := doRequest(t, handler, http.MethodGet, "/api/studio/certification/evidence", nil, false)
	coldElapsed := time.Since(start)
	if cold.Code != http.StatusOK {
		t.Fatalf("cold request = %d: %s", cold.Code, cold.Body.String())
	}

	start = time.Now()
	warm := doRequest(t, handler, http.MethodGet, "/api/studio/certification/evidence", nil, false)
	warmElapsed := time.Since(start)
	if warm.Code != http.StatusOK {
		t.Fatalf("warm request = %d", warm.Code)
	}
	if !bytes.Equal(cold.Body.Bytes(), warm.Body.Bytes()) {
		t.Fatal("cached evidence differs from the run it caches")
	}
	// The suite is hundreds of milliseconds; a cache hit is microseconds.
	if warmElapsed > coldElapsed/10 {
		t.Fatalf("repeat request took %v against a first request of %v: it re-ran the suite", warmElapsed, coldElapsed)
	}

	// A committed edit must invalidate it, or the report would describe a
	// document that no longer exists.
	commitEdit("evidence-cache-invalidate", "After")
	fresh := doRequest(t, handler, http.MethodGet, "/api/studio/certification/evidence", nil, false)
	if fresh.Code != http.StatusOK {
		t.Fatalf("post-edit request = %d", fresh.Code)
	}
	if bytes.Equal(fresh.Body.Bytes(), cold.Body.Bytes()) {
		t.Fatal("evidence did not change after an edit: the cache is not keyed on the document")
	}
}
