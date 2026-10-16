import { mkdir, readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";

import type { ExtensionAPI, ExtensionCommandContext, Theme } from "@earendil-works/pi-coding-agent";
import { matchesKey, Text, truncateToWidth, wrapTextWithAnsi } from "@earendil-works/pi-tui";

import {
  buildFailureContext,
  discoverFlavours,
  type E2EFlavour,
  type E2EProgress,
  type E2ESelection,
} from "./e2e-core.ts";
import {
  startE2ERun,
  type E2EMode,
  type E2EProcessResult,
  type RunningE2E,
} from "./e2e-runner.ts";

const STATUS_ID = "minikube-e2e";
const STATE_FILE = join("test", "e2e", ".work", "pi-dashboard-state.json");
const ANSI_PATTERN = /\x1b\[[0-9;]*m/g;

interface SavedState extends E2ESelection {}

interface RunResult {
  mode: E2EMode;
  code: number | null;
  cancelled: boolean;
  progress: E2EProgress;
  logPath: string;
  elapsedMs: number;
}

function shellQuote(value: string): string {
  return `'${value.replaceAll("'", `'\\''`)}'`;
}

async function loadState(cwd: string): Promise<SavedState | undefined> {
  try {
    return JSON.parse(await readFile(join(cwd, STATE_FILE), "utf8")) as SavedState;
  } catch {
    return undefined;
  }
}

async function saveState(cwd: string, state: SavedState): Promise<void> {
  const path = join(cwd, STATE_FILE);
  await mkdir(join(cwd, "test", "e2e", ".work"), { recursive: true });
  await writeFile(path, `${JSON.stringify(state, null, 2)}\n`, "utf8");
}

async function findFlavours(pi: ExtensionAPI, cwd: string): Promise<E2EFlavour[]> {
  return discoverFlavours(join(cwd, "test", "e2e", "flavours"), {
    async hasCommand(command) {
      const result = await pi.exec("sh", ["-lc", `command -v -- ${shellQuote(command)}`], { timeout: 5_000 });
      return result.code === 0;
    },
    async runHealthcheck(command) {
      const result = await pi.exec("sh", ["-lc", command], { timeout: 15_000 });
      return result.code === 0;
    },
  });
}

function optionSummary(selection: E2ESelection): string {
  return [
    `keep=${selection.keep ? "on" : "off"}`,
    `skip-build=${selection.skipBuild ? "on" : "off"}`,
    `preload=${selection.preloadImage ? "on" : "off"}`,
  ].join(", ");
}

async function chooseBoolean(
  ctx: ExtensionCommandContext,
  title: string,
  current: boolean,
): Promise<boolean | undefined> {
  const currentLabel = current ? "On (current)" : "Off (current)";
  const otherLabel = current ? "Off" : "On";
  const selected = await ctx.ui.select(title, [currentLabel, otherLabel]);
  if (!selected) return undefined;
  return selected.startsWith("On");
}

async function chooseSelection(
  pi: ExtensionAPI,
  ctx: ExtensionCommandContext,
  args: string,
): Promise<E2ESelection | undefined> {
  const flavours = await findFlavours(pi, ctx.cwd);
  const previous = await loadState(ctx.cwd);
  const byName = new Map(flavours.map((flavour) => [flavour.name, flavour]));
  const runnable = flavours.filter((flavour) => flavour.status === "runnable");
  if (runnable.length === 0) {
    ctx.ui.notify("No runnable E2E drivers found", "error");
    return undefined;
  }

  let selectedNames: string[];
  const requested = args.trim().split(/[\s,]+/).filter(Boolean);
  if (requested.length > 0) {
    const unknown = requested.filter((name) => !byName.has(name));
    if (unknown.length > 0) {
      ctx.ui.notify(`Unknown E2E flavour(s): ${unknown.join(", ")}`, "error");
      return undefined;
    }
    const unavailable = requested.map((name) => byName.get(name)!).filter((flavour) => flavour.status !== "runnable");
    if (unavailable.length > 0) {
      ctx.ui.notify(
        unavailable.map((flavour) => `${flavour.name}: ${flavour.reason}`).join("; "),
        "error",
      );
      return undefined;
    }
    selectedNames = requested;
  } else {
    const reusable = previous?.flavours.filter((name) => byName.get(name)?.status === "runnable") ?? [];
    const lastLabel = reusable.length > 0 ? `Last used (${reusable.join(", ")})` : undefined;
    const allLabel = `All runnable (${runnable.map((flavour) => flavour.name).join(", ")})`;
    const labels = [
      ...(lastLabel ? [lastLabel] : []),
      allLabel,
      ...flavours.map((flavour) => {
        const marker = flavour.status === "runnable" ? "ready" : flavour.status;
        const reason = flavour.reason ? ` — ${flavour.reason}` : "";
        return `${flavour.name} [${marker}] — ${flavour.description}${reason}`;
      }),
    ];
    const selected = await ctx.ui.select("E2E driver flavour", labels);
    if (!selected) return undefined;
    if (selected === lastLabel) {
      selectedNames = reusable;
    } else if (selected === allLabel) {
      selectedNames = runnable.map((flavour) => flavour.name);
    } else {
      const name = selected.split(" ", 1)[0];
      const flavour = byName.get(name);
      if (!flavour || flavour.status !== "runnable") {
        ctx.ui.notify(flavour?.reason ?? "Driver is unavailable", "warning");
        return undefined;
      }
      selectedNames = [name];
    }
  }

  let selection: E2ESelection = {
    flavours: selectedNames,
    keep: previous?.keep ?? false,
    skipBuild: previous?.skipBuild ?? false,
    preloadImage: previous?.preloadImage ?? false,
  };

  const optionsChoice = await ctx.ui.select("E2E options", [
    `Run with ${optionSummary(selection)}`,
    "Configure options",
  ]);
  if (!optionsChoice) return undefined;

  if (optionsChoice === "Configure options") {
    const keep = await chooseBoolean(ctx, "Keep clusters and Terraform state after the run?", selection.keep);
    if (keep === undefined) return undefined;
    const skipBuild = await chooseBoolean(ctx, "Reuse the currently installed local provider?", selection.skipBuild);
    if (skipBuild === undefined) return undefined;
    const preloadImage = await chooseBoolean(ctx, "Preload the nginx image into each cluster?", selection.preloadImage);
    if (preloadImage === undefined) return undefined;
    selection = { ...selection, keep, skipBuild, preloadImage };
  }

  await saveState(ctx.cwd, selection);
  return selection;
}

function elapsed(startedAt: number): string {
  const seconds = Math.floor((Date.now() - startedAt) / 1_000);
  return `${Math.floor(seconds / 60)}m ${String(seconds % 60).padStart(2, "0")}s`;
}

function statusText(theme: Theme, progress: E2EProgress, startedAt: number): string {
  const counts = [
    theme.fg("success", `✓ ${progress.passed}`),
    theme.fg("error", `✗ ${progress.failed}`),
    theme.fg("warning", `↷ ${progress.skipped}`),
  ].join("  ");
  return `${theme.fg("accent", progress.flavour ?? "starting")}  ${counts}  ${theme.fg("dim", elapsed(startedAt))}`;
}

async function runDashboard(
  ctx: ExtensionCommandContext,
  selection: E2ESelection,
  mode: E2EMode,
): Promise<RunResult | undefined> {
  if (ctx.mode !== "tui") {
    ctx.ui.notify(`/${mode} requires interactive mode`, "error");
    return undefined;
  }

  const workDir = join(ctx.cwd, "test", "e2e", ".work");
  await mkdir(workDir, { recursive: true });
  const stamp = new Date().toISOString().replaceAll(":", "-");
  const logPath = join(workDir, `pi-${mode}-${stamp}.log`);
  const title = mode === "upgrade-tests" ? "Provider upgrade tests" : "Minikube E2E";

  return ctx.ui.custom<RunResult>((tui, theme, _keybindings, done) => {
    const recentLines: string[] = [];
    const allLines: string[] = [];
    const startedAt = Date.now();
    let run: RunningE2E | undefined;
    let cancelled = false;
    let settled = false;

    const currentProgress = (): E2EProgress =>
      run?.progress() ?? { passed: 0, failed: 0, skipped: 0 };
    const updateStatus = () => {
      ctx.ui.setStatus(STATUS_ID, statusText(theme, currentProgress(), startedAt));
      tui.requestRender();
    };
    const ticker = setInterval(updateStatus, 1_000);
    updateStatus();

    const consumeLine = (line: string) => {
      const clean = line.replace(ANSI_PATTERN, "");
      allLines.push(clean);
      recentLines.push(clean);
      if (recentLines.length > 12) recentLines.shift();
      updateStatus();
    };

    const finish = async ({ code, progress }: E2EProcessResult) => {
      if (settled) return;
      settled = true;
      clearInterval(ticker);
      await writeFile(logPath, `${allLines.join("\n")}\n`, "utf8");
      ctx.ui.setStatus(STATUS_ID, undefined);
      done({ mode, code, cancelled, progress, logPath, elapsedMs: Date.now() - startedAt });
    };

    run = startE2ERun({ cwd: ctx.cwd, mode, selection }, consumeLine);
    void run.completed.then(finish);

    return {
      render(width: number): string[] {
        const progress = currentProgress();
        const lines = [
          theme.fg("accent", "─".repeat(Math.max(1, width))),
          truncateToWidth(` ${title}  ${statusText(theme, progress, startedAt)}`, width),
          truncateToWidth(` ${run?.displayCommand ?? "Starting E2E runner..."}`, width),
          ...(progress.latest
            ? [truncateToWidth(` ${progress.latest.status} ${progress.latest.name}`, width)]
            : []),
          "",
        ];
        for (const line of recentLines) {
          lines.push(...wrapTextWithAnsi(theme.fg("dim", line), Math.max(1, width)));
        }
        lines.push("");
        lines.push(truncateToWidth(theme.fg("muted", " Esc cancels safely and triggers the suite cleanup trap"), width));
        lines.push(theme.fg("accent", "─".repeat(Math.max(1, width))));
        return lines;
      },
      invalidate() {},
      handleInput(data: string) {
        if (!matchesKey(data, "escape") || !run || settled) return;
        cancelled = true;
        consumeLine("Cancellation requested; waiting for E2E cleanup...");
        run.cancel();
      },
    };
  });
}

async function collectDiagnostics(pi: ExtensionAPI, ctx: ExtensionCommandContext, selection: E2ESelection): Promise<string> {
  const parts: string[] = [];
  const profiles = await pi.exec("minikube", ["profile", "list"], { timeout: 30_000 });
  parts.push("$ minikube profile list", profiles.stdout, profiles.stderr);

  for (const flavour of selection.flavours) {
    const profile = `tf-minikube-e2e-${flavour}`;
    const logs = await pi.exec("minikube", ["logs", "-p", profile, "--length", "200"], { timeout: 60_000 });
    parts.push(`\n$ minikube logs -p ${profile} --length 200`, logs.stdout, logs.stderr);
  }

  const path = join(ctx.cwd, "test", "e2e", ".work", `pi-diagnostics-${Date.now()}.log`);
  await writeFile(path, parts.filter(Boolean).join("\n"), "utf8");
  return path;
}

async function cleanKeptClusters(pi: ExtensionAPI, ctx: ExtensionCommandContext, selection: E2ESelection): Promise<void> {
  const confirmed = await ctx.ui.confirm(
    "Destroy kept E2E clusters?",
    `Terraform destroy will run for: ${selection.flavours.join(", ")}`,
  );
  if (!confirmed) return;

  for (const flavour of selection.flavours) {
    const directory = join(ctx.cwd, "test", "e2e", ".work", flavour);
    const result = await pi.exec("terraform", [`-chdir=${directory}`, "destroy", "-auto-approve"], { timeout: 900_000 });
    if (result.code !== 0) {
      ctx.ui.notify(`Cleanup failed for ${flavour}: ${result.stderr.trim()}`, "error");
      return;
    }
  }
  ctx.ui.notify("Kept E2E clusters destroyed", "info");
}

export default function e2eDashboard(pi: ExtensionAPI) {
  pi.registerEntryRenderer("minikube-e2e-run", (entry, _options, theme) => {
    const result = entry.data as RunResult;
    const outcome = result.cancelled ? "cancelled" : result.code === 0 ? "passed" : "failed";
    const color = result.code === 0 ? "success" : result.cancelled ? "warning" : "error";
    const seconds = Math.round(result.elapsedMs / 1_000);
    const label = result.mode === "upgrade-tests" ? "Upgrade tests" : "E2E";
    return new Text(
      theme.fg(color, `${label} ${outcome}`) +
        theme.fg(
          "muted",
          ` — ${result.progress.passed} passed, ${result.progress.failed} failed, ${result.progress.skipped} skipped in ${seconds}s\n${result.logPath}`,
        ),
      0,
      0,
    );
  });

  const flavourCompletions = (prefix: string) => {
    const matches = ["docker", "podman", "qemu"]
      .filter((name) => name.startsWith(prefix))
      .map((name) => ({ value: name, label: name }));
    return matches.length > 0 ? matches : null;
  };

  const runCommand = async (args: string, ctx: ExtensionCommandContext, mode: E2EMode) => {
    const selection = await chooseSelection(pi, ctx, args);
    if (!selection) return;

    const result = await runDashboard(ctx, selection, mode);
    if (!result) return;
    pi.appendEntry("minikube-e2e-run", result);

    const outcome = result.cancelled ? "cancelled" : result.code === 0 ? "passed" : "failed";
    const label = mode === "upgrade-tests" ? "Upgrade tests" : "E2E";
    ctx.ui.notify(`${label} ${outcome}; log: ${result.logPath}`, result.code === 0 ? "info" : "warning");

    let failureContext: string | undefined;
    if (result.code !== 0 && !result.cancelled) {
      const log = await readFile(result.logPath, "utf8");
      failureContext = buildFailureContext(log, result.logPath, result.progress);
    }

    if (result.code === 0 && !result.cancelled) return;
    const actions = ["Collect minikube diagnostics"];
    if (selection.keep) actions.push("Destroy kept clusters");
    actions.push("Done");
    const action = await ctx.ui.select(`${label} follow-up`, actions);
    if (action === "Collect minikube diagnostics") {
      const path = await collectDiagnostics(pi, ctx, selection);
      ctx.ui.notify(`Diagnostics saved to ${path}`, "info");
    } else if (action === "Destroy kept clusters") {
      await cleanKeptClusters(pi, ctx, selection);
    }

    if (failureContext) {
      pi.sendMessage(
        {
          customType: "minikube-e2e-failure",
          content: failureContext,
          display: true,
          details: { selection, result },
        },
        { deliverAs: "followUp", triggerTurn: true },
      );
    }
  };

  pi.registerCommand("e2e", {
    description: "Run minikube end-to-end tests with an interactive dashboard",
    getArgumentCompletions: flavourCompletions,
    handler: (args, ctx) => runCommand(args, ctx, "e2e"),
  });

  pi.registerCommand("upgrade-tests", {
    description: "Verify upgrading the latest release to the local provider produces a no-op plan",
    getArgumentCompletions: flavourCompletions,
    handler: (args, ctx) => runCommand(args, ctx, "upgrade-tests"),
  });

  pi.on("session_shutdown", (_event, ctx) => {
    ctx.ui.setStatus(STATUS_ID, undefined);
  });
}
