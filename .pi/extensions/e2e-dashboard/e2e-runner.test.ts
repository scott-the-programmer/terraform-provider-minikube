import assert from "node:assert/strict";
import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";

import { startE2ERun, type E2ECommandExecutor } from "./e2e-runner.ts";

test("startE2ERun runs the E2E lifecycle directly without invoking run.sh", async () => {
  const cwd = await mkdtemp(join(tmpdir(), "minikube-e2e-runner-"));
  await mkdir(join(cwd, "test", "e2e", "stack"), { recursive: true });
  await mkdir(join(cwd, "test", "e2e", "flavours"), { recursive: true });
  await mkdir(join(cwd, "test", "e2e", "manifests"), { recursive: true });
  await writeFile(join(cwd, "test/e2e/stack/versions.tf"), 'version = "99.99.99"\n');
  await writeFile(join(cwd, "test/e2e/stack/main.tf"), "");
  await writeFile(join(cwd, "test/e2e/flavours/docker.tfvars"), 'cluster_name = "tf-minikube-e2e-docker"\nnodes = 1\n');
  await writeFile(join(cwd, "test/e2e/flavours/docker.env"), 'E2E_DESCRIPTION="docker"\n');
  await writeFile(join(cwd, "test/e2e/manifests/nginx.yaml"), "image: nginx:1.27-alpine\n");

  const calls: string[] = [];
  let scaled = false;
  let namespaceDeleted = false;
  let destroyed = false;
  const executor: E2ECommandExecutor = {
    async run(command, args) {
      calls.push([command, ...args].join(" "));
      const joined = args.join(" ");
      if (command === "terraform" && joined.includes("destroy")) destroyed = true;
      if (command === "kubectl" && joined.includes("scale")) scaled = true;
      if (command === "kubectl" && joined.includes("delete namespace e2e")) namespaceDeleted = true;
      if (command === "kubectl" && namespaceDeleted && joined.endsWith("get namespace e2e")) {
        return { code: 1, stdout: "", stderr: "not found" };
      }
      if (command === "kubectl" && destroyed && joined.endsWith("get nodes")) {
        return { code: 1, stdout: "", stderr: "gone" };
      }
      if (command === "terraform" && joined.includes("output") && joined.endsWith("host")) {
        return { code: 0, stdout: "https://127.0.0.1:8443", stderr: "" };
      }
      if (command === "terraform" && joined.includes("output") && joined.includes("-raw")) {
        return { code: 0, stdout: "-----BEGIN CERTIFICATE-----\ndata\n-----END CERTIFICATE-----", stderr: "" };
      }
      if (command === "terraform" && joined.includes("state list")) {
        return { code: 0, stdout: "", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("cluster-info")) {
        return { code: 0, stdout: "Kubernetes control plane", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get nodes -o jsonpath")) {
        return { code: 0, stdout: joined.includes("InternalIP") ? "127.0.0.1" : "True\n", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get nodes --no-headers")) {
        return { code: 0, stdout: "node Ready", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get pods") && joined.includes("status.phase=Running")) {
        return { code: 0, stdout: "nginx-1 Running\nnginx-2 Running\n", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get storageclass")) {
        return { code: 0, stdout: "standard", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get svc nginx")) {
        return { code: 0, stdout: "10.0.0.1", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get endpointslices")) {
        return { code: 0, stdout: scaled ? "a\nb\nc\n" : "a\nb\n", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("nslookup")) {
        return { code: 0, stdout: "Address: 10.0.0.1", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("wget")) {
        return { code: 0, stdout: "terraform-provider-minikube e2e ok", stderr: "" };
      }
      if (command === "curl" && joined.includes("127.0.0.1")) {
        return { code: 0, stdout: "terraform-provider-minikube e2e ok", stderr: "" };
      }
      if (command === "curl") return { code: 1, stdout: "", stderr: "not routable" };
      return { code: 0, stdout: "", stderr: "" };
    },
    start(command, args, options) {
      calls.push([command, ...args].join(" "));
      options?.onOutput?.("Forwarding from 127.0.0.1:43123 -> 80\n");
      let finish!: (result: { code: number; stdout: string; stderr: string }) => void;
      const completed = new Promise<{ code: number; stdout: string; stderr: string }>((resolve) => { finish = resolve; });
      return {
        completed,
        cancel: () => finish({ code: 130, stdout: "", stderr: "" }),
      };
    },
  };

  try {
    const lines: string[] = [];
    const run = startE2ERun(
      {
        cwd,
        mode: "e2e",
        selection: { flavours: ["docker"], keep: false, skipBuild: false, preloadImage: false },
        executor,
      },
      (line) => lines.push(line),
    );
    const result = await run.completed;

    assert.equal(result.code, 0, lines.join("\n"));
    assert.ok(calls.some((call) => call.startsWith("make -C")));
    assert.ok(calls.some((call) => call.includes("terraform") && call.includes("apply")));
    assert.ok(calls.some((call) => call.includes("kubectl") && call.includes("apply")));
    assert.ok(calls.some((call) => call.includes("terraform") && call.includes("destroy")));
    assert.ok(calls.every((call) => !call.includes("run.sh")));
    assert.ok(result.progress.passed > 0);
    assert.ok(lines.some((line) => line.includes("FLAVOUR: docker")));
  } finally {
    await rm(cwd, { recursive: true, force: true });
  }
});

test("upgrade-tests changes only the provider version and stops after the no-op plan", async () => {
  const cwd = await mkdtemp(join(tmpdir(), "minikube-upgrade-runner-"));
  await mkdir(join(cwd, "test", "e2e", "stack"), { recursive: true });
  await mkdir(join(cwd, "test", "e2e", "flavours"), { recursive: true });
  await writeFile(
    join(cwd, "test/e2e/stack/versions.tf"),
    'terraform {\n  required_version = ">= 1.0"\n  required_providers {\n    minikube = {\n      version = "99.99.99"\n    }\n  }\n}\n',
  );
  await writeFile(join(cwd, "test/e2e/stack/main.tf"), "");
  await writeFile(join(cwd, "test/e2e/flavours/docker.tfvars"), 'cluster_name = "tf-minikube-e2e-docker"\n');
  await writeFile(join(cwd, "test/e2e/flavours/docker.env"), 'E2E_DESCRIPTION="docker"\n');

  const calls: Array<{ command: string; args: string[]; env?: NodeJS.ProcessEnv }> = [];
  let endpointReads = 0;
  const executor: E2ECommandExecutor = {
    async run(command, args, options) {
      calls.push({ command, args, env: options?.env });
      const joined = args.join(" ");
      if (command === "terraform" && joined.includes("output") && joined.endsWith("host")) {
        return { code: 0, stdout: "https://127.0.0.1:8443", stderr: "" };
      }
      if (command === "terraform" && joined.includes("output")) {
        return { code: 0, stdout: "-----BEGIN CERTIFICATE-----\ndata\n-----END CERTIFICATE-----", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("cluster-info")) {
        return { code: 0, stdout: "Kubernetes control plane", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get nodes --no-headers")) {
        return { code: 0, stdout: "node Ready", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get nodes -o jsonpath")) {
        return { code: 0, stdout: joined.includes("InternalIP") ? "127.0.0.1" : "True\n", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get pods")) {
        return { code: 0, stdout: "nginx-1 Running\nnginx-2 Running\n", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get storageclass")) {
        return { code: 0, stdout: "standard", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get svc nginx")) {
        return { code: 0, stdout: "10.0.0.1", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("get endpointslices")) {
        endpointReads++;
        return { code: 0, stdout: endpointReads === 1 ? "a\nb\n" : "a\nb\nc\n", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("nslookup")) {
        return { code: 0, stdout: "Address: 10.0.0.1", stderr: "" };
      }
      if (command === "kubectl" && joined.includes("wget")) {
        return { code: 0, stdout: "terraform-provider-minikube e2e ok", stderr: "" };
      }
      if (command === "curl") return { code: 1, stdout: "", stderr: "not routable" };
      return { code: 0, stdout: "", stderr: "" };
    },
  };
  const originalFetch = globalThis.fetch;
  globalThis.fetch = async () => new Response(JSON.stringify({ version: "0.8.0" }), { status: 200 });

  try {
    const lines: string[] = [];
    const result = await startE2ERun({
      cwd,
      mode: "upgrade-tests",
      selection: { flavours: ["docker"], keep: false, skipBuild: true, preloadImage: false },
      executor,
    }, (line) => lines.push(line)).completed;

    assert.equal(result.code, 0, lines.join("\n"));
    assert.equal(calls.filter(({ command, args }) => command === "terraform" && args.includes("plan")).length, 2);
    assert.ok(calls.some(({ command, args, env }) =>
      command === "terraform" && args.includes("init") && env?.TF_CLI_CONFIG_FILE === undefined));
    assert.ok(calls.some(({ command, args, env }) =>
      command === "terraform" && args.includes("init") && env?.TF_CLI_CONFIG_FILE?.endsWith("terraform-local.tfrc")));
    assert.ok(calls.every(({ command }) => command !== "kubectl"));

    const versions = await readFile(join(cwd, "test/e2e/.work/docker/versions.tf"), "utf8");
    assert.match(versions, /required_version = ">= 1\.0"/);
    assert.match(versions, /version = "99\.99\.99"/);
    assert.ok(lines.some((line) => line.includes("upgrading from 0.8.0 to 99.99.99 is a no-op")));
  } finally {
    globalThis.fetch = originalFetch;
    await rm(cwd, { recursive: true, force: true });
  }
});

test("startE2ERun cancels an active native command", async () => {
  const cwd = await mkdtemp(join(tmpdir(), "minikube-e2e-cancel-"));
  let commandStarted!: () => void;
  const started = new Promise<void>((resolve) => { commandStarted = resolve; });
  const executor: E2ECommandExecutor = {
    async run(_command, _args, options) {
      commandStarted();
      return new Promise((resolve) => {
        options?.signal?.addEventListener(
          "abort",
          () => resolve({ code: 130, stdout: "", stderr: "cancelled" }),
          { once: true },
        );
      });
    },
  };

  try {
    const run = startE2ERun({
      cwd,
      mode: "e2e",
      selection: { flavours: ["docker"], keep: false, skipBuild: false, preloadImage: false },
      executor,
    }, () => {});
    await started;
    run.cancel();

    assert.equal((await run.completed).code, 130);
  } finally {
    await rm(cwd, { recursive: true, force: true });
  }
});
