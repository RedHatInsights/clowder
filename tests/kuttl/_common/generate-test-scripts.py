#!/usr/bin/env python3
"""
Script to convert json-assert yaml files to use shell scripts with error handling.

This script:
1. Parses existing json-assert yaml files
2. Extracts the commands exactly as they are
3. Wraps them in a shell script with error handling
4. Updates the yaml file to call the shell script
"""

import os
import re
import yaml
from pathlib import Path

def extract_namespace_from_commands(commands):
    """Extract namespace from commands to set up error handling."""
    for cmd in commands:
        script = cmd.get('script', '')

        # Extract namespace from kubectl commands
        ns_match = re.search(r'--namespace[= ](\S+)', script)
        if ns_match:
            return ns_match.group(1)

        ns_match = re.search(r'-n\s+(\S+)', script)
        if ns_match:
            return ns_match.group(1)

    return None

def generate_shell_script(test_dir, yaml_file, commands):
    """Generate shell script from commands."""
    test_name = test_dir.name
    namespace = extract_namespace_from_commands(commands) or test_name

    # Build the script header with error handling
    script_lines = [
        "#!/bin/bash",
        "",
        "# Source common error handling",
        'source "$(dirname "$0")/../_common/error-handler.sh"',
        "",
        "# Setup error handling",
        f'setup_error_handling "{test_name}" "{namespace}"',
        "",
        "# Test commands from original yaml file",
    ]

    # Add all commands exactly as they are
    for cmd in commands:
        script = cmd.get('script', '').strip()
        if script:
            script_lines.append(script)

    return "\n".join(script_lines)

def process_yaml_file(yaml_path):
    """Process a single json-assert yaml file."""
    print(f"Processing {yaml_path}")

    with open(yaml_path, 'r') as f:
        data = yaml.safe_load(f)

    if not data or data.get('kind') != 'TestStep':
        print(f"  Skipping - not a TestStep")
        return

    commands = data.get('commands', [])
    if not commands:
        print(f"  Skipping - no commands")
        return

    # Determine script name based on yaml file name
    yaml_name = yaml_path.stem  # e.g., "02-json-asserts"
    script_name = f"{yaml_name}.sh"

    # Generate shell script
    test_dir = yaml_path.parent
    shell_script = generate_shell_script(test_dir, yaml_path, commands)

    # Write shell script
    script_path = test_dir / script_name
    with open(script_path, 'w') as f:
        f.write(shell_script)

    print(f"  Created {script_path}")

    # Update yaml file
    new_yaml = {
        'apiVersion': 'kuttl.dev/v1beta1',
        'kind': 'TestStep',
        'commands': [
            {'script': f'bash {script_name}'}
        ]
    }

    with open(yaml_path, 'w') as f:
        f.write("---\n")
        yaml.dump(new_yaml, f, default_flow_style=False, sort_keys=False)

    print(f"  Updated {yaml_path}")

def main():
    """Main function."""
    # Find all json-assert yaml files
    kuttl_dir = Path(__file__).parent.parent
    yaml_files = list(kuttl_dir.glob('*/*json-assert*.yaml'))

    # Sort for consistent processing
    yaml_files = sorted(set(yaml_files))

    # Skip the one we already did manually
    yaml_files = [f for f in yaml_files if 'test-basic-app/02-json-asserts.yaml' not in str(f)]

    print(f"Found {len(yaml_files)} json-assert files to process\n")

    for yaml_path in yaml_files:
        try:
            process_yaml_file(yaml_path)
        except Exception as e:
            print(f"  ERROR: {e}")
        print()

    print("Done!")

if __name__ == '__main__':
    main()
