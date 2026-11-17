// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package auth

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/build"
	"github.com/larksuite/cli/internal/core"
)

func enableDocumentPreset(t *testing.T) {
	t.Helper()
	old := build.PermissionPreset
	build.PermissionPreset = "docs7-managed"
	t.Cleanup(func() { build.PermissionPreset = old })
}

func TestDocumentRegistrationRequestUsesMinimalSDKPage(t *testing.T) {
	enableDocumentPreset(t)
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Method != http.MethodPost || r.Form.Get("action") != "begin" || r.Form.Get("archetype") != "PersonalAgent" {
			t.Fatalf("unexpected registration request: %s %v", r.Method, r.Form)
		}
		return jsonResponse(`{"device_code":"example-device","user_code":"example-code","verification_uri_complete":"https://open.feishu.cn/page/openclaw?user_code=example-code&clientID=cli_example&custom=preserved","expire_in":60}`), nil
	})}
	response, err := RequestAppRegistration(context.Background(), client, core.BrandFeishu, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(BuildVerificationURL(response.VerificationUriComplete, "1.0.100-docs7.2"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Path != "/page/openclaw" || q.Get("user_code") != "example-code" || q.Get("custom") != "preserved" || q.Get("from") != "sdk" || q.Get("tp") != "sdk" || q.Get("createOnly") != "true" || q.Has("clientID") {
		t.Fatalf("unsafe confirmation URL: %s", u)
	}
	compressed, err := base64.RawURLEncoding.DecodeString(q.Get("addons"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var payload struct {
		Preset *bool `json:"preset"`
		Scopes struct {
			Tenant []string `json:"tenant"`
			User   []string `json:"user"`
		} `json:"scopes"`
	}
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	want := []string{"docs:document.content:read", "docs:document.media:download", "docx:document:readonly", "drive:drive:readonly", "offline_access", "space:document:retrieve", "wiki:wiki:readonly"}
	if payload.Preset == nil || *payload.Preset || !slices.Equal(payload.Scopes.Tenant, []string{"application:application:patch"}) || !slices.Equal(payload.Scopes.User, want) {
		t.Fatalf("wrong permission payload: %+v", payload)
	}
}

func TestDocumentRegistrationRejectsUnsafeOrMissingPage(t *testing.T) {
	enableDocumentPreset(t)
	for _, raw := range []string{"", "http://open.feishu.cn/page/openclaw?user_code=example", "https://open.feishu.cn/page/cli?user_code=example", "https://example.com/page/openclaw?user_code=example", "https://open.feishu.cn/page/openclaw"} {
		t.Run(raw, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{"device_code": "example", "verification_uri_complete": raw})
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(string(body)), nil })}
			got, err := RequestAppRegistration(context.Background(), client, core.BrandFeishu, io.Discard)
			if err == nil || got != nil || !errs.IsTyped(err) {
				t.Fatalf("unsafe URL accepted: response=%v error=%v", got, err)
			}
		})
	}
}

func TestDocumentDeviceAuthorizationRejectsExpansionBeforeHTTP(t *testing.T) {
	enableDocumentPreset(t)
	for _, scope := range []string{"", "im:message", "docs:document.content:read application:application:patch", "offline_access contact:user.basic_profile:readonly"} {
		called := false
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { called = true; return jsonResponse(`{}`), nil })}
		_, err := RequestDeviceAuthorization(context.Background(), client, "cli_example", "example-secret", core.BrandFeishu, scope, io.Discard)
		if err == nil || called || !errs.IsTyped(err) {
			t.Fatalf("scope expansion not blocked: %q error=%v HTTP=%v", scope, err, called)
		}
	}
}

func TestDocumentDeviceAuthorizationSendsOnlySevenScopes(t *testing.T) {
	enableDocumentPreset(t)
	want := strings.Join(DocumentUserScopes(), " ")
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("scope") != want {
			t.Fatalf("unexpected wire scopes: %q", r.Form.Get("scope"))
		}
		return jsonResponse(`{"device_code":"example-device","verification_uri_complete":"https://open.feishu.cn/example","expires_in":60}`), nil
	})}
	if _, err := RequestDeviceAuthorization(context.Background(), client, "cli_example", "example-secret", core.BrandFeishu, want, io.Discard); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentTokenDoesNotAcceptExtraOrUnverifiableScopes(t *testing.T) {
	enableDocumentPreset(t)
	for _, scope := range []string{"", "offline_access im:message"} {
		body, _ := json.Marshal(map[string]any{"access_token": "EXAMPLE_TOKEN", "token_type": "Bearer", "scope": scope})
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(string(body)), nil })}
		result, err := PollDeviceToken(context.Background(), client, "cli_example", "example-secret", core.BrandFeishu, "example-device", 1, 10, io.Discard)
		if err != nil || result.OK || result.Token != nil || result.Err == nil || result.Error != "scope_outside_preset" {
			t.Fatalf("unexpected token result: %+v error=%v", result, err)
		}
	}
}

func TestDocumentTokenAcceptsSevenScopesWithoutRewriting(t *testing.T) {
	enableDocumentPreset(t)
	scope := strings.Join(DocumentUserScopes(), " ")
	body, _ := json.Marshal(map[string]any{"access_token": "EXAMPLE_TOKEN", "refresh_token": "EXAMPLE_REFRESH_TOKEN", "token_type": "Bearer", "scope": scope})
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(string(body)), nil })}
	result, err := PollDeviceToken(context.Background(), client, "cli_example", "example-secret", core.BrandFeishu, "example-device", 1, 10, io.Discard)
	if err != nil || !result.OK || result.Token == nil || result.Token.Scope != scope {
		t.Fatalf("allowed token rejected or scope rewritten: result=%+v error=%v", result, err)
	}
}
