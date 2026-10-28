# Change-aware development checks

Project-local Pi extension that chooses development checks from the current branch and working-tree changes.

## Usage

Reload Pi after checkout, then run:

```text
/check                 # choose recommended, all planned, or one check
/check all             # run every planned check
/check go-full         # run a planned check by id
```

The extension considers committed changes relative to `origin/main`, tracked working-tree changes, and untracked files. Depending on those paths it plans:

- formatting for changed Go files;
- tests for changed Go packages;
- all project-local Pi extension tests;
- recursive Terraform formatting checks;
- schema, mock, and documentation regeneration checks;
- the full Go test suite.

Checks run sequentially in a cancellable dashboard. Press Escape to terminate the current process group and skip pending checks. Full logs are written to the ignored `test_output/` directory. Failed-check output is bounded before being injected into the Pi session for diagnosis.

The generated-artifact check runs `make configure`. It can update stale generated files before failing on their resulting diff, leaving those updates available for review.

## Tests

```sh
node --test .pi/extensions/check-dashboard/check-core.test.ts
```
