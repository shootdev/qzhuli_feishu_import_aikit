#!/usr/bin/env python3
"""Feishu enterprise app setup for macOS. Run with Python 3.8 or later."""
import argparse
import base64
import gzip
import json
import os
import re
import signal
import stat
import struct
import sys
import tempfile
import time
import zlib
from contextlib import contextmanager
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener

ACCOUNTS = "https://accounts.feishu.cn"
OPEN = "https://open.feishu.cn"
REGISTER = ACCOUNTS + "/oauth/v1/app/registration"
TOKEN = ACCOUNTS + "/oauth/v3/token"
REDIRECT = "http://127.0.0.1:4366/oauth/callback"
USER_SCOPES = ("docs:document.content:read", "docs:document.media:download",
               "docx:document:readonly", "drive:drive:readonly", "offline_access",
               "space:document:retrieve", "wiki:wiki:readonly")
MANAGE = "application:application:patch"
ADDONS = {"preset": False, "scopes": {"user": list(USER_SCOPES), "tenant": [MANAGE]}}
CONFIG = {"security": {"add": {"redirect_urls": [REDIRECT]}, "allow_refresh_token": True},
          "visibility": {"is_visible_to_all": True}}
PUBLICATION = {
    "mobile_default_ability": "bot", "pc_default_ability": "bot",
    "remark": "Enable employee-authorized document import for all employees.",
    "changelog": "Seven document user scopes; one application scope for self-management; fixed OAuth redirect and all-employee availability."
}


class KitError(Exception):
    pass


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def request(method, url, *, form=None, body=None, headers=None, polling=False):
    headers = dict(headers or {})
    payload = None
    if form is not None:
        headers["Content-Type"] = "application/x-www-form-urlencoded"
        payload = urlencode(form).encode()
    elif body is not None:
        headers["Content-Type"] = "application/json; charset=utf-8"
        payload = json.dumps(body).encode()
    req = Request(url, data=payload, method=method, headers=headers)
    try:
        response = build_opener(NoRedirect()).open(req, timeout=25)
    except HTTPError as exc:
        response = exc  # OAuth pending/slow_down may also arrive as HTTP 400.
    with response:
        raw = response.read(1_048_577)
        if len(raw) > 1_048_576:
            raise KitError("Response too large")
        try:
            data = json.loads(raw)
        except (ValueError, UnicodeError):
            raise KitError("Invalid JSON from Feishu (HTTP %s)" % response.status) from None
        if not isinstance(data, dict):
            raise KitError("Unexpected response format")
        oauth_error = data.get("error")
        if polling and response.status in (200, 400) and oauth_error in ("authorization_pending", "slow_down"):
            return data
        if not 200 <= response.status < 300 or oauth_error or data.get("code", 0) != 0:
            # Error documents can echo submitted fields. Print only the code.
            code = str(data.get("code") or oauth_error or "unknown")
            code = code if re.fullmatch(r"[-a-zA-Z0-9_]{1,64}", code) else "unknown"
            raise KitError("%s %s failed (HTTP %s, Feishu %s)" %
                           (method, urlsplit(url).path, response.status, code))
        return data


def required(data, field):
    value = data.get(field)
    if not isinstance(value, str) or not value.strip():
        raise KitError("Missing or invalid field: " + field)
    return value


def app_id(data):
    value = required(data, "app_id")
    if not re.fullmatch(r"cli_[A-Za-z0-9]+", value):
        raise KitError("Invalid Feishu App ID")
    return value


def checked_link(raw):
    parts = urlsplit(raw)
    if parts.scheme != "https" or parts.netloc not in ("open.feishu.cn", "accounts.feishu.cn") or parts.fragment:
        raise KitError("Unexpected Feishu confirmation URL")
    return parts


def registration_link(data):
    parts = checked_link(required(data, "verification_uri_complete"))
    if parts.path.rstrip("/") != "/page/launcher":
        raise KitError("Unrecognized SDK creation page; check the current Feishu registration protocol")
    query = dict(parse_qsl(parts.query))
    if not query.get("user_code"):
        raise KitError("Missing creation confirmation code")
    encoded = gzip.compress(json.dumps(ADDONS, separators=(",", ":")).encode(), mtime=0)
    query.pop("clientID", None)
    query.update(createOnly="true", tp="sdk", source="python/docs7-managed",
                 addons=base64.urlsafe_b64encode(encoded).rstrip(b"=").decode())
    query["from"] = "sdk"
    return urlunsplit(parts._replace(query=urlencode(query)))


