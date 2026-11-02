import { execFile } from "node:child_process";
import { promisify } from "node:util";

const run = promisify(execFile);

export const name = "nodepulse";
export const inject = { optional: ["tools"] };

function probe(ctx, service, method) {
  try {
    const svc = ctx.reflect?.get(service, true);
    if (svc && typeof svc[method] === "function") return svc;
  } catch {
    // service not composed
  }
  return undefined;
}

function pick(args, config, key, fallback) {
  const value = args[key] ?? config[key];
  return value === undefined || value === null || value === "" ? fallback : value;
}

function buildCommand(args, config) {
  const action = String(args.action || "").trim();
  if (!action) throw new Error("action is required");
  const bin = config.binary || "nodepulse";
  const configPath = pick(args, config, "config_path", "");
  const api = pick(args, config, "api", "");
  const secret = pick(args, config, "secret", "");
  const results = pick(args, config, "results_path", "results.json");
  const argv = [];

  const need = (cond, message) => {
    if (!cond) throw new Error(message);
  };

  switch (action) {
    case "list":
      need(configPath, "config_path is required for list");
      argv.push("nodes", "-f", configPath);
      break;
    case "test": {
      need(configPath, "config_path is required for test");
      argv.push("test", "-f", configPath, "--out", results);
      if (api) argv.push("--api", api);
      if (secret) argv.push("--secret", secret);
      const mihomo = pick(args, config, "mihomo_path", "");
      if (mihomo) argv.push("--mihomo", mihomo);
      if (args.rounds) argv.push("--rounds", String(args.rounds));
      if (args.url) argv.push("--url", String(args.url));
      if (args.timeout_ms) argv.push("--timeout", `${args.timeout_ms}ms`);
      break;
    }
    case "rank":
      argv.push("rank", "-i", results);
      break;
    case "switch": {
      need(api, "api is required for switch");
      need(args.group, "group is required for switch");
      need(args.node, "node is required for switch");
      argv.push("switch", "--api", api, "--group", String(args.group), "--node", String(args.node));
      if (secret) argv.push("--secret", secret);
      break;
    }
    case "export": {
      need(configPath, "config_path is required for export");
      argv.push("export", "-f", configPath, "-i", results);
      if (args.out) argv.push("-o", String(args.out));
      if (args.min_success !== undefined) argv.push("--min-success", String(args.min_success));
      break;
    }
    case "status":
      need(api, "api is required for status");
      argv.push("status", "--api", api);
      if (secret) argv.push("--secret", secret);
      break;
    case "history":
      argv.push("history", "-n", String(args.n || 20));
      break;
    default:
      throw new Error(`unknown action: ${action}`);
  }
  return { bin, argv };
}

export function apply(ctx, config = {}) {
  const tools = probe(ctx, "tools", "register");
  if (!tools) {
    ctx.logger?.warn?.("nodepulse: tools service is not available; plugin inactive");
    return;
  }

  const tool = {
    name: "nodepulse",
    description:
      "Proxy node stability tool. Actions: list nodes, test nodes (multi-round success/P90/jitter), rank results, switch a Clash Selector, export a filtered config, show controller status, or show switch history.",
    parameters: {
      type: "object",
      properties: {
        action: {
          type: "string",
          enum: ["list", "test", "rank", "switch", "export", "status", "history"],
          description: "Operation to perform.",
        },
        config_path: { type: "string", description: "Clash/mihomo YAML subscription path." },
        api: { type: "string", description: "Clash controller URL, e.g. http://127.0.0.1:9090." },
        secret: { type: "string", description: "Clash controller secret." },
        group: { type: "string", description: "Selector group name (switch)." },
        node: { type: "string", description: "Node name (switch)." },
        results_path: { type: "string", description: "results.json path (default results.json)." },
        out: { type: "string", description: "Output YAML for export (default stable.yaml)." },
        min_success: { type: "number", description: "Minimum success rate for export (0..1)." },
        rounds: { type: "integer", description: "Rounds per node for test (default 5)." },
        url: { type: "string", description: "Test URL for test." },
        timeout_ms: { type: "integer", description: "Per-round timeout in milliseconds." },
        mihomo_path: { type: "string", description: "mihomo executable for sidecar mode." },
        n: { type: "integer", description: "Number of history events (default 20)." },
      },
      required: ["action"],
    },
    output: {
      schema: { type: "string" },
      render: (_args, value) => [{ type: "text", text: String(value) }],
    },
    async execute(args, exec) {
      const { bin, argv } = buildCommand(args, config);
      try {
        const { stdout, stderr } = await run(bin, argv, {
          signal: exec?.signal,
          maxBuffer: 8 * 1024 * 1024,
        });
        const extra = stderr?.trim();
        return [stdout?.trim(), extra].filter(Boolean).join("\n");
      } catch (err) {
        const detail = err?.stderr?.trim() || err?.message || String(err);
        throw new Error(`nodepulse ${argv.join(" ")} failed: ${detail}`);
      }
    },
  };

  const dispose = tools.register(tool);
  if (typeof ctx.effect === "function" && dispose) {
    ctx.effect(() => dispose, "nodepulse: tool");
  }
}
