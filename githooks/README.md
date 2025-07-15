# Git Hooks

This directory contains git hooks for the Clowder project to ensure code quality and consistency.

## Available Hooks

### `pre-commit`
Runs before each commit to ensure code quality:
- **go fmt**: Checks code formatting
- **go vet**: Runs static analysis
- **goimports**: Checks import formatting (if available)
- **ineffassign**: Detects ineffectual assignments (if available)
- **go build**: Ensures code compiles

The hook only checks staged Go files and excludes generated files and third-party code.

### `prepush.sh`
Runs before each push to ensure comprehensive checks:
- Runs `make pre-push` which includes manifests generation, API docs, etc.

## Installation

### Automatic Installation
Run the install script from the project root:
```bash
./install.sh
```

This will copy all hooks to `.git/hooks/` and make them executable.

### Manual Installation
To install hooks manually:
```bash
# Make hooks executable
chmod +x githooks/*

# Copy to git hooks directory
cp githooks/pre-commit .git/hooks/pre-commit
cp githooks/prepush.sh .git/hooks/pre-push
```

## Installing Required Tools

For the best experience, install these linting tools:

```bash
# Install goimports
go install golang.org/x/tools/cmd/goimports@latest

# Install ineffassign
go install github.com/gordonklaus/ineffassign@latest
```

## Bypassing Hooks

If you need to bypass the pre-commit hook in exceptional cases:
```bash
git commit --no-verify -m "your commit message"
```

**Note:** Use this sparingly and ensure your code still meets quality standards.

## Troubleshooting

If the pre-commit hook fails:
1. Run the suggested commands to fix formatting/linting issues
2. Stage the fixed files: `git add .`
3. Retry the commit

For import issues, run:
```bash
goimports -w .
```

For formatting issues, run:
```bash
go fmt ./...
``` 