def make_qr(url, target):
    from qrcodegen import QrCode
    qr = QrCode.encode_text(url, QrCode.Ecc.MEDIUM)
    scale, margin = 5, 4
    size = (qr.get_size() + 2 * margin) * scale
    rows = bytearray()
    for y in range(-margin, qr.get_size() + margin):
        row = b"".join((b"\0" if qr.get_module(x, y) else b"\xff") * scale
                       for x in range(-margin, qr.get_size() + margin))
        rows.extend((b"\0" + row) * scale)

    def chunk(kind, content):
        return struct.pack(">I", len(content)) + kind + content + struct.pack(">I", zlib.crc32(kind + content))

    header = struct.pack(">IIBBBBB", size, size, 8, 0, 0, 0, 0)
    target.write_bytes(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", header)
                       + chunk(b"IDAT", zlib.compress(rows)) + chunk(b"IEND", b""))


def emit(**fields):
    print(json.dumps(fields, ensure_ascii=False), flush=True)


def confirmed(begin, link, poll, ready):
    lifetime = int(begin.get("expire_in") or begin.get("expires_in") or 600)
    interval = max(1, int(begin.get("interval") or 5))
    deadline = time.monotonic() + min(lifetime, 1800)
    device = required(begin, "device_code")
    with tempfile.TemporaryDirectory(prefix="feishu-confirm-") as directory:
        qr = Path(directory) / "confirm.png"
        make_qr(link, qr)
        emit(verification_url=link, qr_file=str(qr))
        while time.monotonic() < deadline:
            time.sleep(min(interval, max(0, deadline - time.monotonic())))
            if time.monotonic() >= deadline:
                break
            try:
                result = poll(device)
            except (URLError, TimeoutError):
                # Polling for a decision is safe to repeat inside the original deadline.
                interval = min(60, interval + 2)
                continue
            if result.get("error") == "slow_down":
                interval = min(60, interval + 5)
            if ready(result):
                return result
    raise KitError("Confirmation expired. Check the app console before creating again.")


def create_credentials():
    begin = request("POST", REGISTER, form={"action": "begin", "archetype": "PersonalAgent",
                    "auth_method": "client_secret", "request_user_info": "open_id tenant_brand"})

    def poll(device):
        result = request("POST", REGISTER, form={"action": "poll", "device_code": device}, polling=True)
        if (result.get("user_info") or {}).get("tenant_brand") not in (None, "", "feishu"):
            raise KitError("This preset is for Feishu China. Check the selected tenant.")
        return result

    result = confirmed(begin, registration_link(begin), poll, lambda r: r.get("client_id") and r.get("client_secret"))
    credentials = {"app_id": required(result, "client_id"), "app_secret": required(result, "client_secret"),
                   "redirect_url": REDIRECT}
    app_id(credentials)
    return credentials


@contextmanager
def new_private_file(path):
    # Exclusive creation reserves the destination before starting a remote action.
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            yield stream
            stream.flush()
            os.fsync(stream.fileno())
    except BaseException:
        path.unlink()
        raise


def load_credentials(path):
    with path.open(encoding="utf-8") as stream:
        if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
            raise KitError("Expected a regular credential file")
        data = json.load(stream)
    if not isinstance(data, dict):
        raise KitError("Invalid credentials JSON")
    app_id(data)
    required(data, "app_secret")
    if data.get("redirect_url") != REDIRECT:
        raise KitError("Delivery file does not match this preset's redirect URL")
    return data


def tenant_token(data):
    result = request("POST", TOKEN, form={"grant_type": "client_credentials",
                     "client_id": app_id(data), "client_secret": required(data, "app_secret")})
    if str(result.get("token_type", "Bearer")).lower() != "bearer":
        raise KitError("Tenant policy requires a different token type")
    return required(result, "access_token")


def api(method, path, token, body=None):
    response = request(method, OPEN + "/open-apis/" + path, body=body,
                       headers={"Authorization": "Bearer " + token})
    if response.get("code") != 0 or not isinstance(response.get("data"), dict):
        raise KitError("Missing successful Feishu API result")
    return response["data"]


def verify_scopes(token):
    expected = {("user", name, 1) for name in USER_SCOPES} | {("tenant", MANAGE, 1)}
    found, seen_pages = set(), set()
    query = ""
    while True:
        page = api("GET", "application/v6/scopes" + query, token)
        for item in page.get("scopes", []):
            found.add((item["scope_type"], item["scope_name"], item["grant_status"]))
        if not page.get("has_more"):
            break
        cursor = required(page, "page_token")
        if cursor in seen_pages:
            raise KitError("Repeated scope page token")
        seen_pages.add(cursor)
        query = "?" + urlencode({"page_token": cursor})
    if found != expected:
        raise KitError("Expected exactly seven granted user scopes and one granted tenant management scope")


