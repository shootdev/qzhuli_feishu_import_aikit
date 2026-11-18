"""Build the macOS Python download and the source-inclusive kit from this checkout."""
import hashlib
import json
import os
import subprocess
import tempfile
from pathlib import Path
from zipfile import ZIP_DEFLATED, ZipFile, ZipInfo

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "dist"
BASE = "feishu-app-kit-1.0.100-docs7.2"
NATIVE = {"windows-x64": "windows-amd64", "windows-arm64": "windows-arm64",
          "linux-x64": "linux-amd64", "linux-arm64": "linux-arm64"}


def tracked(folder):
    result = subprocess.check_output(["git", "ls-files", "-z", "--", folder], cwd=ROOT)
    return [name.decode("utf-8") for name in result.split(b"\0") if name]


def manifest(files):
    return "".join(hashlib.sha256(data).hexdigest() + "  " + name + "\n"
                   for name, data in sorted(files.items())).encode("utf-8")


def write_zip(suffix, files):
    files = dict(files, SHA256SUMS=manifest(files))
    dest = DIST / (BASE + "-" + suffix + ".zip")
    with tempfile.TemporaryDirectory(dir=DIST) as directory:
        temp = Path(directory) / dest.name
        with ZipFile(temp, "w") as archive:
            for name, data in sorted(files.items()):
                entry = ZipInfo(BASE + "/" + name, (2026, 11, 18, 0, 0, 0))
                entry.create_system = 3
                executable = name.startswith("bin/") or name == "install.sh" or name in SOURCE_EXECUTABLES
                entry.external_attr = (0o100755 if executable else 0o100644) << 16
                archive.writestr(entry, data, compress_type=ZIP_DEFLATED, compresslevel=6)
        with ZipFile(temp) as archive:
            if archive.testzip() is not None:
                raise ValueError("ZIP CRC verification failed")
        os.replace(temp, dest)
    checksum = hashlib.sha256(dest.read_bytes()).hexdigest()
    Path(str(dest) + ".sha256").write_bytes((checksum + "  " + dest.name + "\n").encode())
    print(dest.name, dest.stat().st_size, checksum, flush=True)


def json_bytes(value):
    return (json.dumps(value, indent=2, ensure_ascii=False) + "\n").encode()


SOURCE_EXECUTABLES = set()


def build():
    files = {}
    for name in tracked("skills") + tracked("templates") + [
        "feishu_app.py", "LICENSE", "NOTICE.txt", "VALIDATION.md", "tests/test_macos_python.py",
    ]:
        files[name] = (ROOT / name).read_bytes()
    files["install.sh"] = (ROOT / "packaging/install.sh").read_bytes()
    metadata = json.loads((ROOT / "BUILD-INFO.json").read_bytes())
    files["BUILD-INFO.json"] = json_bytes(dict(metadata, targets=["macos-python"], source_included=False,
                                              source_path="skills/feishu-enterprise-app-setup/scripts"))
    files["README.md"] = (ROOT / "skills/feishu-enterprise-app-setup/references/macos-python.md").read_bytes()
    write_zip("macos-python", files)

    files.update({key: (ROOT / key).read_bytes() for key in tracked("src") + tracked("third-party-licenses")})
    for line in subprocess.check_output(["git", "ls-files", "--stage", "-z", "src"], cwd=ROOT).split(b"\0"):
        if line.startswith(b"100755 "):
            SOURCE_EXECUTABLES.add(line.split(b"\t", 1)[1].decode())
    files["README.md"] = (ROOT / "packaging/README-full.md").read_bytes()
    files["BUILD-INFO.json"] = json_bytes(metadata)
    listings = {name: [] for name in ("BINARIES.json", "CREDENTIAL-EXPORT-BINARIES.json")}
    for suffix, target in NATIVE.items():
        native = DIST / (BASE + "-" + suffix + ".zip")
        expected = Path(str(native) + ".sha256").read_text().split()[0]
        if hashlib.sha256(native.read_bytes()).hexdigest() != expected:
            raise ValueError("Native package checksum failed: " + suffix)
        with ZipFile(native) as archive:
            for name in listings:
                records = json.loads(archive.read(BASE + "/" + name))["targets"]
                if len(records) != 1 or records[0]["target"] != target:
                    raise ValueError("Unexpected native binary target")
                record = records[0]
                binary_path = record["path"]
                data = archive.read(BASE + "/" + binary_path)
                if not binary_path.startswith("bin/" + target + "/") or hashlib.sha256(data).hexdigest() != record["sha256"]:
                    raise ValueError("Native binary checksum failed: " + target)
                files[binary_path] = data
                listings[name].append(record)
            if suffix == "windows-x64":
                for name in ("install.ps1", "check-cli.mjs"):
                    files[name] = archive.read(BASE + "/" + name)
    for name, records in listings.items():
        files[name] = json_bytes({"version": metadata["version"], "targets": records})
    write_zip("full-with-source", files)


if __name__ == "__main__":
    build()
