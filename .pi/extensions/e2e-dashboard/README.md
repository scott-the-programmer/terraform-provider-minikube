# Minikube E2E dashboard

Project-local Pi frontend for the repository's E2E test harness.

## Usage

Reload Pi after checkout, then run:

```text
/e2e
/e2e docker
/e2e docker podman
/upgrade-tests docker
/upgrade-tests docker podman
```

`/e2e` runs the complete black-box suite. `/upgrade-tests` creates the same
configuration with the latest released provider, upgrades it to the local
`99.99.99` build, and requires the resulting Terraform plan to be a no-op.

Both commands:

- discover driver flavours and check their executable/health requirements;
- remember the last flavour and options under the ignored `test/e2e/.work/` directory;
- configure `--keep`, `--skip-build`, and `--preload-image`;
- stream a cancellable dashboard with PASS/FAIL/SKIP counts;
- save the complete output log under `test/e2e/.work/`;
- inject a bounded failure summary and recent error output into the Pi session context, then start an agent turn to diagnose it;
- offer minikube diagnostics and cleanup after a failed or cancelled run.

Press Escape during a run to send SIGTERM to the suite process group. The suite's cleanup trap is allowed to finish before the dashboard closes.

## Architecture

The Pi command runs a native TypeScript harness in `e2e-runner.ts`; it does not
invoke `test/e2e/run.sh`. The runner owns provider builds, Terraform lifecycle,
kubeconfig generation, Kubernetes assertions, image preloading, upgrade tests,
cancellation, and cleanup. `index.ts` only owns selection and presentation.

The shell harness remains available independently for command-line and CI use.

## Tests

```sh
node --test .pi/extensions/e2e-dashboard/*.test.ts
```
