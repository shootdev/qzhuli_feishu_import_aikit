// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// credential-export writes the selected local app's credentials to a new file.
// It never prints the secret or accepts it as a command-line argument.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/keychain"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/internal/vfs"
)

const defaultRedirect = "http://127.0.0.1:4366/oauth/callback"

type options struct {
	appID       string
	profile     string
	output      string
	redirectURL string
}

type credentialDocument struct {
	AppID       string `json:"app_id"`
	AppSecret   string `json:"app_secret"`
	RedirectURL string `json:"redirect_url"`
}

func exportCredential(config *core.MultiAppConfig, kc keychain.KeychainAccess, opts options, out io.Writer) (err error) {
	if config == nil || opts.appID == "" || opts.profile == "" || opts.output == "" {
		return errors.New("app ID, profile, and output path are required")
	}
	app := config.FindApp(opts.profile)
	if app == nil || app.AppId != opts.appID {
		return errors.New("profile and App ID do not identify the same configured app")
	}
	if err := core.ValidateSecretKeyMatch(app.AppId, app.AppSecret); err != nil {
		return errors.New("app credential reference does not match App ID")
	}
	u, err := url.Parse(opts.redirectURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errors.New("redirect URL must be an absolute HTTP(S) URL without user info or fragment")
	}
	path, err := validate.SafeOutputPath(opts.output)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	secret, err := core.ResolveSecretInput(app.AppSecret, kc)
	if err != nil || secret == "" {
		return errors.New("could not resolve the selected app's local secret")
	}
	if strings.ContainsAny(secret, "\r\n") {
		return errors.New("the app secret contains unexpected newline characters")
	}
	f, err := vfs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create new output file: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = f.Close()
			_ = vfs.Remove(path)
		}
	}()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(credentialDocument{AppID: app.AppId, AppSecret: secret, RedirectURL: opts.redirectURL}); err != nil {
		return errors.New("could not write the credential delivery file")
	}
	if err := f.Close(); err != nil {
		return errors.New("could not close the credential delivery file")
	}
	complete = true
	return json.NewEncoder(out).Encode(map[string]any{"ok": true, "app_id": app.AppId, "file_path": path})
}

func main() {
	var opts options
	flag.StringVar(&opts.appID, "app-id", "", "the exact App ID")
	flag.StringVar(&opts.profile, "profile", "", "the named CLI profile")
	flag.StringVar(&opts.output, "output", "", "new output file inside the current directory")
	flag.StringVar(&opts.redirectURL, "redirect-url", defaultRedirect, "the configured OAuth redirect URL")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(errors.New("unexpected positional arguments"))
	}
	config, err := core.LoadMultiAppConfig()
	if err != nil {
		fail(errors.New("could not load the current CLI configuration"))
	}
	if err := exportCredential(config, keychain.Default(), opts, os.Stdout); err != nil {
		fail(err)
	}
}

func fail(err error) {
	_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"ok": false, "error": err.Error()})
	os.Exit(1)
}
