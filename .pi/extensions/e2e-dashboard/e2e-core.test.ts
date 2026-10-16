import assert from "node:assert/strict";
import test from "node:test";

import { buildFailureContext, discoverFlavours } from "./e2e-core.ts";

test("buildFailureContext returns bounded recent output for the Pi session", () => {
  const log = [
    "early output that should be omitted",
    ...Array.from({ length: 125 }, (_, index) => `line ${index}`),
    "12:00:00 FAIL terraform apply creates the docker cluster",
  ].join("\n");

  const context = buildFailureContext(log, "/tmp/e2e.log", {
    passed: 2,
    failed: 1,
    skipped: 0,
    flavour: "docker",
  });

  assert.match(context, /E2E failed for docker: 2 passed, 1 failed, 0 skipped/);
  assert.match(context, /Full log: \/tmp\/e2e\.log/);
  assert.match(context, /FAIL terraform apply creates the docker cluster/);
  assert.doesNotMatch(context, /early output that should be omitted/);
  assert.match(context, /Diagnose this failure and fix it when safe to do so\./);
});

test("discoverFlavours reports runnable, missing, and unhealthy drivers", async () => {
  const flavours = await discoverFlavours("test/e2e/flavours", {
    hasCommand: async (command) => command === "docker" || command === "podman",
    runHealthcheck: async (command) => command !== "podman info",
  });

  assert.deepEqual(
    flavours.map(({ name, status, reason }) => ({ name, status, reason })),
    [
      { name: "docker", status: "runnable", reason: undefined },
      { name: "podman", status: "unhealthy", reason: '"podman info" failed' },
      {
        name: "qemu",
        status: "missing",
        reason: "qemu-system-x86_64 or qemu-system-aarch64 or qemu-system-riscv64 not installed",
      },
    ],
  );
});
