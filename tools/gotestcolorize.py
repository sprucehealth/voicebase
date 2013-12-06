#!/usr/bin/env python

import sys

COLOR_BRIGHT_WHITE = "\033[1;97m"
COLOR_BOLD_RED = "\033[1;31m"
COLOR_GREEN = "\033[0;32m"
COLOR_NORMAL = "\033[0m"
CHECK_MARK = "\xe2\x9c\x93"

status = 0
for line in sys.stdin:
    if line.startswith("=== RUN "):
        sys.stdout.write("\n")
        sys.stdout.write(COLOR_BRIGHT_WHITE)
    elif line.startswith("--- PASS:") or line.startswith("PASS"):
        sys.stdout.write(COLOR_GREEN)
    elif line.startswith("--- FAIL:") or line.startswith("FAIL"):
        sys.stdout.write(COLOR_BOLD_RED)
        status = 1
    sys.stdout.write(line)
    sys.stdout.write(COLOR_NORMAL)
    sys.stdout.flush()

sys.exit(status)
