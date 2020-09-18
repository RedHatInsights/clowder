#!/usr/bin/python3

import sys
import re

if len(sys.argv) < 2:
    print("Please provide version")
    sys.exit(1)

current = sys.argv[1]

if not re.match(r"\d\.\d\.\d", current):
    print("Invalid symantic version")
    sys.exit(1)

major, minor, micro = (int(i) for i in current.split("."))

micro = micro - 1

print(".".join(str(i) for i in (major, minor, micro)))
