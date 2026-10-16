import { dirname } from "node:path";

export type CheckStatus = "pending" | "running" | "passed" | "failed" | "cancelled";

export interface CheckRunState {
  id: string;
  label: string;
  status: CheckStatus;
  elapsedMs?: number;
  output: string;
}

export class CheckRunTracker {
  private readonly states: CheckRunState[];

  constructor(checks: Array<{ id: string; label: string }>) {
    this.states = checks.map(({ id, label }) => ({ id, label, status: "pending", output: "" }));
  }

  start(id: string): void {
    this.find(id).status = "running";
  }

  append(id: string, output: string): void {
    const state = this.find(id);
    state.output = `${state.output}${output}`.slice(-16_000);
  }

  finish(id: string, code: number | null, elapsedMs: number, cancelled = false): void {
    const state = this.find(id);
    state.status = cancelled ? "cancelled" : code === 0 ? "passed" : "failed";
    state.elapsedMs = elapsedMs;
  }

  snapshot(): CheckRunState[] {
    return this.states.map((state) => ({ ...state }));
  }

  private find(id: string): CheckRunState {
    const state = this.states.find((candidate) => candidate.id === id);
    if (!state) throw new Error(`Unknown check: ${id}`);
    return state;
  }
}

export function buildFailureContext(states: CheckRunState[], logPath: string): string {
  const failed = states.filter((state) => state.status === "failed");
  const sections = failed.map((state) => {
    const output = state.output.length > 14_000
      ? `[Earlier output omitted]\n${state.output.slice(-14_000)}`
      : state.output;
    return [`## ${state.label}`, "```text", output.trimEnd(), "```"].join("\n");
  });

  return [
    `Development checks failed: ${failed.map((state) => state.label).join(", ")}.`,
    `Full log: ${logPath}`,
    "",
    ...sections,
    "",
    "Diagnose the failed checks and fix them when safe to do so.",
  ].join("\n");
}

export interface PlannedCheck {
  id: string;
  label: string;
  description: string;
  command: string;
  args: string[];
  recommended: boolean;
}

export function planChecks(changedFiles: string[]): PlannedCheck[] {
  const checks: PlannedCheck[] = [];
  const goFiles = changedFiles.filter((path) => path.endsWith(".go"));

  if (goFiles.length > 0) {
    const packages = [...new Set(goFiles.map((path) => {
      const directory = dirname(path);
      return directory === "." ? "." : `./${directory}`;
    }))].sort();

    checks.push(
      {
        id: "go-format",
        label: "Go formatting",
        description: "Verify changed Go files are formatted",
        command: "sh",
        args: [
          "-c",
          "out=$(gofmt -l \"$@\"); test -z \"$out\" || { printf '%s\\n' \"$out\"; exit 1; }",
          "check",
          ...goFiles,
        ],
        recommended: true,
      },
      {
        id: "go-focused",
        label: "Focused Go tests",
        description: `Test ${packages.join(", ")}`,
        command: "go",
        args: ["test", ...packages],
        recommended: true,
      },
    );
  }

  if (changedFiles.some((path) => path.startsWith(".pi/extensions/") && path.endsWith(".ts"))) {
    checks.push({
      id: "extension-tests",
      label: "Pi extension tests",
      description: "Run every project-local Pi extension test",
      command: "node",
      args: ["--test", ".pi/extensions/**/*.test.ts"],
      recommended: true,
    });
  }

  if (changedFiles.some((path) => path.endsWith(".tf"))) {
    checks.push({
      id: "terraform-format",
      label: "Terraform formatting",
      description: "Verify Terraform files are formatted",
      command: "terraform",
      args: ["fmt", "-check", "-recursive"],
      recommended: true,
    });
  }

  const generatorInputs = changedFiles.some((path) =>
    path === "main.go" ||
    path === "go.mod" ||
    path === "go.sum" ||
    path.startsWith("generate/") ||
    path.startsWith("templates/") ||
    path.includes("/generator/") ||
    /minikube\/lib\/minikube_(client|cluster|downloader)\.go$/.test(path),
  );
  if (generatorInputs) {
    checks.push({
      id: "generated",
      label: "Generated artifacts",
      description: "Regenerate schema, mocks, and docs and verify they are committed",
      command: "sh",
      args: [
        "-c",
        "make configure && git diff --exit-code -- docs minikube/schema_cluster.go ':(glob)minikube/**/mock_*.go'",
      ],
      recommended: true,
    });
  }

  checks.push({
    id: "go-full",
    label: "Full Go test suite",
    description: "Run make test across every Go package",
    command: "make",
    args: ["test"],
    recommended: goFiles.length === 0,
  });

  return checks;
}