def publish(credentials, delivery):
    identifier = app_id(credentials)
    receipt = delivery.with_name(delivery.name + ".publish.json")
    if receipt.exists():
        raise KitError("A submission record exists. Check it and the version in the developer console: " + str(receipt))
    token = tenant_token(credentials)
    verify_scopes(token)
    prefix = "application/v7/applications/" + identifier
    api("PATCH", prefix + "/config", token, CONFIG)
    # Keep a pending marker if connection/response is lost after sending POST.
    with new_private_file(receipt) as stream:
        json.dump({"app_id": identifier, "status": "submission_unconfirmed"}, stream)
    version = api("POST", prefix + "/publish", token, PUBLICATION)
    record = {"app_id": identifier, "status": "submitted", "version": required(version, "version"),
              "version_id": required(version, "version_id")}
    with tempfile.NamedTemporaryFile(mode="w", dir=receipt.parent, delete=False, encoding="utf-8") as tmp:
        json.dump(record, tmp)
    os.replace(tmp.name, receipt)
    emit(**record, redirect_url=REDIRECT, credentials_file=str(delivery), receipt=str(receipt))


def user_login(credentials, output):
    identifier = app_id(credentials)
    secret = required(credentials, "app_secret")
    basic = base64.b64encode((identifier + ":" + secret).encode()).decode()
    with new_private_file(output) as stream:
        begin = request("POST", ACCOUNTS + "/oauth/v1/device_authorization",
                        form={"client_id": identifier, "scope": " ".join(USER_SCOPES)},
                        headers={"Authorization": "Basic " + basic})
        link = required(begin, "verification_uri_complete")
        checked_link(link)

        def poll(device):
            return request("POST", TOKEN, polling=True, form={
                "grant_type": "urn:ietf:params:oauth:grant-type:device_code", "device_code": device,
                "client_id": identifier, "client_secret": secret})

        result = confirmed(begin, link, poll, lambda r: r.get("access_token"))
        if set(required(result, "scope").split()) != set(USER_SCOPES):
            raise KitError("Returned user scopes differ from the seven requested scopes")
        if str(result.get("token_type", "Bearer")).lower() != "bearer":
            raise KitError("Unsupported user token type")
        json.dump({k: result[k] for k in ("access_token", "refresh_token", "scope", "expires_in",
                  "refresh_token_expires_in", "token_type") if k in result}, stream, indent=2)
    emit(app_id=identifier, user_credentials_file=str(output))


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", action="version", version="feishu-app Python 1.0.100-docs7.2")
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("preview", help="show creation, configuration and publication defaults offline")
    create = commands.add_parser("create", help="create, export credentials, verify scopes, configure and submit")
    create.add_argument("--output", required=True, type=Path, help="new private credential file")
    for command in ("verify", "publish", "user-login"):
        sub = commands.add_parser(command)
        sub.add_argument("--credentials", required=True, type=Path, help="existing app credential file")
        if command == "user-login":
            sub.add_argument("--output", required=True, type=Path, help="new private user token file")
    args = parser.parse_args(argv)
    if args.command == "preview":
        emit(dry_run=True, create_only=True, permission_preset="docs7-managed", addons=ADDONS,
             configuration_defaults=CONFIG, publication_defaults=PUBLICATION)
    elif args.command == "create":
        path = args.output.expanduser().absolute()
        with new_private_file(path) as stream:
            credentials = create_credentials()
            json.dump(credentials, stream, indent=2)
        emit(app_id=credentials["app_id"], credentials_file=str(path), status="created")
        publish(credentials, path)
    else:
        path = args.credentials.expanduser().absolute()
        credentials = load_credentials(path)
        if args.command == "publish":
            publish(credentials, path)
        elif args.command == "verify":
            verify_scopes(tenant_token(credentials))
            emit(app_id=credentials["app_id"], granted=ADDONS["scopes"])
        else:
            user_login(credentials, args.output.expanduser().absolute())


if __name__ == "__main__":
    def terminate(signum, frame):
        raise SystemExit(128 + signum)

    signal.signal(signal.SIGTERM, terminate)
    try:
        main()
    except (KitError, OSError, URLError, ValueError, TypeError, KeyError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)
