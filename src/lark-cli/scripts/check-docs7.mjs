// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Offline acceptance checks for the actual executable, with no account state.
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { existsSync, lstatSync, mkdtempSync, realpathSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';

const executable = path.resolve(process.argv[2] || './lark-cli');
const tempRoot = realpathSync(tmpdir());
const sandbox = mkdtempSync(path.join(tempRoot, 'lark-cli-docs7-check-'));
const env = { ...process.env };
for (const key of Object.keys(env)) {
  if (/^LARKSUITE_CLI_/i.test(key) || /^(HERMES_HOME|OPENCLAW_HOME)$/i.test(key)) delete env[key];
}
Object.assign(env, {
  LARKSUITE_CLI_CONFIG_DIR: path.join(sandbox, 'config'),
  LARKSUITE_CLI_LOG_DIR: path.join(sandbox, 'logs'),
  LARKSUITE_CLI_NO_UPDATE_NOTIFIER: '1',
  LARKSUITE_CLI_NO_SKILLS_NOTIFIER: '1',
});

const scopes = [
  'docs:document.content:read', 'docs:document.media:download',
  'docx:document:readonly', 'drive:drive:readonly', 'offline_access',
  'space:document:retrieve', 'wiki:wiki:readonly',
];
function run(args) {
  const result = spawnSync(executable, args, { env, encoding: 'utf8', timeout: 30000, windowsHide: true });
  if (result.error) throw result.error;
  return result;
}

try {
  const version = run(['--version']);
  assert.equal(version.status, 0);
  assert.match(version.stdout, /docs7/);
  const preview = run(['config', 'init', '--new', '--dry-run']);
  assert.equal(preview.status, 0, preview.stderr);
  const payload = JSON.parse(preview.stdout);
  assert.equal(payload.permission_preset, 'docs7-managed');
  assert.equal(payload.dry_run, true);
  assert.equal(payload.create_only, true);
  assert.equal(payload.addons.preset, false);
  assert.deepEqual(payload.addons.scopes.tenant, ['application:application:patch']);
  assert.deepEqual(payload.configuration_defaults.security.add.redirect_urls, ['http://127.0.0.1:4366/oauth/callback']);
  assert.equal(payload.configuration_defaults.security.allow_refresh_token, true);
  assert.equal(payload.configuration_defaults.visibility.is_visible_to_all, true);
  assert.equal(payload.publication_defaults.pc_default_ability, 'bot');
  assert.equal(payload.publication_defaults.mobile_default_ability, 'bot');
  assert.deepEqual([...payload.addons.scopes.user].sort(), [...scopes].sort());

  const denied = [
    ['--scope', 'application:application:patch'],
    ['--scope', 'docx:document:readonly im:message'],
    ['--domain', 'all'],
    ['--recommend'],
    ['--exclude', 'im:message'],
    ['--device-code', 'EXAMPLE_DEVICE_WITHOUT_CACHE'],
  ];
  for (const args of denied) {
    const result = run(['auth', 'login', ...args]);
    assert.equal(result.status, 2, `expected local validation rejection for ${args[0]}: ${result.stderr}`);
    const error = JSON.parse(result.stderr);
    assert.equal(error.ok, false);
    assert.equal(error.error.type, 'validation');
    assert.equal(error.error.param, args[0]);
  }
  assert.equal(existsSync(path.join(env.LARKSUITE_CLI_CONFIG_DIR, 'config.json')), false);
  console.log(JSON.stringify({ version: version.stdout.trim(), dry_run: 'passed', seven_user_scopes: 'passed', single_management_tenant_scope: 'passed', callback_and_all_employee_defaults: 'passed', rejected_unsafe_requests: denied.length, local_binding_created: false }, null, 2));
} finally {
  // Only remove this script's freshly allocated, canonical child of the temp root.
  const resolved = realpathSync(sandbox);
  assert.equal(path.dirname(resolved), tempRoot);
  assert.ok(path.basename(resolved).startsWith('lark-cli-docs7-check-'));
  assert.equal(lstatSync(sandbox).isSymbolicLink(), false);
  rmSync(resolved, { recursive: true, force: true });
}
