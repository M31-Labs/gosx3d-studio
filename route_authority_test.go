package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Authority used to live only in whether a handler happened to call
// authorizeAction. Two mutating routes reached production without it, and
// nothing failed. These tests read the routes GoSX actually registers out of
// main.go and hold them against studioRouteAuthority, so a route added
// without a declared authority is a build-breaking omission rather than a
// silent one.
//
// This is the same discipline as TestComponentCatalogCoversEverydSerializedStructField
// in internal/studio: derive the check from the source of truth, never from a
// hand-kept second list.

// registeredRoute is one app.API("METHOD /path", handler) call.
type registeredRoute struct {
	pattern            string
	callsAuthorizeName string // authorizeAction or the helper that calls it
	line               int
}

func parseRegisteredRoutes(t *testing.T) []registeredRoute {
	t.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "main.go", nil, 0)
	if err != nil {
		t.Fatalf("parse main.go: %v", err)
	}

	// Helpers that perform the token check on a handler's behalf.
	authorizingHelpers := map[string]bool{"authorizeAction": true, "historyAction": true}

	var routes []registeredRoute
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "API" {
			return true
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		pattern, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatalf("route pattern %s: %v", literal.Value, err)
		}
		route := registeredRoute{pattern: pattern, line: fileSet.Position(literal.Pos()).Line}
		ast.Inspect(call.Args[1], func(inner ast.Node) bool {
			innerCall, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := innerCall.Fun.(*ast.Ident)
			if ok && authorizingHelpers[identifier.Name] {
				route.callsAuthorizeName = identifier.Name
				return false
			}
			return true
		})
		routes = append(routes, route)
		return true
	})
	if len(routes) == 0 {
		t.Fatal("found no app.API routes in main.go; the parser no longer matches how routes are registered")
	}
	return routes
}

func TestEveryStudioRouteDeclaresAuthority(t *testing.T) {
	routes := parseRegisteredRoutes(t)
	declared := make(map[string]bool, len(studioRouteAuthority))
	for pattern := range studioRouteAuthority {
		declared[pattern] = true
	}

	for _, route := range routes {
		if _, ok := studioRouteAuthority[route.pattern]; !ok {
			t.Errorf("main.go:%d registers %q with no entry in studioRouteAuthority", route.line, route.pattern)
			continue
		}
		delete(declared, route.pattern)
	}
	stale := make([]string, 0, len(declared))
	for pattern := range declared {
		stale = append(stale, pattern)
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("studioRouteAuthority declares routes that are not registered: %v", stale)
	}
}

func TestTokenAuthorityRoutesCheckTheActionToken(t *testing.T) {
	for _, route := range parseRegisteredRoutes(t) {
		authority, ok := studioRouteAuthority[route.pattern]
		if !ok {
			continue // reported by TestEveryStudioRouteDeclaresAuthority
		}
		switch authority {
		case authorityToken:
			if route.callsAuthorizeName == "" {
				t.Errorf("main.go:%d %q declares token authority but never checks the action token", route.line, route.pattern)
			}
		case authorityRead, authoritySession:
			if route.callsAuthorizeName != "" {
				t.Errorf("main.go:%d %q declares %s authority but checks the action token via %s", route.line, route.pattern, authority, route.callsAuthorizeName)
			}
		}
	}
}

// Session cookies and CSRF tokens both derive from SESSION_SECRET, so a
// published default would let anyone mint the CSRF token that guards every
// browser-authority route.
func TestSessionSecretRefusesPublishedPlaceholders(t *testing.T) {
	t.Setenv("GOSX_ENV", "development")
	for _, placeholder := range []string{"", "  ", "gosx-app-session-secret", "replace-with-a-local-development-secret"} {
		secret, err := resolveSessionSecret(placeholder)
		if err != nil {
			t.Fatalf("development fallback for %q: %v", placeholder, err)
		}
		if secret == strings.TrimSpace(placeholder) || sharedSecretPlaceholders[secret] {
			t.Fatalf("resolved secret for %q is still the placeholder", placeholder)
		}
		if len(secret) < 32 {
			t.Fatalf("generated development secret is %d chars", len(secret))
		}
	}

	t.Setenv("GOSX_ENV", "production")
	for _, placeholder := range []string{"", "gosx-app-session-secret", "replace-with-a-local-development-secret"} {
		if _, err := resolveSessionSecret(placeholder); err == nil {
			t.Fatalf("production accepted %q as a session secret", placeholder)
		}
	}
	configured, err := resolveSessionSecret("a-real-private-production-secret")
	if err != nil || configured != "a-real-private-production-secret" {
		t.Fatalf("configured production secret = %q, %v", configured, err)
	}
}

func TestBearerMatchIsExactAndRejectsAnEmptyToken(t *testing.T) {
	if !bearerMatches("Bearer secret", "secret") {
		t.Fatal("exact bearer credential was rejected")
	}
	for _, header := range []string{"", "Bearer", "Bearer ", "Bearer secre", "Bearer secrets", "bearer secret", "Basic secret"} {
		if bearerMatches(header, "secret") {
			t.Fatalf("header %q was accepted", header)
		}
	}
	// An unset token must never authenticate, including against an empty or
	// bare-prefix header.
	for _, header := range []string{"", "Bearer ", "Bearer x"} {
		if bearerMatches(header, "") {
			t.Fatalf("header %q was accepted with no configured token", header)
		}
	}
	if bearerMatches("Bearer replace-with-a-local-action-token", "replace-with-a-local-action-token") {
		t.Fatal("published action-token placeholder authenticated its matching bearer header")
	}
}

func TestActionTokenRefusesThePublishedPlaceholder(t *testing.T) {
	t.Setenv("GOSX_ENV", "development")
	resolved, err := resolveActionToken(" replace-with-a-local-action-token ")
	if err != nil {
		t.Fatalf("development placeholder: %v", err)
	}
	if resolved != "" {
		t.Fatalf("development placeholder resolved to %q, want disabled", resolved)
	}

	t.Setenv("GOSX_ENV", "production")
	if _, err := resolveActionToken("replace-with-a-local-action-token"); err == nil {
		t.Fatal("production accepted the published action-token placeholder")
	}
	resolved, err = resolveActionToken("a-private-action-token")
	if err != nil || resolved != "a-private-action-token" {
		t.Fatalf("private production action token = %q, %v", resolved, err)
	}
}

// A read route must not mutate. GoSX matches on method, so a read declaration
// on anything but GET is a declaration error.
func TestReadAuthorityIsGetOnly(t *testing.T) {
	for pattern, authority := range studioRouteAuthority {
		method, _, found := strings.Cut(pattern, " ")
		if !found {
			t.Fatalf("route pattern %q has no method", pattern)
		}
		if authority == authorityRead && method != "GET" {
			t.Errorf("%q declares read authority on %s; only GET may be unauthenticated", pattern, method)
		}
		if method != "GET" && authority == authorityRead {
			continue
		}
		if method == "POST" && authority != authorityToken && authority != authoritySession {
			t.Errorf("%q is a POST with authority %q; mutating routes need session or token authority", pattern, authority)
		}
	}
}
