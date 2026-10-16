import { spawn, type ChildProcess } from "node:child_process";
import { access, cp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { join } from "node:path";

import type { E2EProgress, E2ESelection } from "./e2e-core.ts";

export type E2EMode = "e2e" | "upgrade-tests";

export interface CommandResult {
  code: number;
  stdout: string;
  stderr: string;
}

interface CommandOptions {
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  timeoutMs?: number;
  signal?: AbortSignal;
  onOutput?: (text: string) => void;
}

export interface E2ECommandExecutor {
  run(command: string, args: string[], options?: CommandOptions): Promise<CommandResult>;
  start?(command: string, args: string[], options?: CommandOptions): RunningCommand;
}

interface RunningCommand {
  completed: Promise<CommandResult>;
  cancel(): void;
}

export interface E2ERunRequest {
  cwd: string;
  mode: E2EMode;
  selection: E2ESelection;
  executor?: E2ECommandExecutor;
}

export interface E2EProcessResult {
  code: number | null;
  progress: E2EProgress;
}

export interface RunningE2E {
  displayCommand: string;
  completed: Promise<E2EProcessResult>;
  progress(): E2EProgress;
  cancel(): void;
}

function localExecutor(): E2ECommandExecutor {
  const start = (command: string, args: string[], options: CommandOptions = {}): RunningCommand => {
    let child: ChildProcess | undefined;
    let timer: NodeJS.Timeout | undefined;
    let cancelled = false;
    const completed = new Promise<CommandResult>((resolve) => {
      let stdout = "";
      let stderr = "";
      let settled = false;
      const finish = (code: number) => {
        if (settled) return;
        settled = true;
        if (timer) clearTimeout(timer);
        options.signal?.removeEventListener("abort", cancel);
        resolve({ code, stdout, stderr });
      };
      const cancel = () => {
        cancelled = true;
        if (!child) return;
        if (child.pid) {
          try { process.kill(-child.pid, "SIGTERM"); return; } catch { /* fall through */ }
        }
        child.kill("SIGTERM");
      };
      try {
        child = spawn(command, args, {
          cwd: options.cwd,
          env: options.env ?? process.env,
          detached: true,
          stdio: ["ignore", "pipe", "pipe"],
        });
        child.stdout?.on("data", (chunk: Buffer) => {
          const text = chunk.toString("utf8");
          stdout += text;
          options.onOutput?.(text);
        });
        child.stderr?.on("data", (chunk: Buffer) => {
          const text = chunk.toString("utf8");
          stderr += text;
          options.onOutput?.(text);
        });
        child.on("error", (error) => {
          stderr += error.message;
          finish(1);
        });
        child.on("close", (code) => finish(cancelled ? 130 : (code ?? 1)));
        options.signal?.addEventListener("abort", cancel, { once: true });
        if (options.timeoutMs) {
          timer = setTimeout(() => { cancel(); finish(124); }, options.timeoutMs);
        }
      } catch (error) {
        stderr += error instanceof Error ? error.message : String(error);
        finish(1);
      }
    });
    return {
      completed,
      cancel() {
        cancelled = true;
        if (child?.pid) {
          try { process.kill(-child.pid, "SIGTERM"); return; } catch { /* fall through */ }
        }
        child?.kill("SIGTERM");
      },
    };
  };
  return { start, run: (command, args, options) => start(command, args, options).completed };
}

const clean = (value: string) => value.trim();
const sleep = (ms: number, signal: AbortSignal) => new Promise<void>((resolve, reject) => {
  const timer = setTimeout(resolve, ms);
  signal.addEventListener("abort", () => { clearTimeout(timer); reject(new Error("cancelled")); }, { once: true });
});

async function parseEnv(path: string): Promise<Record<string, string>> {
  const values: Record<string, string> = {};
  for (const line of (await readFile(path, "utf8")).split("\n")) {
    const match = line.match(/^([A-Z0-9_]+)=(?:"([^"]*)"|'([^']*)'|(.*))$/);
    if (match) values[match[1]] = match[2] ?? match[3] ?? match[4] ?? "";
  }
  return values;
}

class NativeE2ESuite {
  private readonly executor: E2ECommandExecutor;
  private readonly controller = new AbortController();
  private readonly progressState: E2EProgress = { passed: 0, failed: 0, skipped: 0 };
  private readonly root: string;
  private readonly e2eDir: string;
  private readonly workDir: string;
  private readonly request: E2ERunRequest;
  private readonly onLine: (line: string) => void;
  private env: NodeJS.ProcessEnv = { ...process.env, NO_COLOR: "1", TF_IN_AUTOMATION: "1", TF_INPUT: "0" };
  private active?: RunningCommand;
  private readonly cleanupDirs = new Set<string>();

  constructor(request: E2ERunRequest, onLine: (line: string) => void) {
    this.request = request;
    this.onLine = onLine;
    this.executor = request.executor ?? localExecutor();
    this.root = request.cwd;
    this.e2eDir = join(this.root, "test", "e2e");
    this.workDir = join(this.e2eDir, ".work");
  }

  progress(): E2EProgress {
    return { ...this.progressState, latest: this.progressState.latest ? { ...this.progressState.latest } : undefined };
  }

  cancel(): void {
    this.controller.abort();
    this.active?.cancel();
  }

  private output(text: string): void {
    for (const line of text.replace(/\n$/, "").split("\n")) if (line) this.onLine(line);
  }

  private async command(command: string, args: string[], options: CommandOptions = {}): Promise<CommandResult> {
    if (this.controller.signal.aborted) return { code: 130, stdout: "", stderr: "cancelled" };
    return this.executor.run(command, args, {
      cwd: options.cwd ?? this.root,
      env: options.env ?? this.env,
      timeoutMs: options.timeoutMs,
      signal: this.controller.signal,
      onOutput: options.onOutput,
    });
  }

  private record(status: "PASS" | "FAIL" | "SKIP", name: string): void {
    if (status === "PASS") this.progressState.passed++;
    if (status === "FAIL") this.progressState.failed++;
    if (status === "SKIP") this.progressState.skipped++;
    this.progressState.latest = { status, name };
    this.onLine(`${new Date().toISOString().slice(11, 19)} ${status} ${name}`);
  }

  private async check(name: string, action: () => Promise<boolean>): Promise<boolean> {
    try {
      const passed = await action();
      this.record(passed ? "PASS" : "FAIL", name);
      return passed;
    } catch (error) {
      this.output(error instanceof Error ? error.message : String(error));
      this.record("FAIL", name);
      return false;
    }
  }

  private async runOk(command: string, args: string[], options: CommandOptions = {}): Promise<boolean> {
    const result = await this.command(command, args, options);
    if (result.code !== 0) this.output(result.stderr || result.stdout);
    return result.code === 0;
  }

  private tfArgs(dir: string, subcommand: string, ...args: string[]): string[] {
    return [`-chdir=${dir}`, subcommand, "-no-color", ...args];
  }
  private kubeArgs(kubeconfig: string, ...args: string[]): string[] {
    return ["--kubeconfig", kubeconfig, "--request-timeout=30s", ...args];
  }

  private async retry(action: () => Promise<boolean>, timeoutMs = 120_000, intervalMs = 3_000): Promise<boolean> {
    const deadline = Date.now() + timeoutMs;
    do {
      if (await action()) return true;
      if (Date.now() >= deadline || this.controller.signal.aborted) return false;
      await sleep(intervalMs, this.controller.signal);
    } while (true);
  }

  private async prepare(flavour: string, version: string): Promise<string> {
    const dir = join(this.workDir, flavour);
    await rm(dir, { recursive: true, force: true });
    await mkdir(dir, { recursive: true });
    await cp(join(this.e2eDir, "stack"), dir, { recursive: true });
    await cp(join(this.e2eDir, "flavours", `${flavour}.tfvars`), join(dir, "terraform.tfvars"));
    if (version !== "99.99.99") {
      const versions = join(dir, "versions.tf");
      await writeFile(versions, (await readFile(versions, "utf8")).replace('version = "99.99.99"', `version = "${version}"`));
    }
    return dir;
  }

  private async terraformOutput(dir: string, name: string): Promise<string> {
    const result = await this.command("terraform", this.tfArgs(dir, "output", "-raw", name));
    if (result.code !== 0) throw new Error(result.stderr);
    return clean(result.stdout);
  }

  private async writeKubeconfig(dir: string, path: string, name: string): Promise<void> {
    const host = await this.terraformOutput(dir, "host");
    const ca = Buffer.from(await this.terraformOutput(dir, "cluster_ca_certificate")).toString("base64");
    const cert = Buffer.from(await this.terraformOutput(dir, "client_certificate")).toString("base64");
    const key = Buffer.from(await this.terraformOutput(dir, "client_key")).toString("base64");
    await writeFile(path, `apiVersion: v1\nkind: Config\nclusters:\n- name: ${name}\n  cluster:\n    server: ${host}\n    certificate-authority-data: ${ca}\nusers:\n- name: ${name}\n  user:\n    client-certificate-data: ${cert}\n    client-key-data: ${key}\ncontexts:\n- name: ${name}\n  context:\n    cluster: ${name}\n    user: ${name}\ncurrent-context: ${name}\n`, { mode: 0o600 });
  }

  private async cleanPlan(dir: string): Promise<boolean> {
    const result = await this.command("terraform", this.tfArgs(dir, "plan", "-detailed-exitcode"));
    if (result.code !== 0) this.output(result.stderr || result.stdout);
    return result.code === 0;
  }

  private async runFlavour(flavour: string, upgradeFrom?: string): Promise<void> {
    const metadata = await parseEnv(join(this.e2eDir, "flavours", `${flavour}.env`));
    const tfvars = await readFile(join(this.e2eDir, "flavours", `${flavour}.tfvars`), "utf8");
    const clusterName = tfvars.match(/cluster_name\s*=\s*"([^"]+)"/)?.[1] ?? `tf-minikube-e2e-${flavour}`;
    const expectedNodes = Number(tfvars.match(/nodes\s*=\s*(\d+)/)?.[1] ?? 1);
    this.progressState.flavour = flavour;
    this.onLine(`FLAVOUR: ${flavour} - ${metadata.E2E_DESCRIPTION ?? flavour}`);

    const dir = await this.prepare(flavour, upgradeFrom ?? "99.99.99");
    this.cleanupDirs.add(dir);
    const kubeconfig = join(dir, "kubeconfig");
    const releasedEnv = upgradeFrom ? { ...this.env, TF_CLI_CONFIG_FILE: undefined } : this.env;
    if (!await this.check(upgradeFrom ? `terraform init installs released provider ${upgradeFrom}` : "terraform init", () =>
      this.runOk("terraform", this.tfArgs(dir, "init", "-upgrade"), { env: releasedEnv }))) return;
    if (!await this.check(`terraform apply creates the ${flavour} cluster`, () =>
      this.runOk("terraform", this.tfArgs(dir, "apply", "-auto-approve"), { timeoutMs: Number(metadata.E2E_APPLY_TIMEOUT ?? 1200) * 1000, onOutput: (text) => this.output(text) }))) {
      await this.destroy(dir); return;
    }

    await this.check("provider outputs expose a usable apiserver endpoint and credentials", async () => {
      const host = await this.terraformOutput(dir, "host");
      if (!host.startsWith("https://")) return false;
      for (const field of ["cluster_ca_certificate", "client_certificate", "client_key"])
        if (!(await this.terraformOutput(dir, field)).includes("-----BEGIN")) return false;
      return true;
    });

    if (upgradeFrom) {
      await this.check(`released provider ${upgradeFrom} has a clean baseline plan`, () => this.cleanPlan(dir));
      const versions = join(dir, "versions.tf");
      await writeFile(
        versions,
        (await readFile(versions, "utf8")).replace(
          /^(\s*)version = "[^"]*"/m,
          '$1version = "99.99.99"',
        ),
      );
      if (!await this.check("terraform init upgrades to local provider 99.99.99", () =>
        this.runOk("terraform", this.tfArgs(dir, "init", "-upgrade")))) { await this.destroy(dir); return; }
      await this.check(`upgrading from ${upgradeFrom} to 99.99.99 is a no-op`, () => this.cleanPlan(dir));
      if (this.request.selection.keep) this.record("SKIP", "terraform destroy removes the cluster - --keep was set");
      else await this.destroy(dir);
      return;
    } else {
      await this.check("terraform plan is clean after apply", () => this.cleanPlan(dir));
    }

    if (!await this.check("kubeconfig can be built from the provider outputs", async () => {
      await this.writeKubeconfig(dir, kubeconfig, clusterName); return true;
    })) { await this.destroy(dir); return; }

    const kube = (...args: string[]) => this.command("kubectl", this.kubeArgs(kubeconfig, ...args));
    await this.check("kubectl reaches the apiserver with the provider's credentials", async () =>
      this.retry(async () => (await kube("cluster-info")).stdout.includes("Kubernetes control plane")));
    await this.check(`all ${expectedNodes} node(s) report Ready`, async () => this.retry(async () => {
      const total = (await kube("get", "nodes", "--no-headers")).stdout.trim().split("\n").filter(Boolean).length;
      const ready = (await kube("get", "nodes", "-o", 'jsonpath={range .items[*]}{range .status.conditions[?(@.type=="Ready")]}{.status}{"\\n"}{end}{end}')).stdout.split("\n").filter((v) => v === "True").length;
      return total === expectedNodes && ready === expectedNodes;
    }, 300_000, 5_000));
    await this.check("kube-system pods are ready", () => this.runOk("kubectl", this.kubeArgs(kubeconfig, "-n", "kube-system", "wait", "--for=condition=Ready", "pods", "--all", "--timeout=300s")));
    await this.check("the storage-provisioner addon installed a default storage class", async () => (await kube("get", "storageclass", "-o", "jsonpath={.items[*].metadata.name}")).stdout.includes("standard"));

    if (this.request.selection.preloadImage) {
      const image = (await readFile(join(this.e2eDir, "manifests", "nginx.yaml"), "utf8")).match(/^\s*image:\s*(\S+)/m)?.[1] ?? "nginx:1.27-alpine";
      await this.check(`preload ${image} into the cluster`, async () => {
        let puller = "";
        for (const candidate of ["docker", "podman"]) {
          if ((await this.command(candidate, ["--version"])).code === 0) { puller = candidate; break; }
        }
        if (!puller) return false;
        if ((await this.command(puller, ["image", "inspect", image])).code !== 0 &&
            !await this.runOk(puller, ["pull", image])) return false;
        const tar = join(dir, "image.tar");
        return await this.runOk(puller, ["save", image, "-o", tar]) &&
          await this.runOk("minikube", ["-p", clusterName, "image", "load", tar]);
      });
    }

    const manifest = join(this.e2eDir, "manifests", "nginx.yaml");
    await this.check("nginx manifests apply", () => this.runOk("kubectl", this.kubeArgs(kubeconfig, "apply", "-f", manifest)));
    await this.check("nginx deployment rolls out", () => this.runOk("kubectl", this.kubeArgs(kubeconfig, "-n", "e2e", "rollout", "status", "deployment/nginx", "--timeout=300s")));
    await this.check("2 nginx pods are running", async () => (await kube("-n", "e2e", "get", "pods", "-l", "app=nginx", "--field-selector=status.phase=Running", "--no-headers")).stdout.trim().split("\n").filter(Boolean).length === 2);
    const endpoints = async (count: number) => this.retry(async () => (await kube("-n", "e2e", "get", "endpointslices", "-l", "kubernetes.io/service-name=nginx", "-o", "jsonpath={range .items[*]}{range .endpoints[?(@.conditions.ready==true)]}{.addresses[0]}{\"\\n\"}{end}{end}")).stdout.trim().split("\n").filter(Boolean).length === count);
    await this.check("the service has 2 ready endpoints", () => endpoints(2));
    await this.check("cluster DNS resolves the nginx service", async () => {
      const ip = clean((await kube("-n", "e2e", "get", "svc", "nginx", "-o", "jsonpath={.spec.clusterIP}")).stdout);
      return this.retry(async () => (await kube("-n", "e2e", "exec", "deploy/nginx", "--", "nslookup", "nginx.e2e.svc.cluster.local.")).stdout.includes(ip));
    });
    await this.check("nginx answers in-cluster over the service", async () =>
      this.retry(async () =>
        (await kube("-n", "e2e", "exec", "deploy/nginx", "--", "wget", "-q", "-O", "-", "-T", "10", "http://nginx.e2e.svc.cluster.local./"))
          .stdout.includes("terraform-provider-minikube e2e ok"),
      ),
    );
    await this.check("nginx answers on the host through a port-forward", () =>
      this.portForwardWorks(kubeconfig));
    const nodeIp = clean((await kube("get", "nodes", "-o", 'jsonpath={.items[0].status.addresses[?(@.type=="InternalIP")].address}')).stdout);
    if ((await this.command("curl", ["-sS", "-o", "/dev/null", "--connect-timeout", "5", "--max-time", "10", `http://${nodeIp}:30080/`])).code === 0)
      await this.check(`nginx answers on the node port at ${nodeIp}:30080`, async () => {
        const response = await this.command("curl", ["-fsS", "--max-time", "10", `http://${nodeIp}:30080/`]);
        return response.code === 0 && response.stdout.includes("terraform-provider-minikube e2e ok");
      });
    else this.record("SKIP", `nginx answers on the node port - node IP ${nodeIp || "unknown"} is not routable from the host with the ${flavour} driver`);
    await this.check("scaling the deployment to 3 updates the service endpoints", async () =>
      await this.runOk("kubectl", this.kubeArgs(kubeconfig, "-n", "e2e", "scale", "deployment/nginx", "--replicas=3")) &&
      await this.runOk("kubectl", this.kubeArgs(kubeconfig, "-n", "e2e", "rollout", "status", "deployment/nginx", "--timeout=300s")) &&
      await endpoints(3));
    await this.check("the workload can be deleted again", async () => {
      if (!await this.runOk("kubectl", this.kubeArgs(kubeconfig, "delete", "namespace", "e2e", "--timeout=180s"))) return false;
      return (await kube("get", "namespace", "e2e")).code !== 0;
    });

    if (this.request.selection.keep) this.record("SKIP", "terraform destroy removes the cluster - --keep was set");
    else await this.destroy(dir, kubeconfig);
  }

  private async portForwardWorks(kubeconfig: string): Promise<boolean> {
    if (!this.executor.start) {
      this.output("command executor cannot run a background port-forward");
      return false;
    }
    let port = "";
    let exited = false;
    const forward = this.executor.start("kubectl", [
      "--kubeconfig", kubeconfig, "-n", "e2e", "port-forward", "svc/nginx", ":80", "--address", "127.0.0.1",
    ], {
      cwd: this.root,
      env: this.env,
      signal: this.controller.signal,
      onOutput: (text) => {
        this.output(text);
        port ||= text.match(/Forwarding from 127\.0\.0\.1:(\d+)/)?.[1] ?? "";
      },
    });
    this.active = forward;
    void forward.completed.then(() => {
      exited = true;
      if (this.active === forward) this.active = undefined;
    });
    const deadline = Date.now() + 30_000;
    while (!port && !exited && Date.now() < deadline) await sleep(100, this.controller.signal);
    if (!port) { forward.cancel(); await forward.completed; return false; }
    const passed = await this.retry(async () => {
      const response = await this.command("curl", ["-fsS", "--max-time", "10", `http://127.0.0.1:${port}/`]);
      return response.code === 0 && response.stdout.includes("terraform-provider-minikube e2e ok");
    }, 60_000, 3_000);
    forward.cancel();
    await forward.completed;
    return passed;
  }

  private async destroy(dir: string, kubeconfig?: string): Promise<void> {
    if (await this.check("terraform destroy removes the cluster", () => this.runOk("terraform", this.tfArgs(dir, "destroy", "-auto-approve"), { timeoutMs: 900_000 }))) {
      this.cleanupDirs.delete(dir);
      await this.check("the cluster is gone after destroy", async () => {
        const state = await this.command("terraform", [`-chdir=${dir}`, "state", "list"]);
        if (state.stdout.trim()) return false;
        if (!kubeconfig) return true;
        return (await this.command("kubectl", this.kubeArgs(kubeconfig, "get", "nodes"), { timeoutMs: 10_000 })).code !== 0;
      });
    }
  }

  async run(): Promise<E2EProcessResult> {
    await mkdir(this.workDir, { recursive: true });
    try {
      if (!this.request.selection.skipBuild) {
        const built = await this.command("make", ["-C", this.root, "set-local"], { onOutput: (text) => this.output(text) });
        if (built.code !== 0) throw new Error(built.stderr || "provider build failed");
      }
      const cliConfig = join(this.root, "bin", "terraform-local.tfrc");
      try { await access(cliConfig); } catch { await this.command("make", ["-C", this.root, "local-cli-config"]); }
      this.env.TF_CLI_CONFIG_FILE = cliConfig;

      let upgradeFrom: string | undefined;
      if (this.request.mode === "upgrade-tests") {
        const response = await fetch("https://registry.terraform.io/v1/providers/scott-the-programmer/minikube", { signal: this.controller.signal });
        if (!response.ok) throw new Error("could not resolve the latest released provider version");
        upgradeFrom = String((await response.json() as { version?: string }).version ?? "");
        if (!upgradeFrom) throw new Error("Terraform Registry response did not contain a provider version");
      }
      for (const flavour of this.request.selection.flavours) {
        if (this.controller.signal.aborted) break;
        await this.runFlavour(flavour, upgradeFrom);
      }
    } catch (error) {
      this.output(`ERROR ${error instanceof Error ? error.message : String(error)}`);
      return { code: this.controller.signal.aborted ? 130 : 1, progress: this.progress() };
    } finally {
      if (!this.request.selection.keep) {
        for (const dir of this.cleanupDirs) {
          this.output(`cleaning up leftover cluster in ${dir}`);
          await this.executor.run("terraform", this.tfArgs(dir, "destroy", "-auto-approve"), {
            cwd: this.root,
            env: this.env,
            timeoutMs: 900_000,
          });
        }
        this.cleanupDirs.clear();
      }
    }
    return { code: this.controller.signal.aborted ? 130 : (this.progressState.failed ? 1 : 0), progress: this.progress() };
  }
}

/** Run the suite directly in TypeScript; run.sh remains only as a standalone CLI/CI entry point. */
export function startE2ERun(request: E2ERunRequest, onLine: (line: string) => void): RunningE2E {
  const suite = new NativeE2ESuite(request, onLine);
  const completed = suite.run();
  const label = request.mode === "upgrade-tests" ? "native provider upgrade suite" : "native minikube E2E suite";
  return { displayCommand: label, completed, progress: () => suite.progress(), cancel: () => suite.cancel() };
}
