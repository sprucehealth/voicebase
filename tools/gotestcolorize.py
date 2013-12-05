#!/usr/bin/env python

import os
import sys

PASS = "\033[0;32m"
FAIL = "\033[1;31m"
END = "\033[0m"

def output_run(run, success):
    for line in run:
        if success:
            sys.stdout.write(PASS)
        elif success == False:
            sys.stdout.write(FAIL)
        sys.stdout.write(line+"\n")
        sys.stdout.write(END)
    sys.stdout.flush()

run = []
success = None
for line in sys.stdin:
    line = line.strip()
    if line.startswith("=== RUN "):
        if run:
            output_run(run, success)
        run = []
        success = None
    elif line.startswith("--- PASS:"):
        success = True
    elif line.startswith("--- FAIL:"):
        success = False
    run.append(line)
if run:
    output_run(run, success)
