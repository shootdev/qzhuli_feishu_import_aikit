# Seven user scopes plus application self-configuration

This local distribution is based on official `larksuite/cli v1.0.100`
(commit `ac22a60a23d66c37d43a7ec6d9e99b99a4320356`). Version
`1.0.100-docs7.2` uses build policy `docs7-managed`.

New applications use the official SDK minimal-template protocol (`preset:false`)
with exactly seven document user scopes and exactly one tenant scope:
`application:application:patch`. The user OAuth allowlist remains seven; the
management scope is never requested as a user scope.

## Build

```sh
make build VERSION=1.0.100-docs7.2
```

Without Make, use the equivalent command and choose the output filename for the
host:

```sh
go build -trimpath -ldflags "-s -w -X github.com/larksuite/cli/internal/build.Version=1.0.100-docs7.2 -X github.com/larksuite/cli/internal/build.PermissionPreset=docs7-managed" -o lark-cli .
```

Plain `go build` without the policy linker flag retains upstream defaults and is
not this distribution. Verify both the version and the dry-run payload.

## Create, configure, and submit publication

1. Run `lark-cli config init --new --dry-run`. Confirm seven user scopes, the
   single tenant management scope, `preset:false`, and `create_only:true`.
2. Run `lark-cli config init --new --brand feishu --name <unique-profile>`.
   Open the generated URL and confirm the correct tenant and exact permissions.
   Stop if the platform displays additional permissions. The CLI rejects missing
   or unsafe SDK URLs instead of falling back to the broad CLI template.
3. Save `configuration_defaults` and `publication_defaults` from the preview as
   two local JSON files. They contain no credentials. The configuration adds
   `http://127.0.0.1:4366/oauth/callback`, enables refresh tokens, and sets
   `visibility.is_visible_to_all=true`. It does not overwrite other callbacks or
   change contact data permissions.
4. With the newly returned app ID and named profile, preview and execute:

```text
lark-cli api PATCH /open-apis/application/v7/applications/<app_id>/config --profile <profile> --as bot --data @config.json --dry-run
lark-cli api PATCH /open-apis/application/v7/applications/<app_id>/config --profile <profile> --as bot --data @config.json
lark-cli api POST /open-apis/application/v7/applications/<app_id>/publish --profile <profile> --as bot --data @publish.json --dry-run
lark-cli api POST /open-apis/application/v7/applications/<app_id>/publish --profile <profile> --as bot --data @publish.json
```

The management scope must already be effective. These APIs only support eligible
self-built app sources; do not assume every SDK-created app is eligible before
live verification. Publication success means submitted, not approved. Version
and availability read APIs may require other scopes; do not add them. Use the
necessary console step to verify final status or complete required review.

## User authorization and cleanup

`auth login` defaults to the original seven user scopes. Extra scopes,
`--domain`, `--recommend`, and `--exclude` are rejected before HTTP. Device flow
resumption requires an allowed local scope record; tokens with extra or missing
scope claims are rejected without storing them.

Use tenant credentials only to configure this same app. Use user credentials for
employee documents. Do not require user OAuth merely to configure or publish.
After verification and delivery, remove only this task's local app binding and
credentials. Keep the remote application, its authorized seven-plus-one scope
configuration, unrelated profiles, and requested credential deliverables.

## Checks

```sh
go test ./internal/auth ./cmd/auth ./cmd/config
go vet ./internal/auth ./cmd/auth ./cmd/config
node scripts/check-docs7.mjs <path-to-built-executable>
```

Focused tests enable the shipped build policy and check decoded registration
payloads, actual user OAuth requests, rejection before HTTP, safe resumption,
returned token scopes, and default configuration. Local tests do not prove live
scope acceptance, callback persistence, app-source eligibility, approval, or
all-employee availability.

Official CLI updates can replace this binary. Recheck the version and preview
before future creation; keep the backup and source patch for rebuilding.

## Official references

- [CLI registration source](https://github.com/larksuite/cli/blob/v1.0.100/internal/auth/app_registration.go)
- [SDK minimal-template protocol](https://github.com/larksuite/oapi-sdk-go/tree/v3.13.0/scene/registration)
- [Configuration API](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/application-v7/application-v7/application-config/patch.md)
- [Publication API](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/application-v7/application-v7/application-publish/create.md)
