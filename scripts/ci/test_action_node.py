import tempfile
import unittest
from pathlib import Path

from check_action_node import scan


class NodeActionPolicyTests(unittest.TestCase):
    def check(self, source: str, extra: dict[str, str] | None = None):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            file = root / ".github" / "workflows" / "ci.yml"
            file.parent.mkdir(parents=True)
            file.write_text(source)
            for path, text in (extra or {}).items():
                child = root / path
                child.parent.mkdir(parents=True, exist_ok=True)
                child.write_text(text)
            return scan(root)

    def test_flags_legacy_action_in_each_job(self):
        errors, _, jobs = self.check("jobs:\n  one:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n  two:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/setup-go@v5\n")
        self.assertEqual(jobs, 2)
        self.assertEqual(len(errors), 2)
        self.assertIn("job:one", errors[0])
        self.assertIn("job:two", errors[1])

    def test_accepts_node24_actions(self):
        errors, _, jobs = self.check("jobs:\n  build:\n    runs-on: blacksmith-4vcpu-ubuntu-2404\n    steps:\n      - uses: actions/checkout@v7\n      - uses: actions/setup-go@v6\n      - uses: actions/upload-artifact@v7\n")
        self.assertEqual(errors, [])
        self.assertEqual(jobs, 1)

    def test_checks_nested_local_composite(self):
        errors, _, _ = self.check("jobs:\n  check:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: ./.github/actions/local\n", {".github/actions/local/action.yml": "runs:\n  using: composite\n  steps:\n    - uses: actions/checkout@v4\n"})
        self.assertEqual(len(errors), 1)
        self.assertIn("action.yml", errors[0])

    def test_prohibits_insecure_runtime_override(self):
        errors, _, _ = self.check("env:\n  ACTIONS_ALLOW_USE_UNSECURE_NODE_VERSION: 'true'\njobs:\n  check:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v7\n")
        self.assertTrue(any("insecure" in e for e in errors))

    def test_unverified_third_party_is_warning_not_false_assurance(self):
        errors, warnings, _ = self.check("jobs:\n  check:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: vendor/custom@0123456789abcdef0123456789abcdef01234567\n")
        self.assertEqual(errors, [])
        self.assertEqual(len(warnings), 1)


if __name__ == "__main__":
    unittest.main()
