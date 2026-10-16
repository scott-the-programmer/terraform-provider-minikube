# Minikube E2E dashboard

Project-local Pi extension for running `test/e2e/run.sh` interactively.

## Usage

Reload Pi after checkout, then run:

```text
/e2e
/e2e docker
/e2e docker podman
```

The command:

- discovers driver flavours and checks their executable/health requirements;
- remembers the last flavour and options under the ignored `test/e2e/.work/` directory;
- configures `--keep`, `--skip-build`, and `--preload-image`;
- streams a cancellable dashboard with PASS/FAIL/SKIP counts;
- saves the complete output log under `test/e2e/.work/`;
- injects a bounded failure summary and recent error output into the Pi session context, then starts an agent turn to diagnose it;
- offers minikube diagnostics and cleanup after a failed or cancelled run.

Press Escape during a run to send SIGTERM to the suite process group. The suite's cleanup trap is allowed to finish before the dashboard closes.

## Tests

```sh
node --test .pi/extensions/e2e-dashboard/e2e-core.test.ts
```
