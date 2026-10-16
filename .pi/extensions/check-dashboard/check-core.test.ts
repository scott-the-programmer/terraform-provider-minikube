import assert from "node:assert/strict";
import test from "node:test";

import { buildFailureContext, CheckRunTracker, planChecks } from "./check-core.ts";

test("planChecks recommends focused checks for changed Go packages", () => {
  const checks = planChecks([
    "minikube/lib/memory.go",
    "minikube/state_utils/cpu_test.go",
  ]);

  assert.deepEqual(
    checks.filter((check) => check.recommended).map((check) => check.id),
    ["go-format", "go-focused"],
  );
  assert.deepEqual(
    checks.find((check) => check.id === "go-focused")?.args,
    ["test", "./minikube/lib", "./minikube/state_utils"],
  );
});

test("planChecks maps project files to specialized checks while keeping the full suite available", () => {
  const checks = planChecks([
    ".pi/extensions/e2e-dashboard/index.ts",
    "test/e2e/stack/main.tf",
    "minikube/generator/schema_builder.go",
  ]);

  assert.deepEqual(
    checks.map(({ id, recommended }) => ({ id, recommended })),
    [
      { id: "go-format", recommended: true },
      { id: "go-focused", recommended: true },
      { id: "extension-tests", recommended: true },
      { id: "terraform-format", recommended: true },
      { id: "generated", recommended: true },
      { id: "go-full", recommended: false },
    ],
  );
});

test("CheckRunTracker reports sequential check progress", () => {
  const tracker = new CheckRunTracker([
    { id: "fmt", label: "Formatting" },
    { id: "test", label: "Tests" },
  ]);

  tracker.start("fmt");
  tracker.append("fmt", "checking files\n");
  tracker.finish("fmt", 0, 125);
  tracker.start("test");
  tracker.finish("test", 1, 800);

  assert.deepEqual(tracker.snapshot(), [
    { id: "fmt", label: "Formatting", status: "passed", elapsedMs: 125, output: "checking files\n" },
    { id: "test", label: "Tests", status: "failed", elapsedMs: 800, output: "" },
  ]);
});

test("buildFailureContext includes failed checks and bounded recent output", () => {
  const context = buildFailureContext(
    [
      { id: "fmt", label: "Formatting", status: "passed", elapsedMs: 10, output: "" },
      { id: "test", label: "Tests", status: "failed", elapsedMs: 20, output: `very early output${"x".repeat(20_000)}useful failure` },
    ],
    "/tmp/check.log",
  );

  assert.match(context, /Development checks failed: Tests/);
  assert.match(context, /Full log: \/tmp\/check\.log/);
  assert.match(context, /useful failure/);
  assert.doesNotMatch(context, /very early output/);
  assert.ok(context.length < 18_000);
  assert.match(context, /Diagnose the failed checks and fix them when safe to do so\./);
});
