#!/usr/bin/env python3
"""
Fix early exit 0 in json-assert shell scripts.

The issue: for loops like `for i in {1..10}; do kubectl ... && exit 0 || sleep 1; done`
will cause the entire script to exit when the condition succeeds, skipping remaining assertions.

The fix: Replace `&& exit 0` with `&& break` in for loops that are NOT wrapped in bash -c subshells.
"""

import re
from pathlib import Path

def fix_script(script_path):
    """Fix early exits in a script."""
    with open(script_path, 'r') as f:
        content = f.read()

    original_content = content
    lines = content.split('\n')
    fixed_lines = []
    modified = False

    for line in lines:
        # Check if this is a for loop with exit 0
        # Pattern: for i in {1..N}; do ... && exit 0 ...
        # BUT skip if it's inside bash -c '...' (these are subshells and are OK)

        if 'for i in {' in line and '&& exit 0' in line:
            # Check if it's wrapped in bash -c
            if line.strip().startswith('bash -c'):
                # This is a subshell, exit 0 only exits the subshell, keep as-is
                fixed_lines.append(line)
            else:
                # This is a regular for loop, replace exit 0 with break
                new_line = line.replace('&& exit 0', '&& break')
                fixed_lines.append(new_line)
                modified = True
                print(f"  Fixed: {script_path.name}")
                print(f"    Old: {line.strip()[:100]}...")
                print(f"    New: {new_line.strip()[:100]}...")
        else:
            fixed_lines.append(line)

    if modified:
        with open(script_path, 'w') as f:
            f.write('\n'.join(fixed_lines))
        return True
    return False

def main():
    """Main function."""
    kuttl_dir = Path(__file__).parent.parent
    script_files = list(kuttl_dir.glob('*/*json-assert*.sh'))

    print(f"Checking {len(script_files)} scripts for early exits...\n")

    fixed_count = 0
    for script_path in sorted(script_files):
        if fix_script(script_path):
            fixed_count += 1

    print(f"\nFixed {fixed_count} scripts")

if __name__ == '__main__':
    main()
