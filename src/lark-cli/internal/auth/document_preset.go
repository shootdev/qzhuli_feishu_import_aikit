// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package auth

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"slices"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/build"
	"github.com/larksuite/cli/internal/core"
)

// DocumentsReadOnlyEnabled is a build-time distribution policy, not an OAuth scope.
func DocumentsReadOnlyEnabled() bool {
	return build.PermissionPreset == "docs7-managed"
}

// DocumentUserScopes returns a fresh copy of the seven user-approved scopes.
func DocumentUserScopes() []string {
	return []string{
		"docs:document.content:read",
		"docs:document.media:download",
		"docx:document:readonly",
		"drive:drive:readonly",
		"offline_access",
		"space:document:retrieve",
		"wiki:wiki:readonly",
	}
}

// DocumentAppAddons follows the official registration SDK's addons wire format.
// Preset must be serialized even when false; omitting it restores broad defaults.
type DocumentAppAddons struct {
	Preset bool `json:"preset"`
	Scopes struct {
		Tenant []string `json:"tenant"`
		User   []string `json:"user"`
	} `json:"scopes"`
}

func DocumentRegistrationAddons() DocumentAppAddons {
	var addons DocumentAppAddons
	addons.Scopes.Tenant = []string{"application:application:patch"}
	addons.Scopes.User = DocumentUserScopes()
	return addons
}

// ValidateDocumentScope rejects expansion before any authorization request.
// It also validates granted scopes without silently removing unexpected claims.
func ValidateDocumentScope(scope string) error {
	if strings.TrimSpace(scope) == "" {
		return errs.NewValidationError(errs.SubtypeInvalidArgument, "docs7-managed requires a non-empty scope list").WithParam("--scope")
	}
	allowed := DocumentUserScopes()
	for _, item := range strings.Fields(scope) {
		if !slices.Contains(allowed, item) {
			return errs.NewValidationError(errs.SubtypeInvalidArgument,
				"docs7-managed does not permit scope %q", item).WithParam("--scope")
		}
	}
	return nil
}

// documentRegistrationURL uses the server's SDK confirmation URL. /page/cli
// does not consume addons, so falling back to that page could grant extra scopes.
func documentRegistrationURL(raw string, brand core.LarkBrand) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || raw == "" || u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.Path == "/page/cli" {
		return "", errs.NewConfigError(errs.SubtypeInvalidConfig, "docs7-managed registration requires a valid SDK verification URL; no default-template fallback is allowed")
	}
	ep := core.ResolveEndpoints(brand)
	openURL, openErr := url.Parse(ep.Open)
	accountsURL, accountsErr := url.Parse(ep.Accounts)
	if openErr != nil || accountsErr != nil || (u.Host != openURL.Host && u.Host != accountsURL.Host) {
		return "", errs.NewConfigError(errs.SubtypeInvalidConfig, "docs7-managed registration returned an unexpected verification host")
	}
	query := u.Query()
	if query.Get("user_code") == "" {
		return "", errs.NewConfigError(errs.SubtypeInvalidConfig, "docs7-managed registration returned a verification URL without user_code")
	}
	body, err := json.Marshal(DocumentRegistrationAddons())
	if err != nil {
		return "", errs.NewInternalError(errs.SubtypeSDKError, "encode document registration config: %v", err).WithCause(err)
	}
	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	if _, err := w.Write(body); err != nil {
		_ = w.Close()
		return "", errs.NewInternalError(errs.SubtypeSDKError, "compress document registration config: %v", err).WithCause(err)
	}
	if err := w.Close(); err != nil {
		return "", errs.NewInternalError(errs.SubtypeSDKError, "finish document registration config: %v", err).WithCause(err)
	}
	query.Set("from", "sdk")
	query.Set("tp", "sdk")
	query.Set("source", "lark-cli/docs7-managed")
	query.Set("createOnly", "true")
	query.Del("clientID")
	query.Set("addons", base64.RawURLEncoding.EncodeToString(compressed.Bytes()))
	u.RawQuery = query.Encode()
	return u.String(), nil
}
