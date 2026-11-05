"""Verify the exact official APT fix and native Trivy delta; never filter findings."""
import argparse
from collections import Counter
from copy import deepcopy
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CVE = "CVE-2026-109518"
VERSION = "3.2.1ubuntu0.1"
PURLS = {name: "pkg:deb/ubuntu/" + name + "@" + VERSION + "?arch=amd64&distro=ubuntu-26.04"
         for name in ("apt", "libapt-pkg7.0")}
SHAS = {"apt": "0de729b8fe0e1c67c04e8e8d5bd6e0420dbd47f75b93f0b71c8e740edc2d5ffc",
        "libapt-pkg7.0": "94e5d41303265b66af0f8c70ade18bfb77260ad286edee6c8045faddef82b94c"}


def validate_vex(vex):
    if (set(vex) != {"@context", "@id", "author", "timestamp", "version", "statements"}
            or vex["@context"] != "https://openvex.dev/ns/v0.2.0"
            or vex["@id"] != "https://github.com/PastureStack/compose-cli/security/dapper/apt-usn-8996-1"
            or vex["author"] != "PastureStack contributors" or vex["version"] != 1
            or vex["timestamp"] != "2026-11-05T00:00:00Z" or len(vex["statements"]) != 1):
        raise ValueError("Exact Compose fixed document required")
    statement = vex["statements"][0]
    if (set(statement) != {"vulnerability", "products", "status", "impact_statement"}
            or statement["vulnerability"] != {"name": CVE} or statement["status"] != "fixed"
            or Counter(json.dumps(p, sort_keys=True) for p in statement["products"])
            != Counter(json.dumps({"@id": p}, sort_keys=True) for p in PURLS.values())
            or "https://ubuntu.com/security/notices/USN-8996-1" not in statement["impact_statement"]
            or not all(sha in statement["impact_statement"] for sha in SHAS.values())):
        raise ValueError("Only the exact official fixed APT pair is permitted")


def exact_finding(finding):
    name = finding.get("PkgName")
    return (finding.get("VulnerabilityID") == CVE and name in PURLS
            and finding.get("InstalledVersion") == VERSION
            and (finding.get("PkgIdentifier") or {}).get("PURL") == PURLS[name])


def validate(raw, effective, dpkg, vex):
    validate_vex(vex)
    installed = {}
    for line in dpkg.splitlines():
        fields = line.split("\t")
        if len(fields) == 2 and fields[0].split(":")[0] in PURLS:
            name = fields[0].split(":")[0]
            if name in installed:
                raise ValueError("Duplicate APT package")
            installed[name] = fields[1]
    if installed != dict.fromkeys(PURLS, VERSION):
        raise ValueError("Actual installed fixed pair required")
    if (raw.get("SchemaVersion") != 2 or effective.get("SchemaVersion") != 2
            or not raw.get("Results") or not effective.get("Results")):
        raise ValueError("Both original complete native reports are required")
    packages = {}
    for result in raw["Results"]:
        if result.get("Type") == "ubuntu":
            for package in result.get("Packages", []):
                name = package.get("Name")
                if name in PURLS:
                    if name in packages:
                        raise ValueError("Duplicate analyzed APT package")
                    packages[name] = (package.get("Version"), (package.get("Identifier") or {}).get("PURL"))
    if packages != {name: (VERSION, purl) for name, purl in PURLS.items()}:
        raise ValueError("Actual Ubuntu scan package identity required")
    def sections(report):
        values = {}
        for result in report["Results"]:
            key = (result.get("Target"), result.get("Class"), result.get("Type"))
            if key in values:
                raise ValueError("Duplicate scan section")
            value = dict(result)
            findings = value.pop("Vulnerabilities", []) or []
            if not isinstance(findings, list):
                raise ValueError("Malformed findings")
            # Native output may reorder packages or findings; compare every record.
            if "Packages" in value:
                normalized = []
                for package in value["Packages"]:
                    package = deepcopy(package)
                    # Trivy's per-scan UID is not a package identity. A second
                    # native scan may also materialize this derived ID.
                    if package.get("ID") == str(package.get("Name")) + "@" + str(package.get("Version")):
                        package.pop("ID")
                    if isinstance(package.get("Identifier"), dict):
                        package["Identifier"].pop("UID", None)
                    normalized.append(package)
                value["Packages"] = sorted(normalized, key=lambda x: json.dumps(x, sort_keys=True))
            values[key] = (value, findings)
        return values
    original, native = sections(raw), sections(effective)
    if original.keys() != native.keys():
        raise ValueError("Native scan section drift")
    removed = 0
    for key, (metadata, findings) in original.items():
        new_metadata, new_findings = native[key]
        if metadata != new_metadata or any(exact_finding(item) for item in new_findings):
            raise ValueError("Native fix or package/secret metadata mismatch")
        retained = [item for item in findings if not (key[2] == "ubuntu" and exact_finding(item))]
        removed += len(findings) - len(retained)
        if Counter(json.dumps(x, sort_keys=True) for x in retained) != Counter(json.dumps(x, sort_keys=True) for x in new_findings):
            raise ValueError("Unrelated finding changed by native VEX")
    if removed > 2:
        raise ValueError("Duplicate fixed occurrence")
    return {"status": "PASS_EXACT_COMPOSE_NATIVE_APT_FIXED_VEX", "fixedOccurrences": removed,
            "packageVersions": installed, "purls": PURLS, "nativeTrivyFilter": True,
            "pythonFiltered": False, "rawReportPreserved": True, "runtimePass": False}


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--raw", type=Path, required=True)
    parser.add_argument("--native", type=Path, required=True)
    parser.add_argument("--dpkg", type=Path, required=True)
    args = parser.parse_args()
    proof = validate(json.loads(args.raw.read_text()), json.loads(args.native.read_text()),
                     args.dpkg.read_text(), json.loads((ROOT / "security/dapper-apt.openvex.json").read_text()))
    print(json.dumps(proof, indent=2))
