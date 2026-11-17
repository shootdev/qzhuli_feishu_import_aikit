// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package config

import (
	"github.com/larksuite/cli/errs"
	larkauth "github.com/larksuite/cli/internal/auth"
	"github.com/larksuite/cli/internal/output"
)

func documentRegistrationHelp() string {
	if !larkauth.DocumentsReadOnlyEnabled() {
		return ""
	}
	return "\n\nPermission preset: docs7-managed. New apps use preset=false, one tenant scope\n(application:application:patch), and exactly seven document user scopes.\nUser login still permits only those seven user scopes. There is no broad-template fallback.\nUse the dry-run configuration/publication defaults with lark-cli api --as bot\nto configure this app and submit publication. Tenant approval remains separate."
}

type documentConfigurationDefaults struct {
	Security struct {
		Add struct {
			RedirectURLs []string `json:"redirect_urls"`
		} `json:"add"`
		AllowRefreshToken bool `json:"allow_refresh_token"`
	} `json:"security"`
	Visibility struct {
		IsVisibleToAll bool `json:"is_visible_to_all"`
	} `json:"visibility"`
}

type documentPublicationDefaults struct {
	MobileDefaultAbility string `json:"mobile_default_ability"`
	PCDefaultAbility     string `json:"pc_default_ability"`
	Remark               string `json:"remark"`
	Changelog            string `json:"changelog"`
}

func defaultDocumentConfiguration() documentConfigurationDefaults {
	var config documentConfigurationDefaults
	config.Security.Add.RedirectURLs = []string{"http://127.0.0.1:4366/oauth/callback"}
	config.Security.AllowRefreshToken = true
	config.Visibility.IsVisibleToAll = true
	return config
}

func defaultDocumentPublication() documentPublicationDefaults {
	return documentPublicationDefaults{
		MobileDefaultAbility: "bot", PCDefaultAbility: "bot",
		Remark:    "Enable employee-authorized document import for all employees.",
		Changelog: "Seven document user scopes; one application scope for self-management; fixed OAuth redirect and all-employee availability.",
	}
}

func previewDocumentRegistration(opts *ConfigInitOptions) error {
	if !larkauth.DocumentsReadOnlyEnabled() || !opts.New || opts.AppID != "" || opts.AppSecretStdin {
		return errs.NewValidationError(errs.SubtypeInvalidArgument,
			"--dry-run requires config init --new in a docs7-managed build, without existing-app credentials").WithParam("--dry-run")
	}
	output.PrintJson(opts.Factory.IOStreams.Out, struct {
		DryRun           bool                          `json:"dry_run"`
		PermissionPreset string                        `json:"permission_preset"`
		CreateOnly       bool                          `json:"create_only"`
		Addons           larkauth.DocumentAppAddons    `json:"addons"`
		Configuration    documentConfigurationDefaults `json:"configuration_defaults"`
		Publication      documentPublicationDefaults   `json:"publication_defaults"`
	}{
		DryRun: true, PermissionPreset: "docs7-managed", CreateOnly: true,
		Addons:        larkauth.DocumentRegistrationAddons(),
		Configuration: defaultDocumentConfiguration(), Publication: defaultDocumentPublication(),
	})
	return nil
}
