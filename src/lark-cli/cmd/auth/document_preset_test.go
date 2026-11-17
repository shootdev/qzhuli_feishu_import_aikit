// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	larkauth "github.com/larksuite/cli/internal/auth"
	"github.com/larksuite/cli/internal/build"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/httpmock"
)

func enableDocumentPreset(t *testing.T) {
	t.Helper()
	old := build.PermissionPreset
	build.PermissionPreset = "docs7-managed"
	t.Cleanup(func() { build.PermissionPreset = old })
}

func TestDocumentLoginDefaultReachesWireWithSevenScopes(t *testing.T) {
	enableDocumentPreset(t)
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, registry := cmdutil.TestFactory(t, &core.CliConfig{ProfileName: "example", AppID: "cli_example", AppSecret: "example-secret", Brand: core.BrandFeishu})
	stub := &httpmock.Stub{Method: "POST", URL: larkauth.PathDeviceAuthorization, Body: map[string]any{
		"device_code": "example-device", "verification_uri_complete": "https://open.feishu.cn/example", "expires_in": 60,
	}}
	registry.Register(stub)
	oldFetch := fetchRemoteScopes
	fetchRemoteScopes = func(core.LarkBrand) (map[string][]string, bool) {
		t.Fatal("docs-only login fetched the broad scope catalog")
		return nil, false
	}
	t.Cleanup(func() { fetchRemoteScopes = oldFetch })
	if err := authLoginRun(&LoginOptions{Factory: f, Ctx: context.Background(), NoWait: true, JSON: true}, builtinResolver(t)); err != nil {
		t.Fatal(err)
	}
	form, err := url.ParseQuery(string(stub.CapturedBody))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := form.Get("scope"), strings.Join(larkauth.DocumentUserScopes(), " "); got != want {
		t.Fatalf("wire scopes=%q want=%q", got, want)
	}
	if cached, err := loadLoginRequestedScope("example-device"); err != nil || cached != form.Get("scope") {
		t.Fatalf("scope cache mismatch: %q %v", cached, err)
	}
}

func TestDocumentLoginRejectsBroadOptionsBeforeCredentials(t *testing.T) {
	enableDocumentPreset(t)
	for _, opts := range []*LoginOptions{{Recommend: true}, {Domains: []string{"all"}}, {Exclude: []string{"im:message"}}, {Scope: "application:application:patch"}} {
		// A nil Factory would panic if execution proceeded to credentials or HTTP.
		err := authLoginRun(opts, domainResolver{})
		problem, ok := errs.ProblemOf(err)
		var validation *errs.ValidationError
		if !ok || problem.Category != errs.CategoryValidation || !errors.As(err, &validation) || validation.Param == "" {
			t.Fatalf("missing typed rejection: %v", err)
		}
	}
}

func TestDocumentLoginResumeRequiresAllowedRecordedScopes(t *testing.T) {
	enableDocumentPreset(t)
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	opts := &LoginOptions{DeviceCode: "example-device"}
	if err := applyDocumentLoginPreset(opts); err == nil {
		t.Fatal("unrecorded authorization accepted")
	}
	if err := saveLoginRequestedScope(opts.DeviceCode, "im:message"); err != nil {
		t.Fatal(err)
	}
	if err := applyDocumentLoginPreset(opts); err == nil {
		t.Fatal("broad authorization resumed")
	}
	if err := saveLoginRequestedScope(opts.DeviceCode, strings.Join(larkauth.DocumentUserScopes(), " ")); err != nil {
		t.Fatal(err)
	}
	if err := applyDocumentLoginPreset(opts); err != nil {
		t.Fatal(err)
	}
}
