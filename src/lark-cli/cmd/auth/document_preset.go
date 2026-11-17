// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package auth

import (
	"strings"

	"github.com/larksuite/cli/errs"
	larkauth "github.com/larksuite/cli/internal/auth"
)

func applyDocumentLoginPreset(opts *LoginOptions) error {
	for flag, supplied := range map[string]bool{
		"--domain": len(opts.Domains) > 0, "--recommend": opts.Recommend, "--exclude": len(opts.Exclude) > 0,
	} {
		if supplied {
			return errs.NewValidationError(errs.SubtypeInvalidArgument,
				"%s is disabled in the docs7-managed build; use the default seven scopes or an allowed --scope subset", flag).WithParam(flag)
		}
	}
	if opts.DeviceCode != "" {
		requested, err := loadLoginRequestedScope(opts.DeviceCode)
		if err != nil || requested == "" {
			return errs.NewValidationError(errs.SubtypeFailedPrecondition,
				"docs7-managed cannot resume an authorization without a valid local requested-scope record").WithParam("--device-code")
		}
		if err := larkauth.ValidateDocumentScope(requested); err != nil {
			return errs.NewValidationError(errs.SubtypeFailedPrecondition,
				"recorded authorization is outside docs7-managed: %v", err).WithParam("--device-code").WithCause(err)
		}
		return nil
	}
	opts.Scope = normalizeScopeInput(opts.Scope)
	if opts.Scope == "" {
		opts.Scope = strings.Join(larkauth.DocumentUserScopes(), " ")
	}
	return larkauth.ValidateDocumentScope(opts.Scope)
}
