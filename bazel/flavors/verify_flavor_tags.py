"""Verify that bazel/flavors/defs.bzl is in sync with tasks/build_tags.py
and the flavorUnitTestTags literal in bazel/rules/dd_go_test/_gazelle_extension.go.

Run with: bazel test //bazel/flavors:verify_flavor_tags
"""

import importlib.util
import re
import sys
import types
import unittest
from pathlib import Path


def _load_tasks():
    """Load tasks/flavor.py and tasks/build_tags.py via importlib, bypassing tasks/__init__.py."""
    # Provide a minimal tasks package stub so build_tags.py can do
    # "from tasks.flavor import AgentFlavor" without executing __init__.py,
    # which would pull in dozens of unrelated modules.
    if "tasks" not in sys.modules:
        sys.modules["tasks"] = types.ModuleType("tasks")

    def _load(mod_name, rel_path):
        spec = importlib.util.spec_from_file_location(mod_name, Path(rel_path))
        mod = importlib.util.module_from_spec(spec)
        sys.modules[mod_name] = mod
        spec.loader.exec_module(mod)
        return mod

    flavor = _load("tasks.flavor", "tasks/flavor.py")
    bt = _load("tasks.build_tags", "tasks/build_tags.py")
    return flavor, bt


def _expected_tags(flavor_mod, bt_mod):
    result = {}
    for f in flavor_mod.AgentFlavor:
        tags = bt_mod.build_tags[f].get("unit-tests", set())
        result[f.name] = sorted(tags | bt_mod.COMMON_TAGS)
    return result


def _load_defs_bzl():
    ns = {}
    exec(Path("bazel/flavors/defs.bzl").read_text(), ns)  # noqa: S102
    return ns


def _load_go_extension_tags():
    """Parse the flavorUnitTestTags map literal from the Gazelle extension's Go source.

    The literal is a fixed shape (map[string][]string{ "name": {"tag", ...}, ... }),
    so a small regex pass is enough — no need to drag in a Go AST parser.
    """
    text = Path("bazel/rules/dd_go_test/_gazelle_extension.go").read_text()
    block_match = re.search(
        r"var\s+flavorUnitTestTags\s*=\s*map\[string\]\[\]string\{(.+?)^\}",
        text,
        flags=re.MULTILINE | re.DOTALL,
    )
    if not block_match:
        raise AssertionError("flavorUnitTestTags map not found in _gazelle_extension.go")
    body = block_match.group(1)
    result = {}
    for entry in re.finditer(r'"([^"]+)"\s*:\s*\{([^}]*)\}', body, flags=re.DOTALL):
        name = entry.group(1)
        tags = re.findall(r'"([^"]+)"', entry.group(2))
        result[name] = sorted(tags)
    return result


class TestFlavorTagsSync(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.flavor_mod, cls.bt_mod = _load_tasks()
        cls.expected = _expected_tags(cls.flavor_mod, cls.bt_mod)
        cls.defs = _load_defs_bzl()
        cls.go_tags = _load_go_extension_tags()

    def test_all_flavors_present(self):
        self.assertEqual(set(self.expected), set(self.defs["FLAVOR_UNIT_TEST_TAGS"]))

    def test_flavor_tag_sets(self):
        for flavor_name, expected_tags in self.expected.items():
            with self.subTest(flavor=flavor_name):
                actual = sorted(self.defs["FLAVOR_UNIT_TEST_TAGS"][flavor_name])
                self.assertEqual(expected_tags, actual)

    def test_linux_only_tags(self):
        expected = sorted(self.bt_mod.LINUX_ONLY_TAGS)
        actual = sorted(self.defs["LINUX_ONLY_TAGS"])
        self.assertEqual(expected, actual)

    def test_go_extension_flavor_tag_sets(self):
        for flavor_name, expected_tags in self.expected.items():
            with self.subTest(flavor=flavor_name):
                self.assertIn(flavor_name, self.go_tags, f"{flavor_name} missing from Go flavorUnitTestTags")
                self.assertEqual(expected_tags, self.go_tags[flavor_name])

    def test_go_extension_has_no_extra_flavors(self):
        self.assertEqual(set(self.expected), set(self.go_tags))


if __name__ == "__main__":
    unittest.main()
