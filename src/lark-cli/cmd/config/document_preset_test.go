// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/build"
	"github.com/larksuite/cli/internal/cmdutil"
)

func TestDocumentRegistrationDryRunDoesNotNeedCredentials(t *testing.T) {
	old := build.PermissionPreset
	build.PermissionPreset = "docs7-managed"
	t.Cleanup(func() { build.PermissionPreset = old })
	f, stdout, _, _ := cmdutil.TestFactory(t, nil)
	cmd := NewCmdConfigInit(f, nil)
	cmd.SetArgs([]string{"--new", "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		DryRun     bool `json:"dry_run"`
		CreateOnly bool `json:"create_only"`
		Addons     struct {
			Preset *bool `json:"preset"`
			Scopes struct {
				Tenant []string `json:"tenant"`
				User   []string `json:"user"`
			} `json:"scopes"`
		} `json:"addons"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.DryRun || !result.CreateOnly || result.Addons.Preset == nil || *result.Addons.Preset || len(result.Addons.Scopes.Tenant) != 1 || result.Addons.Scopes.Tenant[0] != "application:application:patch" || len(result.Addons.Scopes.User) != 7 {
		t.Fatalf("unsafe dry-run: %s", stdout.String())
	}
	ProjectInitHelp(cmd, false)
	if !strings.Contains(cmd.Long, "exactly seven document user scopes") {
		t.Fatal("projected help lost permission policy")
	}
}
