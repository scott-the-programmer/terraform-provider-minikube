import { readFile, readdir } from "node:fs/promises";
import { join } from "node:path";

export interface E2ESelection {
  flavours: string[];
  keep: boolean;
  skipBuild: boolean;
  preloadImage: boolean;
}

export function buildE2EArgs(selection: E2ESelection): string[] {
  const args: string[] = [];
  if (selection.keep) args.push("--keep");
  if (selection.skipBuild) args.push("--skip-build");
  if (selection.preloadImage) args.push("--preload-image");
  return [...args, ...selection.flavours];
}

export type E2EStatus = "PASS" | "FAIL" | "SKIP";

export interface E2EProgress {
  passed: number;
  failed: number;
  skipped: number;
  flavour?: string;
  latest?: { status: E2EStatus; name: string };
}

export class E2EOutputParser {
  private buffered = "";
  private progress: E2EProgress = { passed: 0, failed: 0, skipped: 0 };

  push(chunk: string): void {
    this.buffered += chunk;
    const lines = this.buffered.split("\n");
    this.buffered = lines.pop() ?? "";
    for (const line of lines) this.consume(line);
  }

  finish(): void {
    if (this.buffered) this.consume(this.buffered);
    this.buffered = "";
  }

  snapshot(): E2EProgress {
    return {
      ...this.progress,
      latest: this.progress.latest ? { ...this.progress.latest } : undefined,
    };
  }

  private consume(rawLine: string): void {
    const line = rawLine.replace(/\x1b\[[0-9;]*m/g, "");
    const flavour = line.match(/^FLAVOUR:\s+(\S+)/);
    if (flavour) this.progress.flavour = flavour[1];

    const result = line.match(/^\d{2}:\d{2}:\d{2}\s+(PASS|FAIL|SKIP)\s+(.+)$/);
    if (!result) return;

    const status = result[1] as E2EStatus;
    if (status === "PASS") this.progress.passed++;
    if (status === "FAIL") this.progress.failed++;
    if (status === "SKIP") this.progress.skipped++;
    this.progress.latest = { status, name: result[2] };
  }
}

export function buildFailureContext(log: string, logPath: string, progress: E2EProgress): string {
  const recentLines = log.trimEnd().split("\n").slice(-120);
  let recentOutput = recentLines.join("\n");
  if (recentOutput.length > 16_000) {
    recentOutput = `[Earlier output omitted]\n${recentOutput.slice(-16_000)}`;
  }

  const flavour = progress.flavour ?? "unknown flavour";
  return [
    `E2E failed for ${flavour}: ${progress.passed} passed, ${progress.failed} failed, ${progress.skipped} skipped.`,
    `Full log: ${logPath}`,
    "",
    "Recent E2E output:",
    "```text",
    recentOutput,
    "```",
    "",
    "Diagnose this failure and fix it when safe to do so.",
  ].join("\n");
}

export interface FlavourProbe {
  hasCommand(command: string): Promise<boolean>;
  runHealthcheck(command: string): Promise<boolean>;
}

export interface E2EFlavour {
  name: string;
  description: string;
  status: "runnable" | "missing" | "unhealthy";
  reason?: string;
}

function parseEnv(content: string): Record<string, string> {
  const values: Record<string, string> = {};
  for (const line of content.split("\n")) {
    const match = line.match(/^([A-Z0-9_]+)=(?:"([^"]*)"|'([^']*)'|(.*))$/);
    if (match) values[match[1]] = match[2] ?? match[3] ?? match[4] ?? "";
  }
  return values;
}

export async function discoverFlavours(directory: string, probe: FlavourProbe): Promise<E2EFlavour[]> {
  const names = (await readdir(directory))
    .filter((file) => file.endsWith(".tfvars"))
    .map((file) => file.slice(0, -".tfvars".length))
    .sort();

  return Promise.all(
    names.map(async (name) => {
      const metadata = parseEnv(await readFile(join(directory, `${name}.env`), "utf8"));
      const requirements = (metadata.E2E_REQUIRES ?? "").split(/\s+/).filter(Boolean);

      for (const requirement of requirements) {
        const alternatives = requirement.split("|");
        const available = await Promise.all(alternatives.map((command) => probe.hasCommand(command)));
        if (!available.some(Boolean)) {
          return {
            name,
            description: metadata.E2E_DESCRIPTION ?? name,
            status: "missing" as const,
            reason: `${alternatives.join(" or ")} not installed`,
          };
        }
      }

      const healthcheck = metadata.E2E_HEALTHCHECK;
      if (healthcheck && !(await probe.runHealthcheck(healthcheck))) {
        return {
          name,
          description: metadata.E2E_DESCRIPTION ?? name,
          status: "unhealthy" as const,
          reason: `"${healthcheck}" failed`,
        };
      }

      return {
        name,
        description: metadata.E2E_DESCRIPTION ?? name,
        status: "runnable" as const,
        reason: undefined,
      };
    }),
  );
}
