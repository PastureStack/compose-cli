#!/usr/bin/env python3
"""Negative controls for the exact pip-vendor VEX boundary."""
from copy import deepcopy
import ast
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile
from vendor_source_inventory import inventory, matching_sbom

spec = importlib.util.spec_from_file_location("pip_vex", Path(__file__).with_name("filter-pip-vendor-findings.py"))
vex = importlib.util.module_from_spec(spec)
spec.loader.exec_module(vex)


class VexBoundary(unittest.TestCase):
    def setUp(self):
        self.proof = {"verifiedWheelSha256": vex.WHEEL_SHA, "verifiedWheelPath": "/usr/share/python-wheels/" + vex.WHEEL_NAME,
                      "pip": "26.2.2+pasturestack.2",
                      "urllib3": "2.8.0", "msgpackImplementation": "pip._vendor.msgpack.fallback",
                      "setuptoolsPackageAbsent": True, "vendorFilesByteCompared": 299,
                      "vulnerableModulesAbsent": ["msgpack._cmsgpack", "setuptools.wheel"],
                      "vendorSourceInventory": {
                          "complete": True, "noForeignMatchingDistribution": True, "expectedBomSha256": vex.BOM_SHA,
                          "sources": [{"kind": "wheel", "path": "/usr/share/python-wheels/" + vex.WHEEL_NAME, "sha256": vex.WHEEL_SHA},
                                      {"kind": "sbom", "path": "/opt/python-security/pip-vendor.cdx.json", "sha256": vex.BOM_SHA}]},
                      "notAffected": [{"id": key, "package": value[0], "version": value[1],
                                       "detail": "Test fixture only"} for key, value in vex.RULES.items()]}
        self.raw = {"Results": [{"Type": "python-pkg", "Packages": [
            {"Name": name, "Version": version, "AnalyzedBy": "sbom"}
            for name, version in set(vex.RULES.values())], "Vulnerabilities": [
            {"VulnerabilityID": key, "PkgName": value[0], "InstalledVersion": value[1],
             "PkgIdentifier": {"PURL": "pkg:pypi/" + value[0] + "@" + value[1]}}
            for key, value in vex.RULES.items()]}]}

    def test_exact_slice_only(self):
        original = deepcopy(self.raw)
        result, document = vex.apply(self.raw, self.proof, "urn:test:exact-image")
        self.assertEqual(result["Results"][0]["Vulnerabilities"], [])
        self.assertEqual(len(document["statements"]), len(vex.RULES))
        self.assertEqual(self.raw, original, "raw evidence must not be mutated")

    def test_invalid_proof_fails_closed(self):
        for key, value in [("verifiedWheelSha256", "0" * 64), ("urllib3", "2.7.0"),
                           ("vendorFilesByteCompared", 300), ("setuptoolsPackageAbsent", False),
                           ("verifiedWheelPath", "/tmp/unverified/" + vex.WHEEL_NAME),
                           ("pip", "26.2.2"), ("vulnerableModulesAbsent", [])]:
            proof = dict(self.proof, **{key: value})
            with self.subTest(key=key), self.assertRaises(ValueError):
                vex.apply(self.raw, proof, "urn:test:image")

    def test_full_distribution_is_not_excluded(self):
        for package in self.raw["Results"][0]["Packages"]:
            package["AnalyzedBy"] = "python-pkg"
        result, document = vex.apply(self.raw, self.proof, "urn:test:image")
        self.assertEqual(len(result["Results"][0]["Vulnerabilities"]), len(vex.RULES))
        self.assertEqual(document["statements"], [])

    def test_missing_packages_is_not_excluded(self):
        self.raw["Results"][0]["Packages"] = []
        result, _ = vex.apply(self.raw, self.proof, "urn:test:image")
        self.assertEqual(len(result["Results"][0]["Vulnerabilities"]), len(vex.RULES))

    def test_wrong_version_is_not_excluded(self):
        for finding in self.raw["Results"][0]["Vulnerabilities"]:
            finding["InstalledVersion"] = "999.0"
        result, document = vex.apply(self.raw, self.proof, "urn:test:image")
        self.assertEqual(len(result["Results"][0]["Vulnerabilities"]), len(vex.RULES))
        self.assertEqual(document["statements"], [])

    def test_unknown_advisory_and_urllib3_are_retained(self):
        unknown = {"VulnerabilityID": "CVE-TEST-UNKNOWN", "PkgName": "setuptools", "InstalledVersion": "70.3.0"}
        urllib = {"VulnerabilityID": "CVE-2026-97689", "PkgName": "urllib3", "InstalledVersion": "2.7.0"}
        self.raw["Results"][0]["Vulnerabilities"] += [unknown, urllib]
        result, _ = vex.apply(self.raw, self.proof, "urn:test:image")
        self.assertEqual(result["Results"][0]["Vulnerabilities"], [unknown, urllib])

    def test_explicit_distribution_path_cannot_inherit_vendor_proof(self):
        for item in self.raw["Results"][0]["Vulnerabilities"]:
            item["PkgPath"] = "/unrelated/site-packages/" + item["PkgName"]
        result, document = vex.apply(self.raw, self.proof, "urn:test:image")
        self.assertEqual(len(result["Results"][0]["Vulnerabilities"]), len(vex.RULES))
        self.assertEqual(document["statements"], [])

    def test_wrong_purl_cannot_inherit_vendor_proof(self):
        for item in self.raw["Results"][0]["Vulnerabilities"]:
            item["PkgIdentifier"]["PURL"] = "pkg:generic/foreign@1"
        result, document = vex.apply(self.raw, self.proof, "urn:test:image")
        self.assertEqual(len(result["Results"][0]["Vulnerabilities"]), len(vex.RULES))
        self.assertEqual(document["statements"], [])

    def test_missing_or_unrecognized_source_inventory_fails_closed(self):
        for change in [None, {"complete": False}, {"sources": []}]:
            proof = deepcopy(self.proof)
            proof["vendorSourceInventory"] = {} if change is None else dict(proof["vendorSourceInventory"], **change)
            with self.subTest(change=change), self.assertRaises(ValueError):
                vex.apply(self.raw, proof, "urn:test:image")
        proof = deepcopy(self.proof)
        proof["vendorSourceInventory"]["sources"][0]["path"] = "/unrelated/" + vex.WHEEL_NAME
        with self.assertRaises(ValueError):
            vex.apply(self.raw, proof, "urn:test:image")

    def test_vulnerable_module_presence_fails_actual_proof_check(self):
        source = Path(__file__).with_name("check-pip-vendor-slice.py").read_text()
        tree = ast.parse(source)
        function = next(node for node in tree.body if isinstance(node, ast.FunctionDef)
                        and node.name == "require_absent_vulnerable_modules")
        namespace = {}
        exec(compile(ast.Module(body=[function], type_ignores=[]), "<actual-vendor-proof>", "exec"), namespace)
        check = namespace["require_absent_vulnerable_modules"]
        check(["pip/_vendor/msgpack/fallback.py", "pip/_vendor/pkg_resources/__init__.py"])
        for path in ["pip/_vendor/msgpack/_cmsgpack.so", "pip/_vendor/msgpack/_cmsgpack.pyd",
                     "pip/_vendor/setuptools/wheel.py"]:
            with self.subTest(path=path), self.assertRaises(AssertionError):
                check(name for name in [path])

    def test_actual_source_inventory_rejects_foreign_sbom_distribution_and_wheel(self):
        bom = json.dumps({"bomFormat": "CycloneDX", "components": [
            {"name": "msgpack", "version": "1.1.2"}, {"name": "setuptools", "version": "70.3.0"}]}).encode()
        with tempfile.TemporaryDirectory(prefix="pip-source-inventory-") as directory:
            root = Path(directory)
            known = root / "opt/python-security/pip-vendor.cdx.json"
            known.parent.mkdir(parents=True); known.write_bytes(bom)
            wheel = root / ("usr/share/python-wheels/" + vex.WHEEL_NAME)
            wheel.parent.mkdir(parents=True)
            with zipfile.ZipFile(wheel, "w") as archive:
                archive.writestr("pip/_vendor/bom.cdx.json", bom)
            wheel_sha = hashlib.sha256(wheel.read_bytes()).hexdigest()
            bom_sha = hashlib.sha256(bom).hexdigest()
            self.assertEqual(len(inventory(root, wheel_sha, bom_sha)["sources"]), 2)
            bad_inputs = {
                "foreign/bom.cdx.json": bom,
                "foreign/sbom.spdx.json": json.dumps({"spdxVersion": "SPDX-2.3", "packages": [
                    {"name": "setuptools", "versionInfo": "70.3.0"}]}).encode(),
                "outside-sys-path/msgpack.dist-info/METADATA": b"Name: msgpack\nVersion: 1.1.2\n",
                "foreign/copy.whl": wheel.read_bytes(),
            }
            for name, content in bad_inputs.items():
                path = root / name; path.parent.mkdir(parents=True, exist_ok=True); path.write_bytes(content)
                with self.subTest(path=name), self.assertRaises(ValueError):
                    inventory(root, wheel_sha, bom_sha)
                path.unlink()
            known.write_bytes(bom + b" ")
            with self.assertRaises(ValueError):
                inventory(root, wheel_sha, bom_sha)

    def test_unrelated_or_versionless_sbom_components_do_not_match(self):
        unrelated = {"spdxVersion": "SPDX-2.3", "packages": [
            {"name": "other", "externalRefs": [{"referenceLocator": "pkg:pypi/other@1.0"}]}]}
        self.assertFalse(matching_sbom(json.dumps(unrelated).encode()))
        versionless = {"bomFormat": "CycloneDX", "components": [{"name": "other"}, {}, {"name": "msgpack"}]}
        self.assertFalse(matching_sbom(json.dumps(versionless).encode()))
        primary = {"bomFormat": "CycloneDX", "metadata": {"component": {"name": "msgpack", "version": "1.1.2"}}}
        self.assertTrue(matching_sbom(json.dumps(primary).encode()))

    def test_pip_core_proof_compares_complete_set_and_every_file(self):
        spec = importlib.util.spec_from_file_location("core_proof", Path(__file__).with_name("check-pip-vendor-slice.py"))
        checker = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(checker)
        with tempfile.TemporaryDirectory() as directory:
            live = Path(directory) / "pip"
            live.mkdir()
            (live / "__init__.py").write_bytes(b"core fixture\n")
            selected = Path(directory) / "selected.whl"
            with zipfile.ZipFile(selected, "w") as archive:
                archive.writestr("pip/__init__.py", b"core fixture\n")
            with zipfile.ZipFile(selected) as archive:
                self.assertEqual(checker.require_pip_core_exact(archive, live)["pipCoreFilesByteCompared"], 1)
                (live / "__init__.py").write_bytes(b"changed\n")
                with self.assertRaises(AssertionError): checker.require_pip_core_exact(archive, live)
                (live / "__init__.py").write_bytes(b"core fixture\n")
                (live / "extra.py").write_bytes(b"unexpected\n")
                with self.assertRaises(AssertionError): checker.require_pip_core_exact(archive, live)
                (live / "extra.py").unlink()
                (live / "__init__.py").unlink()
                with self.assertRaises(AssertionError): checker.require_pip_core_exact(archive, live)

    def test_pip_8643_requires_exact_native_tuple_and_complete_patched_core(self):
        purl = "pkg:deb/ubuntu/python3-pip-whl@25.1.1%2Bdfsg-1ubuntu2?arch=all&distro=ubuntu-26.04"
        finding = {"VulnerabilityID": "CVE-2026-8643", "PkgName": "python3-pip-whl",
                   "InstalledVersion": "25.1.1+dfsg-1ubuntu2", "Severity": "MEDIUM",
                   "FixedVersion": "", "PkgIdentifier": {"PURL": purl}}
        raw = {"Results": [{"Type": "ubuntu", "Class": "os-pkgs",
                "Packages": [{"Name": "python3-pip-whl", "Version": "25.1.1+dfsg-1ubuntu2"}],
                "Vulnerabilities": [finding]}]}
        proof = dict(self.proof, vendorInventorySha256="6b9e911363afa748740d5a514de1ecf34974645d0fb01dac23546e66d4d0f94c",
                     pipCoreFilesByteCompared=157, pipCoreFileSetMatched=True,
                     pipCoreInventorySha256="51be05e205130ecb1655f762f5933177269a9e665d1f4075a8921557998ddce5")
        original = deepcopy(raw)
        for fixed in (None, "", "26.1.2"):
            raw["Results"][0]["Vulnerabilities"][0]["FixedVersion"] = fixed
            result, statements = vex.apply(raw, proof, "urn:test:fixture-image")
            self.assertEqual(result["Results"][0]["Vulnerabilities"], [])
            self.assertEqual(len(statements["statements"]), 1)
        self.assertEqual(original["Results"][0]["Vulnerabilities"][0]["Severity"], "MEDIUM")
        for key, value in [("pipCoreFilesByteCompared", 156), ("pipCoreFileSetMatched", False),
                           ("pipCoreInventorySha256", "0" * 64), ("vendorInventorySha256", "0" * 64),
                           ("pipCoreFilesByteCompared", None)]:
            with self.subTest(proof=key):
                result, statements = vex.apply(original, dict(proof, **{key: value}), "urn:test:fixture-image")
                self.assertEqual(result, original)
                self.assertEqual(statements["statements"], [])
        for key, value in [("Severity", "HIGH"), ("FixedVersion", "26.1.3"),
                           ("VulnerabilityID", "CVE-UNKNOWN"), ("InstalledVersion", "25.1.2"),
                           ("PkgIdentifier", {"PURL": purl + "&changed=1"})]:
            changed = deepcopy(original); changed["Results"][0]["Vulnerabilities"][0][key] = value
            result, statements = vex.apply(changed, proof, "urn:test:fixture-image")
            self.assertEqual(result, changed)
            self.assertEqual(statements["statements"], [])
        for key, value in [("Type", "python-pkg"), ("Class", "lang-pkgs"), ("Packages", [])]:
            changed = deepcopy(original); changed["Results"][0][key] = value
            result, statements = vex.apply(changed, proof, "urn:test:fixture-image")
            self.assertEqual(result, changed)
            self.assertEqual(statements["statements"], [])
        for key in ("pipCoreFilesByteCompared", "pipCoreFileSetMatched", "pipCoreInventorySha256"):
            proof.pop(key)
        result, statements = vex.apply(original, proof, "urn:test:old-vendor-only-proof")
        self.assertEqual(result, original)
        self.assertEqual(statements["statements"], [])

    def test_source_walk_error_cannot_produce_complete_inventory(self):
        def unreadable(*args, **kwargs):
            kwargs["onerror"](PermissionError("unreadable source path"))
        with patch("vendor_source_inventory.os.walk", unreadable), self.assertRaises(PermissionError):
            inventory(Path("/"), vex.WHEEL_SHA, vex.BOM_SHA)


if __name__ == "__main__":
    unittest.main()
