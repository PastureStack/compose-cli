#!/usr/bin/env python3
"""Source-only bootstrap contract. Does not invoke APT, Docker or the network."""
from pathlib import Path
import re
import shlex
import unittest

ROOT = Path(__file__).resolve().parents[1]
TOOLS = {
    'bash', 'ca-certificates', 'curl', 'default-jre', 'gcc', 'git', 'gzip',
    'libc6-dev', 'libcurl3t64-gnutls', 'libcurl4t64', 'libssl3t64',
    'linux-libc-dev', 'make', 'openssl', 'openssl-provider-legacy', 'python3',
    'python3-virtualenv', 'rsync', 'tar', 'tox', 'xz-utils', 'zip',
}


def check_bootstrap(docker, lock, versions, gate):
    security_dates = re.findall(r'^# Official Ubuntu 26[.]04 security/updates snapshot: ([0-9]{8}T[0-9]{6}Z)$', lock, re.M)
    dates = re.findall(r"^UBUNTU_APT_SNAPSHOT='([0-9]{8}T[0-9]{6}Z)'$", versions, re.M)
    if len(dates) != 1 or len(security_dates) != 1 or dates != ['20261023T000000Z']:
        raise ValueError('one canonical snapshot required')
    if 'bootstrap_snapshot=${UBUNTU_APT_SNAPSHOT};' not in docker:
        raise ValueError('bootstrap must use its own immutable base lock')
    before, phase = docker.split('RUN set -eux;', 1)
    if 'COPY ubuntu-security-packages.lock /licenses/ubuntu-security-packages.lock' not in before:
        raise ValueError('lock must precede bootstrap')
    if "test \"${bootstrap_snapshot}\" = '" + dates[0] + "';" not in phase:
        raise ValueError('closed snapshot identity required')
    source = next(line for line in phase.splitlines() if "printf 'Types: deb" in line)
    if ('URIs: https://snapshot.ubuntu.com/ubuntu/%s\\n' not in source
            or 'Suites: resolute resolute-updates resolute-backports resolute-security\\n' not in source
            or 'Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg\\n' not in source
            or source.count('"${bootstrap_snapshot}"') != 1):
        raise ValueError('all bootstrap pockets must use the canonical signed source')
    checks = [shlex.split(line) for line in gate.splitlines()
              if line.startswith('require_line Dockerfile.dapper ') and 'URIs: https://snapshot.ubuntu.com/' in line]
    if checks != [['require_line', 'Dockerfile.dapper', source]]:
        raise ValueError('source gate must check the actual source command')
    update = phase.find('apt-get update --error-on=any;')
    install = phase.find('apt-get install -y --no-install-recommends')
    if not 0 <= update < install:
        raise ValueError('index failure must stop before installation')
    bindings = dict(re.findall(r'^\s+([a-z0-9+.-]+)="\$\{(UBUNTU_APT_[A-Z0-9_]+)\}"', phase, re.M))
    locked = dict(re.findall(r"^(UBUNTU_APT_[A-Z0-9_]+)='([^']+)'$", versions, re.M))
    if set(bindings) != TOOLS or not all(name in locked for name in bindings.values()):
        raise ValueError('all original compiler and compatibility tools must remain exactly pinned')
    if any(flag in phase for flag in ('--allow-downgrades', '--allow-unauthenticated', '--force-')):
        raise ValueError('solver and authentication must not be bypassed')


def check_overlay(installer, lock):
    dates = re.findall(r'^# Official Ubuntu 26[.]04 security/updates snapshot: ([0-9]{8}T[0-9]{6}Z)$', lock, re.M)
    if dates != ['20261105T000000Z'] or 'test "$security_snapshot" = 20261105T000000Z' not in installer:
        raise ValueError('only the reviewed overlay snapshot is permitted')
    parser = next(line for line in installer.splitlines() if line.startswith('security_snapshot='))
    if not parser.startswith("security_snapshot=$(sed -n 's/^# Official Ubuntu") or not parser.endswith(' "$lock")'):
        raise ValueError('snapshot must derive only from the supplied canonical lock')
    source = next(line for line in installer.splitlines() if "printf 'Types: deb" in line)
    if ('URIs: https://snapshot.ubuntu.com/ubuntu/%s\\n' not in source
            or 'Suites: resolute-updates resolute-security\\n' not in source
            or '"$security_snapshot"' not in source
            or 'Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg' not in source):
        raise ValueError('only the canonical signed overlay source is permitted')
    for required in ('apt_security update --error-on=any',
                     'apt_security install -y --only-upgrade --no-install-recommends --no-remove "$@"',
                     'cmp "$tmp/before" "$tmp/after"',
                     'dpkg --compare-versions "$installed" ge "$version" && continue',
                     'test "$status" = installed || continue',
                     '-o Acquire::https::Verify-Peer=true -o Acquire::https::Verify-Host=true',
                     '-o Acquire::AllowInsecureRepositories=false -o APT::Get::AllowUnauthenticated=false'):
        if required not in installer:
            raise ValueError('overlay package-set/authentication contract changed')
    if 'apt_security upgrade' in installer or '--allow-downgrades' in installer or '--force-' in installer:
        raise ValueError('blanket overlay promotion or solver bypass')


