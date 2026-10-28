"""Exercise upgrade runner sequencing and failure paths without real clusters."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


FAKE_TERRAFORM = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
args = sys.argv[1:]
if args == ['version', '-json']:
    print(json.dumps({'platform': 'linux_amd64'}))
    sys.exit(0)
stack = pathlib.Path(args.pop(0).split('=', 1)[1])
phase = 'branch' if '99.99.99' in (stack / 'versions.tf').read_text() else 'released'
command = args[0]
with open(os.environ['CALL_LOG'], 'a') as log:
    log.write(f'{phase} {" ".join(args)}\n')
assert os.environ['TF_CLI_CONFIG_FILE'].endswith(f'{phase}.tfrc')
assert not any(key.startswith('TF_CLI_ARGS') for key in os.environ)
assert 'TF_PLUGIN_CACHE_DIR' not in os.environ
failure = os.environ.get('FAIL_AT', '')
if command == 'init':
    if failure == 'branch-init' and phase == 'branch':
        sys.exit(1)
    (stack / '.terraform.lock.hcl').write_text(phase)
if command == 'apply' and phase == 'released':
    (stack / 'terraform.tfstate').write_text('{"resources": []}')
    if failure == 'released-apply':
        sys.exit(1)
if command == 'plan':
    if failure == f'{phase}-plan':
        sys.exit(int(os.environ.get('FAIL_CODE', '2')))
    if phase == 'branch' and '-out=upgrade.tfplan' in args:
        (stack / 'upgrade.tfplan').touch()
if command == 'apply' and phase == 'branch':
    assert args[-1] == 'upgrade.tfplan'
    assert (stack / 'upgrade.tfplan').exists()
if command == 'destroy' and failure == 'destroy':
    sys.exit(1)
'''


class UpgradeRunnerTest(unittest.TestCase):
    def run_upgrade(self, failure='', code='2'):
        # Copy only runner inputs so tests never create files in the real .work.
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            suite = root / 'test/e2e'
            source = Path(__file__).resolve().parent
            suite.mkdir(parents=True)
            shutil.copy(source / 'upgrade.sh', suite)
            shutil.copytree(source / 'stack', suite / 'stack')
            shutil.copytree(source / 'flavours', suite / 'flavours')
            tools = root / 'tools'
            tools.mkdir()
            terraform = tools / 'terraform'
            terraform.write_text(FAKE_TERRAFORM)
            terraform.chmod(0o755)
            binary = root / 'provider'
            binary.write_text('#!/bin/sh\nexit 0\n')
            binary.chmod(0o755)
            log = root / 'calls.log'
            env = {
                **os.environ,
                'PATH': f'{tools}:{os.environ["PATH"]}',
                'UPGRADE_FROM_VERSION': '0.8.0',
                'UPGRADE_PROVIDER_BINARY': str(binary),
                'CALL_LOG': str(log),
                'FAIL_AT': failure,
                'FAIL_CODE': code,
                'TF_CLI_ARGS_plan': '-refresh=false',
                'TF_PLUGIN_CACHE_DIR': '/nonexistent-cache',
            }
            result = subprocess.run(
                ['bash', str(suite / 'upgrade.sh')], env=env,
                capture_output=True, text=True, timeout=30,
            )
            return result, log.read_text().splitlines()

    def test_success_applies_only_verified_saved_plan(self):
        result, calls = self.run_upgrade()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(calls, [
            'released init -no-color',
            'released apply -no-color -auto-approve',
            'released plan -no-color -detailed-exitcode',
            'branch init -no-color -upgrade',
            'branch plan -no-color -detailed-exitcode -out=upgrade.tfplan',
            'branch apply -no-color -auto-approve upgrade.tfplan',
            'branch plan -no-color -detailed-exitcode',
            'branch destroy -no-color -auto-approve',
        ])

    def test_diff_or_error_fails_without_applying_branch(self):
        for code in ('1', '2'):
            with self.subTest(code=code):
                result, calls = self.run_upgrade('branch-plan', code)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(c.startswith('branch apply') for c in calls))
                self.assertEqual(calls[-1], 'branch destroy -no-color -auto-approve')

    def test_baseline_diff_does_not_upgrade(self):
        result, calls = self.run_upgrade('released-plan')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(c.startswith('branch') for c in calls))
        self.assertEqual(calls[-1], 'released destroy -no-color -auto-approve')

    def test_failed_create_still_destroys(self):
        result, calls = self.run_upgrade('released-apply')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(calls[-1], 'released destroy -no-color -auto-approve')

    def test_failed_branch_init_cleans_up_with_release(self):
        result, calls = self.run_upgrade('branch-init')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(calls[-1], 'released destroy -no-color -auto-approve')

    def test_destroy_failure_fails_test(self):
        result, _ = self.run_upgrade('destroy')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('cleanup failed', result.stderr)


if __name__ == '__main__':
    unittest.main()
