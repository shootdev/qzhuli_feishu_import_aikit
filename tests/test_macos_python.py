import base64
import contextlib
import gzip
import importlib.util
import io
import json
import os
import subprocess
import sys
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from unittest.mock import patch
from urllib.parse import parse_qs, urlsplit

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = ROOT / "skills/feishu-enterprise-app-setup/scripts"
sys.path.insert(0, str(SCRIPTS))
spec = importlib.util.spec_from_file_location("mac_cli", SCRIPTS / "feishu_app.py")
cli = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cli)
BUILD = json.loads((ROOT / "BUILD-INFO.json").read_text(encoding="utf-8"))
SCOPES = [{"scope_name": name, "scope_type": kind, "grant_status": 1}
          for kind in ("user", "tenant") for name in BUILD[kind + "_scopes"]]


class WireTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.delivery = Path(self.tmp.name) / "app.json"
        self.requests = []
        self.scopes = SCOPES.copy()
        self.user_scope = " ".join(BUILD["user_scopes"])
        self.fail_publish = False
        owner = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def handle_request(self):
                raw = self.rfile.read(int(self.headers.get("Content-Length", 0)))
                if self.headers.get_content_type() == "application/json":
                    data = json.loads(raw)
                else:
                    data = {k: v[0] for k, v in parse_qs(raw.decode()).items()}
                path = self.path
                owner.requests.append((self.command, path, data, self.headers.get("Authorization")))
                status = 200
                if path == "/register" and data["action"] == "begin":
                    payload = {"verification_uri_complete": "https://open.feishu.cn/page/launcher?user_code=TEST&clientID=old",
                               "device_code": "test-device", "interval": 1}
                elif path == "/register":
                    payload = {"client_id": "cli_test", "client_secret": "TEST_SECRET"}
                elif path == "/oauth/v1/device_authorization":
                    payload = {"device_code": "test-user-device", "verification_uri_complete": "https://accounts.feishu.cn/confirm"}
                elif path == "/token" and data.get("grant_type") == "client_credentials":
                    payload = {"access_token": "tenant-test", "token_type": "Bearer"}
                elif path == "/token":
                    payload = {"access_token": "user-test", "scope": owner.user_scope}
                elif path.endswith("/scopes"):
                    payload = {"code": 0, "data": {"scopes": owner.scopes[:3], "has_more": True, "page_token": "next"}}
                elif path.endswith("/scopes?page_token=next"):
                    payload = {"code": 0, "data": {"scopes": owner.scopes[3:]}}
                elif path.endswith("/config"):
                    payload = {"code": 0, "data": {}}
                elif path.endswith("/publish"):
                    status = 500 if owner.fail_publish else 200
                    payload = {"code": 123} if owner.fail_publish else {"code": 0, "data": {"version": "1.0.1", "version_id": "test-version"}}
                else:
                    status, payload = 404, {}
                self.send_response(status)
                self.end_headers()
                self.wfile.write(json.dumps(payload).encode())

            do_GET = do_PATCH = do_POST = handle_request

        server = HTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)
        base = "http://127.0.0.1:%d" % server.server_port
        for k, v in {"OPEN": base, "ACCOUNTS": base, "REGISTER": base + "/register", "TOKEN": base + "/token"}.items():
            item = patch.object(cli, k, v)
            item.start()
            self.addCleanup(item.stop)
        sleep = patch.object(cli.time, "sleep")
        sleep.start()
        self.addCleanup(sleep.stop)

    def run_command(self, *argv):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            cli.main(list(map(str, argv)))
        self.assertNotIn("TEST_SECRET", out.getvalue())
        return [json.loads(line) for line in out.getvalue().splitlines()]

    def test_create_through_publication(self):
        result = self.run_command("create", "--output", self.delivery)
        self.assertEqual(result[-1]["status"], "submitted")
        confirmation = result[0]
        query = parse_qs(urlsplit(confirmation["verification_url"]).query)
        self.assertEqual(query["createOnly"], ["true"])
        self.assertNotIn("clientID", query)
        addons = json.loads(gzip.decompress(base64.urlsafe_b64decode(query["addons"][0] + "===")))
        self.assertEqual(addons, {"preset": False, "scopes": {"user": BUILD["user_scopes"], "tenant": BUILD["tenant_scopes"]}})
        self.assertFalse(Path(confirmation["qr_file"]).exists())
        self.assertEqual(json.loads(self.delivery.read_text()), {"app_id": "cli_test", "app_secret": "TEST_SECRET", "redirect_url": BUILD["redirect_url"]})
        if os.name == "posix":
            self.assertEqual(self.delivery.stat().st_mode & 0o777, 0o600)
        for method, path, data, auth in self.requests[-2:]:
            filename = "config" if method == "PATCH" else "publish"
            self.assertTrue(path.endswith("/cli_test/" + filename))
            self.assertEqual(data, json.loads((ROOT / "templates" / (filename + ".json")).read_text()))
            self.assertEqual(auth, "Bearer tenant-test")
        with self.assertRaises(cli.KitError):
            self.run_command("publish", "--credentials", self.delivery)
        with self.assertRaises(FileExistsError):
            self.run_command("create", "--output", self.delivery)
        self.assertEqual(sum(path.endswith("/publish") for _, path, _, _ in self.requests), 1)

    def test_stop_on_unexpected_scope_then_resume(self):
        self.scopes.append({"scope_type":"user", "scope_name":"im:message", "grant_status":1})
        with self.assertRaises(cli.KitError):
            self.run_command("create", "--output", self.delivery)
        self.assertTrue(self.delivery.is_file())
        self.assertFalse(any(method == "PATCH" for method, *_ in self.requests))
        self.scopes.pop()
        self.fail_publish = True
        with self.assertRaises(cli.KitError):
            self.run_command("publish", "--credentials", self.delivery)
        with self.assertRaises(cli.KitError):
            self.run_command("publish", "--credentials", self.delivery)
        self.assertEqual(sum(path.endswith("/publish") for _, path, _, _ in self.requests), 1)

    def test_user_oauth_scopes_and_private_output(self):
        self.delivery.write_text(json.dumps({"app_id": "cli_test", "app_secret": "TEST_SECRET", "redirect_url": cli.REDIRECT}))
        output = self.delivery.with_name("employee.json")
        self.run_command("user-login", "--credentials", self.delivery, "--output", output)
        self.assertEqual(self.requests[0][2]["scope"].split(), BUILD["user_scopes"])
        self.assertEqual(json.loads(output.read_text())["access_token"], "user-test")
        output.unlink()
        self.user_scope += " application:application:patch"
        with self.assertRaises(cli.KitError):
            self.run_command("user-login", "--credentials", self.delivery, "--output", output)
        self.assertFalse(output.exists())

    def test_external_and_broad_template_links_fail(self):
        for url in ("https://open.feishu.cn.evil.test/page/launcher?user_code=x",
                    "https://open.feishu.cn/page/cli?user_code=x", "https://open.feishu.cn/page/launcher"):
            with self.assertRaises(cli.KitError):
                cli.registration_link({"verification_uri_complete": url})

    def test_normal_package_entrypoint_preview(self):
        output = subprocess.check_output([sys.executable, "-B", str(ROOT / "feishu_app.py"), "preview"], cwd=self.tmp.name)
        result = json.loads(output)
        self.assertEqual(result["configuration_defaults"], json.loads((ROOT / "templates/config.json").read_text()))
        self.assertEqual(result["addons"]["scopes"]["tenant"], BUILD["tenant_scopes"])


if __name__ == "__main__":
    unittest.main()
