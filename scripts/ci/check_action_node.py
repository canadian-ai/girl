#!/usr/bin/env python3
"""GIRL GitHub Actions Node runtime policy, no third-party dependencies.

Checks every workflow job independently and recursively checks referenced local
composite actions. Official actions have an audited Node 24 compatible minimum
major; unknown third-party actions are reported as unverified, not falsely
claimed to be Node 24 compatible.
"""
from __future__ import annotations

import argparse
from pathlib import Path
import re
import sys

# Minimum major verified to use the node24 JavaScript action runtime.
MINIMUM = {
    "actions/checkout": 5,
    "actions/setup-go": 6,
    "actions/setup-node": 5,
    "actions/upload-artifact": 7,
    "actions/download-artifact": 7,
    "actions/github-script": 8,
    "actions/cache": 5,
}
USE = re.compile(r"^\s*-?\s*uses:\s*['\"]?([^\s'\"#]+)", re.M)
JOB = re.compile(r"^  ([a-zA-Z][\w-]*):\s*(?:#.*)?$", re.M)
RUNS_ON = re.compile(r"^    runs-on:\s*(\S+)", re.M)
INSECURE = re.compile(r"(?m)^\s*ACTIONS_ALLOW_USE_UNSECURE_NODE_VERSION\s*:")
LEGACY = re.compile(r"(?m)^\s*(?:FORCE_JAVASCRIPT_ACTIONS_TO_NODE24)\s*:")
PIN = re.compile(r"^v(\d+)(?:\.|$)")
FIND_NODES = re.compile(r"(?m)^\s*node-version:\s*['\"]?(\d+)")


def scan(repo: Path) -> tuple[list[str], list[str], int]:
    errors: list[str] = []
    warnings: list[str] = []
    jobs_count = 0
    visited_actions: set[str] = set()

    def inspect_uses(content: str, label: str) -> None:
        for item in USE.finditer(content):
            target = item.group(1)
            if target.startswith("./"):
                folder = (repo / target).resolve()
                if not folder.is_relative_to(repo.resolve()):
                    errors.append(f"{label}: local action escapes repository: {target}")
                    continue
                key = str(folder)
                if key in visited_actions:
                    continue
                visited_actions.add(key)
                action = next((folder / name for name in ("action.yml", "action.yaml") if (folder / name).is_file()), None)
                if action is None:
                    errors.append(f"{label}: local action metadata missing: {target}")
                else:
                    local = action.read_text()
                    if INSECURE.search(local):
                        errors.append(f"{action.relative_to(repo)}: insecure Node runtime opt-out")
                    inspect_uses(local, str(action.relative_to(repo)))
                continue
            if target.startswith("docker://"):
                warnings.append(f"{label}: container action {target}, Node policy not applicable")
                continue
            if "@" not in target:
                errors.append(f"{label}: invalid action reference {target}")
                continue
            action_name, ref = target.rsplit("@", 1)
            if action_name in MINIMUM:
                match = PIN.match(ref)
                if not match:
                    errors.append(f"{label}: {target}: pin to an audited Node 24-compatible major (SHA must be reviewed separately)")
                elif int(match.group(1)) < MINIMUM[action_name]:
                    errors.append(f"{label}: {target} targets deprecated Node; use v{MINIMUM[action_name]} or newer")
            else:
                warnings.append(f"{label}: unverified third-party action {target}; audit runs.using and transitive composite steps")

    workflow_dir = repo / ".github" / "workflows"
    files = sorted([*workflow_dir.glob("*.yml"), *workflow_dir.glob("*.yaml")])
    if not files:
        return ["No GitHub Actions workflows found"], [], 0
    for path in files:
        content = path.read_text()
        name = str(path.relative_to(repo))
        if INSECURE.search(content):
            errors.append(f"{name}: insecure Node runtime opt-out is prohibited")
        if LEGACY.search(content):
            warnings.append(f"{name}: FORCE_JAVASCRIPT_ACTIONS_TO_NODE24 masks old action metadata; upgrade the action instead")
        if not re.search(r"(?m)^jobs:\s*$", content):
            errors.append(f"{name}: no jobs section found")
            continue
        job_text = content.split("\njobs:", 1)[-1]
        boundaries = list(JOB.finditer(job_text))
        if not boundaries:
            errors.append(f"{name}: cannot identify jobs")
            continue
        for index, match in enumerate(boundaries):
            job_id = match.group(1)
            chunk = job_text[match.end(): boundaries[index + 1].start() if index + 1 < len(boundaries) else len(job_text)]
            jobs_count += 1
            label = f"{name} job:{job_id}"
            if not RUNS_ON.search(chunk) and "uses:" not in chunk:
                errors.append(f"{label}: no runner or reusable workflow")
            inspect_uses(chunk, label)
            for major in FIND_NODES.findall(chunk):
                if int(major) < 24:
                    errors.append(f"{label}: setup-node uses Node {major}; require supported Node >=24")
    return errors, warnings, jobs_count


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    errors, warnings, jobs = scan(args.repo.resolve())
    print(f"GitHub Actions Node runtime audit: {jobs} jobs checked independently")
    for value in warnings:
        print(f"::warning::{value}")
    for value in errors:
        print(f"::error::{value}")
    print(f"Summary: {len(errors)} errors, {len(warnings)} follow-up reviews")
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
