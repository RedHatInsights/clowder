Pull Request Flow
=================

Changes to the Clowder codebase can be broken down into three distinct categories. Each of these
is treated in a slightly different way with regards to signoff and review. The goal of this is to
reduce the size of pull requests to allow code to be merged faster and make production pushes less
dramatic.

* **Typo/Docs** No detailed explanation/justification needed

* **Functional Change** any significant modification to code that gets compiled (i.e. anything over
  typo/code style changes) requires a good commit message, detailing functions that have been altered,
  behaviour that has changed, etc, a set of functional tests added to the e2e suite, with unit tests
  optional, and should be reviewed by at least one Clowder core developer.

* **Architectural Change** anything more advanced than a functional change, which typically
  includes, any changes to API specs or changes to external behaviour that is observable by a
  customer, should have architect sign off, must be run locally to validate tests and behaviour, must
  include any deprecations, should have a design doc, and must be reviewed by two clowder core
  developers.

All PRs should be squashed to as few commits as makes sense to a) keep the version history clean
and b) assist with any reverts/cherrypicks that need to happen.
