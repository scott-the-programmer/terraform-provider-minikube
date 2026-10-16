import { spawn, type ChildProcess } from "node:child_process";
import { mkdir, writeFile } from "node:fs/promises";
import { join } from "node:path";

import type { ExtensionAPI, ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { matchesKey, Text, truncateToWidth, wrapTextWithAnsi } from "@earendil-works/pi-tui";

import {
  buildFailureContext,
  CheckRunTracker,
  type CheckRunState,
  planChecks,
  type PlannedCheck,
} from "./check-core.ts";

const STATUS_ID = "project-checks";
const ANSI_PATTERN = /\x1b\[[0-9;]*m/g;

interface CheckResult {
  cancelled: boolean;
  states: CheckRunState[];
  logPath: string;
  elapsedMs: number;
}

async function changedFiles(pi: ExtensionAPI, cwd: string): Promise<string[]> {
  const [tracked, untracked, branch] = await Promise.all([
    pi.exec("git", ["diff", "--name-only", "HEAD", "--"], { cwd, timeout: 10_000 }),
    pi.exec("git", ["ls-files", "--others", "--exclude-standard"], { cwd, timeout: 10_000 }),
    pi.exec("git", ["diff", "--name-only", "origin/main...HEAD", "--"], { cwd, timeout: 10_000 }),
  ]);

  return [...new Set(
    [tracked.stdout, untracked.stdout, branch.code === 0 ? branch.stdout : ""]
      .flatMap((output) => output.split("\n"))
      .map((path) => path.trim())
      .filter(Boolean),
  )].sort();
}

function commandText(check: PlannedCheck): string {
  const quote = (value: string) => /^[A-Za-z0-9_./:=+-]+$/.test(value)
    ? value
    : `'${value.replaceAll("'", `'\\''`)}'`;
  return [check.command, ...check.args].map(quote).join(" ");
}

async function chooseChecks(
  ctx: ExtensionCommandContext,
  checks: PlannedCheck[],
  args: string,
): Promise<PlannedCheck[] | undefined> {
  const requested = args.trim();
  if (requested) {
    if (requested === "all") return checks;
    const ids = requested.split(/[\s,]+/).filter(Boolean);
    const selected = ids.map((id) => checks.find((check) => check.id === id));
    const unknown = ids.filter((_id, index) => !selected[index]);
    if (unknown.length > 0) {
      ctx.ui.notify(`Unknown check(s): ${unknown.join(", ")}`, "error");
      return undefined;
    }
    return selected as PlannedCheck[];
  }

  const recommended = checks.filter((check) => check.recommended);
  const recommendedLabel = `Run recommended (${recommended.map((check) => check.label).join(", ")})`;
  const allLabel = `Run all ${checks.length} checks`;
  const labels = [
    recommendedLabel,
    allLabel,
    ...checks.map((check) => `${check.label} — ${check.description}`),
  ];
  const selected = await ctx.ui.select("Development checks", labels);
  if (!selected) return undefined;
  if (selected === recommendedLabel) return recommended;
  if (selected === allLabel) return checks;
  return [checks[labels.indexOf(selected) - 2]!];
}

function statusSummary(states: CheckRunState[]): string {
  const passed = states.filter((state) => state.status === "passed").length;
  const failed = states.filter((state) => state.status === "failed").length;
  const running = states.find((state) => state.status === "running");
  return running ? `${running.label} · ${passed} passed · ${failed} failed` : `${passed} passed · ${failed} failed`;
}

async function runDashboard(
  ctx: ExtensionCommandContext,
  checks: PlannedCheck[],
): Promise<CheckResult | undefined> {
  if (ctx.mode !== "tui") {
    ctx.ui.notify("/check requires interactive mode", "error");
    return undefined;
  }

  const outputDir = join(ctx.cwd, "test_output");
  await mkdir(outputDir, { recursive: true });
  const stamp = new Date().toISOString().replaceAll(":", "-");
  const logPath = join(outputDir, `pi-check-${stamp}.log`);

  return ctx.ui.custom<CheckResult>((tui, theme, _keybindings, done) => {
    const tracker = new CheckRunTracker(checks);
    const log: string[] = [];
    const recentLines: string[] = [];
    const startedAt = Date.now();
    let child: ChildProcess | undefined;
    let cancelled = false;
    let settled = false;

    const update = () => {
      ctx.ui.setStatus(STATUS_ID, statusSummary(tracker.snapshot()));
      tui.requestRender();
    };

    const consume = (check: PlannedCheck, chunk: Buffer) => {
      const text = chunk.toString("utf8").replace(ANSI_PATTERN, "");
      tracker.append(check.id, text);
      log.push(text);
      for (const line of text.split("\n")) {
        if (!line) continue;
        recentLines.push(line);
        if (recentLines.length > 10) recentLines.shift();
      }
      update();
    };

    const runCheck = (check: PlannedCheck): Promise<number | null> => new Promise((resolve) => {
      const checkStartedAt = Date.now();
      tracker.start(check.id);
      log.push(`\n$ ${commandText(check)}\n`);
      update();

      try {
        child = spawn(check.command, check.args, {
          cwd: ctx.cwd,
          detached: true,
          env: { ...process.env, NO_COLOR: "1" },
          stdio: ["ignore", "pipe", "pipe"],
        });
        child.stdout?.on("data", (chunk: Buffer) => consume(check, chunk));
        child.stderr?.on("data", (chunk: Buffer) => consume(check, chunk));
        child.on("error", (error) => consume(check, Buffer.from(`${error.message}\n`)));
        child.on("close", (code) => {
          tracker.finish(check.id, code, Date.now() - checkStartedAt, cancelled);
          child = undefined;
          update();
          resolve(code);
        });
      } catch (error) {
        consume(check, Buffer.from(`${error instanceof Error ? error.message : String(error)}\n`));
        tracker.finish(check.id, 1, Date.now() - checkStartedAt);
        child = undefined;
        update();
        resolve(1);
      }
    });

    const run = async () => {
      for (const check of checks) {
        if (cancelled) break;
        await runCheck(check);
      }
      if (settled) return;
      settled = true;
      await writeFile(logPath, log.join(""), "utf8");
      ctx.ui.setStatus(STATUS_ID, undefined);
      done({ cancelled, states: tracker.snapshot(), logPath, elapsedMs: Date.now() - startedAt });
    };
    void run();

    return {
      render(width: number): string[] {
        const lines = [theme.fg("accent", "─".repeat(Math.max(1, width)))];
        lines.push(truncateToWidth(` Project checks · ${statusSummary(tracker.snapshot())}`, width));
        lines.push("");
        for (const state of tracker.snapshot()) {
          const marker = state.status === "passed" ? theme.fg("success", "✓")
            : state.status === "failed" ? theme.fg("error", "✗")
            : state.status === "running" ? theme.fg("accent", "●")
            : state.status === "cancelled" ? theme.fg("warning", "↷")
            : theme.fg("dim", "○");
          const elapsed = state.elapsedMs === undefined ? "" : ` ${(state.elapsedMs / 1_000).toFixed(1)}s`;
          lines.push(truncateToWidth(` ${marker} ${state.label}${elapsed}`, width));
        }
        lines.push("");
        for (const line of recentLines) {
          lines.push(...wrapTextWithAnsi(theme.fg("dim", line), Math.max(1, width)));
        }
        lines.push("");
        lines.push(truncateToWidth(theme.fg("muted", " Esc cancels the current check and skips the rest"), width));
        lines.push(theme.fg("accent", "─".repeat(Math.max(1, width))));
        return lines;
      },
      invalidate() {},
      handleInput(data: string) {
        if (!matchesKey(data, "escape") || cancelled || settled) return;
        cancelled = true;
        if (child?.pid) {
          try {
            process.kill(-child.pid, "SIGTERM");
          } catch {
            child.kill("SIGTERM");
          }
        }
        update();
      },
    };
  });
}

export default function checkDashboard(pi: ExtensionAPI) {
  pi.registerEntryRenderer("project-check-run", (entry, _options, theme) => {
    const result = entry.data as CheckResult;
    const failed = result.states.filter((state) => state.status === "failed").length;
    const passed = result.states.filter((state) => state.status === "passed").length;
    const outcome = result.cancelled ? "cancelled" : failed > 0 ? "failed" : "passed";
    const color = result.cancelled ? "warning" : failed > 0 ? "error" : "success";
    return new Text(
      theme.fg(color, `Checks ${outcome}`) +
        theme.fg("muted", ` — ${passed} passed, ${failed} failed in ${Math.round(result.elapsedMs / 1_000)}s\n${result.logPath}`),
      0,
      0,
    );
  });

  pi.registerCommand("check", {
    description: "Run change-aware development checks with an interactive dashboard",
    getArgumentCompletions(prefix) {
      const ids = ["all", "go-format", "go-focused", "go-full", "extension-tests", "terraform-format", "generated"];
      const matches = ids.filter((id) => id.startsWith(prefix)).map((id) => ({ value: id, label: id }));
      return matches.length > 0 ? matches : null;
    },
    handler: async (args, ctx) => {
      const files = await changedFiles(pi, ctx.cwd);
      const checks = planChecks(files);
      const selected = await chooseChecks(ctx, checks, args);
      if (!selected || selected.length === 0) return;

      const result = await runDashboard(ctx, selected);
      if (!result) return;
      pi.appendEntry("project-check-run", result);

      const failures = result.states.filter((state) => state.status === "failed");
      if (result.cancelled) {
        ctx.ui.notify(`Checks cancelled; log: ${result.logPath}`, "warning");
        return;
      }
      if (failures.length === 0) {
        ctx.ui.notify(`All checks passed; log: ${result.logPath}`, "info");
        return;
      }

      ctx.ui.notify(`${failures.length} check(s) failed; log: ${result.logPath}`, "warning");
      pi.sendMessage(
        {
          customType: "project-check-failure",
          content: buildFailureContext(result.states, result.logPath),
          display: true,
          details: result,
        },
        { deliverAs: "followUp", triggerTurn: true },
      );
    },
  });

  pi.on("session_shutdown", (_event, ctx) => {
    ctx.ui.setStatus(STATUS_ID, undefined);
  });
}
