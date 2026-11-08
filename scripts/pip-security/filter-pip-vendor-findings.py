#!/usr/bin/env python3
"""Apply narrowly proven pip-vendor VEX, retaining the original Trivy report."""
import argparse
from copy import deepcopy
from datetime import datetime, timezone
import json
from pathlib import Path
from vendor_source_inventory import SBOM_PATHS, WHEEL_PATHS

WHEEL_SHA = "2aef7c341eef310645ab56455f5789de2819463b25fa70151954e1d208d2db36"
WHEEL_NAME = "pip-26.2.2+pasturestack.2-py3-none-any.whl"
BOM_SHA = "59163489fd72f0732db7ef5d532b4ea83565d2113c9179fdfbdf67e6fddd5f6b"
RULES = {
    "GHSA-6v7p-g79w-8964": ("msgpack", "1.1.2"),
    "CVE-2026-57585": ("msgpack", "1.1.2"),
    "CVE-2026-114613": ("msgpack", "1.1.2"),
    "GHSA-j586-36cw-2gc2": ("msgpack", "1.1.2"),
    "CVE-2025-47273": ("setuptools", "70.3.0"),
    "CVE-2026-59890": ("setuptools", "70.3.0"),
    "GHSA-xmj3-9gf7-h52w": ("setuptools", "70.3.0"),
}


def apply(raw, proof, subject):
    inventory = proof.get("vendorSourceInventory", {})
    sources = inventory.get("sources", [])
    valid_sources = (sources and len({item.get("path") for item in sources}) == len(sources) and
                     any(item.get("kind") == "sbom" for item in sources) and
                     any(item.get("kind") == "wheel" and item.get("path") == proof.get("verifiedWheelPath") for item in sources) and
                     all((item.get("kind") == "wheel" and item.get("path") in WHEEL_PATHS and item.get("sha256") == WHEEL_SHA) or
                         (item.get("kind") == "sbom" and item.get("path") in SBOM_PATHS and item.get("sha256") == BOM_SHA)
                         for item in sources))
    valid = (proof["verifiedWheelSha256"] == WHEEL_SHA and
             proof.get("verifiedWheelPath") in WHEEL_PATHS and
             proof["pip"] == "26.2.2+pasturestack.2" and proof["urllib3"] == "2.8.0" and
             proof["msgpackImplementation"] == "pip._vendor.msgpack.fallback" and
             proof["setuptoolsPackageAbsent"] is True and proof["vendorFilesByteCompared"] == 299 and
             proof.get("vulnerableModulesAbsent") == ["msgpack._cmsgpack", "setuptools.wheel"] and
             inventory.get("complete") is True and inventory.get("noForeignMatchingDistribution") is True and
             inventory.get("expectedBomSha256") == BOM_SHA and valid_sources)
    if not valid:
        raise ValueError("Exact installed pip vendor proof is required")
    reasons = {item["id"]: item for item in proof["notAffected"]}
    result = deepcopy(raw)
    statements = []
    for section in result.get("Results", []):
        retained = []
        for item in section.get("Vulnerabilities", []):
            key = item.get("VulnerabilityID")
            expected = RULES.get(key)
            actual = (item.get("PkgName"), item.get("InstalledVersion"))
            purl = "pkg:pypi/" + str(actual[0]) + "@" + str(actual[1])
            packages = [pkg for pkg in section.get("Packages", []) if
                        (pkg.get("Name"), pkg.get("Version")) == actual]
            reason = reasons.get(key, {})
            eligible = (expected == actual and key in reasons and
                        (reason.get("package"), reason.get("version")) == actual and
                        section.get("Type") == "python-pkg" and packages and
                        all(pkg.get("AnalyzedBy") == "sbom" for pkg in packages) and
                        (item.get("PkgIdentifier") or {}).get("PURL") == purl and
                        all(not pkg.get("FilePath") and not pkg.get("PkgPath") for pkg in packages) and
                        not item.get("PkgPath"))
            if key == "CVE-2026-8643":
                # PYSEC-2026-196 is fixed in pip 26.1.2. Distro seed metadata
                # needs the complete installed core, not just a vendor proof.
                purl = "pkg:deb/ubuntu/python3-pip-whl@25.1.1%2Bdfsg-1ubuntu2?arch=all&distro=ubuntu-26.04"
                eligible = (actual == ("python3-pip-whl", "25.1.1+dfsg-1ubuntu2")
                            and section.get("Type") == "ubuntu" and section.get("Class") == "os-pkgs"
                            and packages and item.get("Severity") == "MEDIUM"
                            and item.get("FixedVersion") in (None, "", "26.1.2")
                            and (item.get("PkgIdentifier") or {}).get("PURL") == purl
                            and proof.get("vendorInventorySha256") == "6b9e911363afa748740d5a514de1ecf34974645d0fb01dac23546e66d4d0f94c"
                            and type(proof.get("pipCoreFilesByteCompared")) is int
                            and proof.get("pipCoreFilesByteCompared") == 157
                            and proof.get("pipCoreFileSetMatched") is True
                            and proof.get("pipCoreInventorySha256") == "51be05e205130ecb1655f762f5933177269a9e665d1f4075a8921557998ddce5")
            if not eligible:
                retained.append(item)
                continue
            detail = ("The distro seed has been replaced by the exact pip 26.2.2+pasturestack.2 wheel; "
                      "all 157 live core files and their full set match the patched wheel"
                      if key == "CVE-2026-8643" else reasons[key]["detail"])
            statements.append({
                "vulnerability": {"name": key},
                "products": [{"@id": subject, "subcomponents": [{"@id": purl}]}],
                "status": "not_affected", "justification": "vulnerable_code_not_present",
                "impact_statement": detail + "; exact pip wheel SHA-256 " + WHEEL_SHA,
            })
        section["Vulnerabilities"] = retained
    vex = {"@context": "https://openvex.dev/ns/v0.2.0", "@id": subject + "/pip-vendor-vex",
           "author": "PastureStack", "timestamp": datetime.now(timezone.utc).isoformat(),
           "version": 1, "statements": statements}
    return result, vex


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--raw", type=Path, required=True)
    parser.add_argument("--proof", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--vex", type=Path, required=True)
    parser.add_argument("--subject", required=True)
    args = parser.parse_args()
    report, vex = apply(json.loads(args.raw.read_text()), json.loads(args.proof.read_text()), args.subject)
    args.output.write_text(json.dumps(report, indent=2) + "\n")
    args.vex.write_text(json.dumps(vex, indent=2) + "\n")
    print(f"PROVEN_PIP_VENDOR_VEX statements={len(vex['statements'])}")
