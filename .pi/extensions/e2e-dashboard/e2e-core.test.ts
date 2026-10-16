import assert from "node:assert/strict";
import test from "node:test";

import { buildE2EArgs, buildFailureContext, discoverFlavours, E2EOutputParser } from "./e2e-core.ts";

test("buildE2EArgs maps the selected flavours and options to run.sh arguments", () => {
  assert.deepEqual(
    buildE2EArgs({
      flavours: ["docker", "qemu"],
      keep: true,
      skipBuild: true,
      preloadImage: true,
    }),
    ["--keep", "--skip-build", "--preload-image", "docker", "qemu"],
  );
});

test("E2EOutputParser tracks results across streamed chunks", () => {
  const parser = new E2EOutputParser();

  parser.push("12:00:00 [32mPASS[0m terraform init\nFLAVOUR: dock");
  parser.push("er - docker driver\n12:00:01 FAIL deployment rolls out\n12:00:02 SKIP node port - not routable\n");

  assert.deepEqual(parser.snapshot(), {
    passed: 1,
    failed: 1,
    skipped: 1,
    flavour: "docker",
    latest: { status: "SKIP", name: "node port - not routable" },
  });
});

test("E2EOutputParser finishes a final line without a newline", () => {
  const parser = new E2EOutputParser();
  parser.push("12:00:00 PASS final check");
  parser.finish();

  assert.equal(parser.snapshot().passed, 1);
});

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
