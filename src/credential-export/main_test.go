// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/core"
)

type fakeKeychain struct{ value string }

func (f fakeKeychain) Get(_, _ string) (string, error) { return f.value, nil }
func (f fakeKeychain) Set(_, _, _ string) error        { return nil }
func (f fakeKeychain) Remove(_, _ string) error        { return nil }

func TestExportCredentialWritesOnlySelectedAppToNewFile(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	config := &core.MultiAppConfig{Apps: []core.AppConfig{
		{Name: "other", AppId: "cli_other"},
		{Name: "chosen", AppId: "cli_example", AppSecret: core.SecretInput{Ref: &core.SecretRef{Source: "keychain", ID: "appsecret:cli_example"}}},
	}}
	const secret = "EXAMPLE_TEST_SECRET_ONLY"
	opts := options{appID: "cli_example", profile: "chosen", output: "delivery.json", redirectURL: defaultRedirect}
	var stdout bytes.Buffer
	if err := exportCredential(config, fakeKeychain{secret}, opts, &stdout); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), secret) {
		t.Fatal("secret leaked to stdout")
	}
	data, err := os.ReadFile("delivery.json")
	if err != nil {
		t.Fatal(err)
	}
	var result credentialDocument
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.AppID != opts.appID || result.AppSecret != secret || result.RedirectURL != defaultRedirect {
		t.Fatal("wrong delivery values")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat("delivery.json")
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0077 != 0 {
			t.Fatal("delivery file grants group or other access")
		}
	}
	if err := exportCredential(config, fakeKeychain{"different"}, opts, &stdout); err == nil {
		t.Fatal("existing delivery file was overwritten")
	}
	again, err := os.ReadFile("delivery.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again) {
		t.Fatal("existing delivery file changed")
	}
}

func TestExportCredentialRejectsWrongAppAndUnsafePaths(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	config := &core.MultiAppConfig{Apps: []core.AppConfig{{Name: "chosen", AppId: "cli_example", AppSecret: core.PlainSecret("EXAMPLE_TEST_SECRET_ONLY")}}}
	for _, opts := range []options{
		{appID: "cli_wrong", profile: "chosen", output: "delivery.json", redirectURL: defaultRedirect},
		{appID: "cli_example", profile: "chosen", output: filepath.Join("..", "escape.json"), redirectURL: defaultRedirect},
		{appID: "cli_example", profile: "chosen", output: "delivery.json", redirectURL: "file:///tmp/example"},
	} {
		var out bytes.Buffer
		if err := exportCredential(config, fakeKeychain{}, opts, &out); err == nil {
			t.Fatalf("unsafe request accepted: %+v", opts)
		}
		if out.Len() != 0 {
			t.Fatal("rejected request produced output")
		}
	}
	if _, err := os.Stat("delivery.json"); !os.IsNotExist(err) {
		t.Fatal("rejected export created a file")
	}
}
