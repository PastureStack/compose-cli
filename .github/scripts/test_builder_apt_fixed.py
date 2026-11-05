import importlib.util
import json
from copy import deepcopy
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("apt_fixed", Path(__file__).with_name("check_builder_apt_fixed.py"))
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)


class AptFixedTest(unittest.TestCase):
    def setUp(self):
        self.vex = json.loads((gate.ROOT / "security/dapper-apt.openvex.json").read_text())
        self.raw = {"SchemaVersion": 2, "Results": [{"Target": "ubuntu", "Class": "os-pkgs", "Type": "ubuntu",
            "Packages": [{"Name": name, "Version": gate.VERSION, "Identifier": {"PURL": purl}}
                         for name, purl in gate.PURLS.items()],
            "Vulnerabilities": [{"VulnerabilityID": gate.CVE, "PkgName": name,
                "InstalledVersion": gate.VERSION, "Severity": "HIGH", "PkgIdentifier": {"PURL": purl}}
                for name, purl in gate.PURLS.items()]}]}
        self.native = deepcopy(self.raw)
        self.native["Results"][0]["Vulnerabilities"] = []
        self.dpkg = "".join(name + "\t" + gate.VERSION + "\n" for name in gate.PURLS)

    def check(self):
        return gate.validate(self.raw, self.native, self.dpkg, self.vex)

    def test_native_fixed_pair_and_unrelated_findings_are_exact(self):
        finding = {"VulnerabilityID": "UNRELATED", "Severity": "HIGH"}
        for report in (self.raw, self.native):
            report["Results"][0]["Vulnerabilities"].append(deepcopy(finding))
        self.assertEqual(self.check()["fixedOccurrences"], 2)
        self.native["Results"][0]["Packages"][0]["Identifier"]["UID"] = "new-native-scan"
        self.native["Results"][0]["Packages"][0]["ID"] = "apt@" + gate.VERSION
        self.assertEqual(self.check()["fixedOccurrences"], 2)
        # A refreshed database need not repeat a vendor-fixed finding.
        self.raw["Results"][0]["Vulnerabilities"] = [finding]
        self.assertEqual(self.check()["fixedOccurrences"], 0)

    def test_status_identity_scope_and_unrelated_delta_fail_closed(self):
        original = deepcopy(self.vex)
        for field, value in (("status", "not_affected"), ("vulnerability", {"name": "OTHER"}),
                             ("products", [{"@id": "pkg:deb/ubuntu/apt@3.2.0?arch=amd64&distro=ubuntu-26.04"}]),
                             ("products", [{"@id": gate.PURLS["apt"].replace("amd64", "arm64")}]),
                             ("products", [{"@id": gate.PURLS["apt"].replace("ubuntu-26.04", "ubuntu-24.04")}]),
                             ("products", [{"@id": "pkg:deb/ubuntu/*"}])):
            self.vex = deepcopy(original)
            self.vex["statements"][0][field] = value
            with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                self.check()
        self.vex = original
        self.dpkg = self.dpkg.replace(gate.VERSION, "3.2.0", 1)
        with self.assertRaises(ValueError): self.check()
        self.setUp()
        self.raw["Results"][0]["Packages"][0]["Version"] = "3.2.0"
        with self.assertRaises(ValueError): self.check()
        self.setUp()
        self.native["Results"][0]["Vulnerabilities"] = [deepcopy(self.raw["Results"][0]["Vulnerabilities"][0])]
        with self.assertRaises(ValueError): self.check()
        self.setUp()
        self.raw["Results"][0]["Vulnerabilities"].append({"VulnerabilityID": "OTHER", "Severity": "CRITICAL"})
        with self.assertRaises(ValueError): self.check()
        self.setUp()
        self.raw["Results"][0]["Secrets"] = [{"RuleID": "secret"}]
        with self.assertRaises(ValueError): self.check()


if __name__ == "__main__":
    unittest.main()
