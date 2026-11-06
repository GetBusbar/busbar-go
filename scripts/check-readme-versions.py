#!/usr/bin/env python3
"""Fail if README.md's version claims have drifted from what this repo ships.

The README states the busbar OpenAPI `info.version` it was generated against by
hand, and it went stale silently once already (the README claimed `1.5.0` while
the repo was generated from `1.5.3`), because nothing checked it. This does.

Source of truth: openapi.json's info.version.

Unlike the Python and TypeScript SDKs, this one has NO in-repo manifest naming
the SDK's own version: a Go module's version is purely a git tag. Rather than
hardcode a number nothing can check, the README points at the releases page and
tells you to `go get ...@latest`, so there is nothing to go stale.

Two checks:
  1. The marker bullet in README.md must name exactly the spec version.
  2. Outside the `### History` section (where old version numbers are the
     point), no busbar-shaped `1.x.y` token may disagree with the spec version.
     Put `version-check: ignore` on a line to exempt it.
"""

import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]

SPEC_MARKER = "- **Generated from:** busbar OpenAPI `info.version` `{}`"

readme = (ROOT / "README.md").read_text(encoding="utf-8")
spec_version = json.loads((ROOT / "openapi.json").read_text(encoding="utf-8"))["info"]["version"]

failures = []

spec_line = SPEC_MARKER.format(spec_version)
if spec_line not in readme:
    failures.append(f"spec version: README.md is missing the exact line: {spec_line}")

# Everything from `### History` to the next same-or-higher heading is allowed to
# name older versions: that is what a history section is for.
body = re.sub(r"^### History\n(?:(?!^#{1,3} )[\s\S])*", "", readme, flags=re.MULTILINE)
for raw_line in body.split("\n"):
    if "version-check: ignore" in raw_line:
        continue
    for found in re.findall(r"\b1\.\d+\.\d+\b", raw_line):
        if found != spec_version:
            failures.append(
                f"README.md names busbar {found} outside the History section, "
                f"but this repo is generated from {spec_version}: {raw_line.strip()}"
            )

if failures:
    print("\n".join(f"::error::{f}" for f in failures), file=sys.stderr)
    sys.exit(1)

print(f"README version claims OK: busbar spec {spec_version}")
