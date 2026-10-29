"""Fail closed on every fixed, unknown or unreviewed builder finding."""
import json
import sys
from pathlib import Path

REPOSITORY = "PastureStack/compose-cli"
HEADER_VERSION = "7.0.0-38.38"
HEADER_PURL = "pkg:deb/ubuntu/linux-libc-dev@7.0.0-38.38?arch=amd64&distro=ubuntu-26.04"


def violations(report, ledger, boundary, headers):
    if (not isinstance(report, dict) or report.get("SchemaVersion") != 2
            or not isinstance(report.get("Results"), list) or not report["Results"]):
        raise ValueError("Missing complete builder scan")
    results = report["Results"]
    if any(not isinstance(r, dict) for r in results):
        raise ValueError("Malformed scan result")
    if not any(r.get("Type") == "ubuntu" and r.get("Packages") for r in results):
        raise ValueError("Missing nonempty Ubuntu package analysis")
    binaries = {r.get("Target"): r for r in results if r.get("Type") == "gobinary"}
    if not binaries.get("usr/local/go/bin/go", {}).get("Packages"):
        raise ValueError("Missing nonempty Go compiler analysis")
    if any(not r.get("Packages") for r in binaries.values()):
        raise ValueError("Incomplete Go binary analysis")
    if ledger.get("repository") != REPOSITORY or headers.get("repository") != REPOSITORY:
        raise ValueError("Review repository mismatch")
    if (headers.get("package") != "linux-libc-dev"
            or headers.get("version") != HEADER_VERSION or headers.get("purl") != HEADER_PURL):
        raise ValueError("Changed exact header review")
    refresh = headers.get("mediumRefresh", {})
    refreshed = {}
    for entry in refresh.get("entries", []):
        identifier = entry.get("id")
        official = ("https://raw.githubusercontent.com/canonical/ubuntu-security-notices/"
                    + str(refresh.get("officialSnapshot")) + "/osv/cve/2026/UBUNTU-"
                    + str(identifier) + ".json")
        if (identifier in refreshed or identifier not in headers.get("ids", [])
                or entry.get("package") != "linux-libc-dev" or entry.get("version") != HEADER_VERSION
                or entry.get("purl") != HEADER_PURL or entry.get("severity") != "MEDIUM"
                or entry.get("decision") != "UNFIXED_MEDIUM_HEADER_ONLY"
                or entry.get("officialFixedVersions") != []
                or entry.get("officialRangeEvents") != [{"introduced": "0"}]
                or entry.get("officialUrl") != official
                or not entry.get("rawSha256") or not entry.get("target")):
            raise ValueError("Invalid exact medium header refresh")
        refreshed[identifier] = entry
    reviewed = {}
    for entry in ledger.get("unfixed", []):
        key = (entry["id"], entry["package"], entry["version"], entry["purl"])
        if (key in reviewed or entry.get("severity") not in ("LOW", "MEDIUM")
                or entry.get("status") != "no_official_fix"
                or not entry.get("officialEvidence", "").startswith("https://")
                or not entry.get("reason") or not entry.get("checkedAt")):
            raise ValueError("Invalid or duplicate no-fix evidence")
        reviewed[key] = entry
    errors = []
    for result in results:
        if (not isinstance(result.get("Target"), str)
                or any(result.get(k) is not None and not isinstance(result.get(k), list)
                       for k in ("Vulnerabilities", "Secrets"))):
            raise ValueError("Malformed scan result")
        if result.get("Secrets"):
            errors.append("secret findings remain")
        for finding in result.get("Vulnerabilities") or []:
            key = (finding.get("VulnerabilityID"), finding.get("PkgName"),
                   finding.get("InstalledVersion"), (finding.get("PkgIdentifier") or {}).get("PURL", ""))
            fixed = str(finding.get("FixedVersion") or "").strip()
            if key[1] == "linux-libc-dev":
                expected = {"package": "linux-libc-dev", "version": HEADER_VERSION,
                            "kernel_images": False, "kernel_modules": False}
                if boundary != expected:
                    raise ValueError("Missing exact installed header-only proof")
                if (not fixed and key[0] in headers.get("ids", [])
                        and key[2] == HEADER_VERSION and key[3] == HEADER_PURL
                        and (key[0] not in refreshed or finding.get("Severity") == "MEDIUM")):
                    continue
            severity = finding.get("Severity", "UNKNOWN")
            entry = reviewed.get(key)
            if (fixed or severity not in ("LOW", "MEDIUM")
                    or entry is None or entry["severity"] != severity):
                errors.append("%s %s@%s severity=%s fixed=%s" %
                              (key[0], key[1], key[2], severity, fixed or "none"))
    return errors


if __name__ == "__main__":
    # stdin is data, not a path. All file paths are repository-owned literals.
    root = Path(__file__).resolve().parents[2]
    ledger = json.loads((root / ".github/dapper-unfixed.json").read_text(encoding="utf-8"))
    headers = json.loads((root / ".github/dapper-headers.json").read_text(encoding="utf-8"))
    boundary = json.loads((root / "evidence/dapper-header-boundary.json").read_text(encoding="utf-8"))
    failures = violations(json.load(sys.stdin), ledger, boundary, headers)
    if failures:
        raise SystemExit("BUILDER_POLICY_FAILED\n" + "\n".join(failures))
    print("BUILDER_FULL_SEVERITY_OK fixed=0 unregistered=0 unknown=0 secrets=0")
