#!/usr/bin/env python3
"""Count SamNPlayer's lines (non-blank) for the README "In numbers" table.

Counts tracked files only (git ls-files). Generated and third-party files
are left out: Wails bindings, node_modules, vendor, dist, minified files.
Tests are Go *_test.go files and everything under a test/ or tests/ folder.

Run:  python3 scripts/count_lines.py
"""

import collections
import re
import subprocess

LANGS = {"go": "Go", "js": "JavaScript", "py": "Python", "css": "CSS", "html": "HTML",
         "sh": "Shell", "ps1": "PowerShell"}
SKIP = re.compile(r"(^|/)(node_modules|vendor|dist)/|\.min\.|/wailsjs/")


def is_test(path):
    name = path.rsplit("/", 1)[-1]
    return (path.endswith("_test.go") or name.startswith("test_")
            or "/test/" in "/" + path or "/tests/" in "/" + path)


def main():
    files = subprocess.run(["git", "ls-files"], capture_output=True, text=True, check=True).stdout.split("\n")
    code, test, count = collections.Counter(), collections.Counter(), collections.Counter()
    docs = 0
    for path in files:
        if not path or SKIP.search(path):
            continue
        ext = path.rsplit(".", 1)[-1] if "." in path else ""
        if ext not in LANGS and ext != "md":
            continue
        try:
            with open(path, encoding="utf-8", errors="ignore") as f:
                lines = sum(1 for line in f if line.strip())
        except OSError:
            continue
        if ext == "md":
            docs += lines
            continue
        (test if is_test(path) else code)[ext] += lines
        count[ext] += 1
    print(f"{'Language':<12}{'Files':>7}{'Code':>9}{'Tests':>9}")
    for ext, name in sorted(LANGS.items(), key=lambda kv: -(code[kv[0]] + test[kv[0]])):
        if count[ext]:
            print(f"{name:<12}{count[ext]:>7}{code[ext]:>9}{test[ext]:>9}")
    print(f"{'Total':<12}{sum(count.values()):>7}{sum(code.values()):>9}{sum(test.values()):>9}")
    print(f"Docs (Markdown): {docs} lines")


if __name__ == "__main__":
    main()