class DapperSnapshotTest(unittest.TestCase):
    def assert_pair(self, docker, pair_lock, lock):
        rows = [line.split() for line in pair_lock.splitlines()
                if line and not line.startswith('#')]
        self.assertEqual({(r[0], r[1]) for r in rows},
                         {(arch, pkg) for arch in ('amd64', 'arm64') for pkg in ('apt', 'libapt-pkg7.0')})
        self.assertEqual(len(rows), 4)
        for arch, package, version, size, sha in rows:
            self.assertEqual(version, '3.2.1ubuntu0.1')
            self.assertGreater(int(size), 1000000)
            self.assertRegex(sha, r'^[0-9a-f]{64}$')
            host = 'security.ubuntu.com/ubuntu' if arch == 'amd64' else 'ports.ubuntu.com/ubuntu-ports'
            line = ('ADD --checksum=sha256:' + sha + ' https://' + host + '/pool/main/a/apt/'
                    + package + '_' + version + '_' + arch + '.deb /tmp/' + package + '-' + arch + '.deb')
            self.assertIn(line, docker)
        phase = docker.split('RUN set -eux;', 1)[1].split('\nENV ', 1)[0]
        for required in ('native_arch=$(dpkg --print-architecture)',
                         'test "${native_arch}" = "${DAPPER_HOST_ARCH}"',
                         'case "${native_arch}" in amd64|arm64)',
                         "-f='${db:Status-Status}'", ' le "${version}"',
                         '"${deb}" Package', '"${deb}" Version', '"${deb}" Architecture',
                         "stat -c '%s'", 'sha256sum -c -', 'test "$#" -eq 2',
                         'dpkg -i "$@"', "-f='${Version}' apt", "-f='${Version}' libapt-pkg7.0"):
            self.assertIn(required, phase)
        self.assertNotIn('apt-get', phase)
        self.assertIn('\napt=3.2.1ubuntu0.1\n', lock)
        self.assertIn('\nlibapt-pkg7.0=3.2.1ubuntu0.1\n', lock)

    def test_official_apt_pair_is_architecture_bound_before_network_use(self):
        docker = (ROOT / 'Dockerfile.dapper').read_text()
        pair = (ROOT / 'ubuntu-apt-security.lock').read_text()
        lock = (ROOT / 'ubuntu-security-packages.lock').read_text()
        self.assert_pair(docker, pair, lock)
        cases = [docker.replace('sha256sum -c -;', 'true;'),
                 docker.replace('"${deb}" Architecture', '"${deb}" Package'),
                 docker.replace('test "$#" -eq 2;', 'test "$#" -ge 1;'),
                 docker.replace("-f='${db:Status-Status}'", "-f='${Version}'"),
                 docker.replace(' le "${version}"', ' ge "${version}"'),
                 docker.replace('native_arch=$(dpkg --print-architecture)', 'native_arch=amd64'),
                 docker.replace('ADD --checksum=sha256:0de729', 'ADD --checksum=sha256:9de729')]
        for index, changed in enumerate(cases):
            with self.subTest(drift=index), self.assertRaises(AssertionError):
                self.assert_pair(changed, pair, lock)

    def test_security_overlay_is_exact_only_upgrade_and_separate_from_bootstrap(self):
        installer = (ROOT / 'scripts/install-ubuntu-security-updates.sh').read_text()
        lock = (ROOT / 'ubuntu-security-packages.lock').read_text()
        check_overlay(installer, lock)
        cases = [installer.replace('apt_security update --error-on=any', 'apt_security update'),
                 installer.replace('--only-upgrade ', ''), installer.replace('--no-remove ', ''),
                 installer.replace('cmp "$tmp/before" "$tmp/after"', 'true'),
                 installer.replace('apt_security install -y', 'apt_security upgrade -y'),
                 installer.replace('"$security_snapshot" >', '"20261106T000000Z" >'),
                 installer.replace('"$lock")', '"/tmp/other.lock")')]
        for index, changed in enumerate(cases):
            with self.subTest(drift=index), self.assertRaises(ValueError):
                check_overlay(changed, lock)

    def test_existing_snapshot_preserves_exact_tools_and_fails_closed_on_drift(self):
        docker = (ROOT / 'Dockerfile.dapper').read_text()
        lock = (ROOT / 'ubuntu-security-packages.lock').read_text()
        versions = (ROOT / 'ubuntu-apt.lock').read_text()
        gate = (ROOT / 'scripts/check-pasturestack-source').read_text()
        check_bootstrap(docker, lock, versions, gate)
        cases = [
            (docker.replace('https://snapshot.ubuntu.com/ubuntu/%s', 'https://archive.ubuntu.com/ubuntu'), lock),
            (docker.replace('apt-get update --error-on=any;', 'apt-get update;'), lock),
            (re.sub(r'^\s+gcc="[^\n]+\n', '', docker, flags=re.M), lock),
            (docker.replace('"${bootstrap_snapshot}" \\', '"20261105T000000Z" \\'), lock),
            (docker, lock + '# Official Ubuntu 26.04 security/updates snapshot: 20261023T000000Z\n'),
        ]
        for changed, changed_lock in cases:
            with self.subTest(drift=cases.index((changed, changed_lock))):
                with self.assertRaises(ValueError):
                    check_bootstrap(changed, changed_lock, versions, gate)


if __name__ == '__main__':
    unittest.main()
