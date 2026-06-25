# Contributing

Thank you for your interest in contributing! This document outlines the process and guidelines for
contributing to this project.

## Getting Started

- Fork the repository
- Read the [project README][readme] and the [developer guide][developer-guide] for setup instructions
- Create a feature branch off `master` with a descriptive name

## Opening a Pull Request

All PRs should be squashed to as few commits as makes sense to keep the version history clean and
to assist with any reverts or cherry-picks.

Changes fall into three categories, each with different review requirements:

- **Typo/Docs** — no detailed explanation or justification needed
- **Functional Change** — requires a descriptive commit message detailing altered functions and
  changed behaviour, functional tests added to the E2E suite, and review by at least one Clowder
  core developer
- **Architectural Change** — requires architect sign-off, local validation of tests and behaviour,
  documentation of any deprecations, a design doc, and review by two Clowder core developers

For full details on the PR flow see [docs/contributing.md][contributing-docs].

## Commit Messages

Clowder uses [Conventional Commits][conventional-commits-spec] as a mandatory pipeline step.
Commits must follow this format:

```text
<purpose>(<scope>): <description>

Body explaining what changed and why.
```

Accepted `purpose` values: `fix`, `feat`, `build`, `chore`, `ci`, `docs`, `style`, `refactor`,
`perf`, `test`.

Use `!` after the purpose/scope to denote a breaking change (API or `cdappconfig.json` spec
changes). `scope` is optional but encouraged for provider-level changes (e.g., `feat(database):`).

## Signing Commits

All commits must be signed with a GPG or SSH key.

```sh
git config --global commit.gpgSign true
```

For setup instructions see the [git-commit signing documentation][git-commit-signing].

## AI-Assisted Commit Messages

If you use an AI tool to generate or refine your commit message:

1. **Author responsibility:** Read, understand, and edit the message before committing. The final
   message is your responsibility.
1. **Disclose the tool:** Add a `Co-authored-by:` trailer to the commit:

   ```text
   Co-authored-by: Claude Sonnet 4.6 <noreply@anthropic.com>
   ```

## Testing

Run `make pre-push` before opening a PR — it runs formatting, vetting, unit tests, and
regenerates all generated files. See [docs/contributing.md][contributing-docs] for unit test and
KUTTL E2E test development patterns.

## Code Review

- Respond to feedback promptly
- Make requested changes in new commits (avoid force-pushing unless asked)
- Discuss design decisions and tradeoffs openly

Thank you for contributing!

[readme]: ./README.md
[developer-guide]: docs/developer-guide.md
[contributing-docs]: docs/contributing.md
[conventional-commits-spec]: https://www.conventionalcommits.org/
[git-commit-signing]: https://git-scm.com/docs/git-commit#Documentation/git-commit.txt--S
