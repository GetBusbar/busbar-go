#!/usr/bin/env python3
"""Refresh current README version claims from the bundled OpenAPI schema."""
import json
import pathlib
import re

root = pathlib.Path(__file__).resolve().parents[1]
version = json.loads((root / "openapi.json").read_text())["info"]["version"]
if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?", version):
    raise SystemExit("invalid OpenAPI version")
path = root / "README.md"
lines = path.read_text().splitlines(keepends=True)
history = False
for index, line in enumerate(lines):
    if line.startswith("### History"):
        history = True
    elif re.match(r"^#{1,3} ", line):
        history = False
    if not history and "version-check: ignore" not in line:
        lines[index] = re.sub(r"\b1\.\d+\.\d+\b", version, line)
path.write_text("".join(lines))
