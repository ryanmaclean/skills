#!/usr/bin/env node

/**
 * Datadog MCP Server
 *
 * Model Context Protocol server that wraps the Datadog CLI,
 * making all 22 Datadog commands accessible to AI agents
 * (Cursor, Claude Desktop, GitHub Copilot, etc.)
 */

import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { exec } from "child_process";
import { promisify } from "util";
import { resolve } from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";

const execAsync = promisify(exec);

// Get the directory of the current module
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Path to the Datadog CLI binary (adjust based on your setup)
const CLI_BINARY = process.env.DD_CLI_PATH || resolve(__dirname, "../../bin/dd-darwin-arm64");

/**
 * Execute Datadog CLI command and return parsed JSON response
 */
async function executeDDCommand(command: string, args: string[]): Promise<any> {
  const fullCommand = `${CLI_BINARY} ${command} ${args.join(" ")} --json`;

  try {
    const { stdout, stderr } = await execAsync(fullCommand, {
      env: {
        ...process.env,
        DD_API_KEY: process.env.DD_API_KEY,
        DD_APP_KEY: process.env.DD_APP_KEY,
        DD_SITE: process.env.DD_SITE || "datadoghq.com",
      },
      maxBuffer: 10 * 1024 * 1024, // 10MB buffer for large responses
    });

    if (stderr) {
      console.error(`CLI stderr: ${stderr}`);
    }

    // Parse JSON response
    try {
      return JSON.parse(stdout);
    } catch {
      // If not JSON, return as text
      return { output: stdout, raw: true };
    }
  } catch (error: any) {
    throw new Error(`Datadog CLI error: ${error.message}`);
  }
}

/**
 * Tool definitions for the 5 core Datadog operations
 */
const TOOLS = [
  {
    name: "datadog_health",
    description: "Check service health with multi-signal analysis. Evaluates error rates, response times, and active alerts to determine if a service is healthy.",
    inputSchema: {
      type: "object",
      properties: {
        service: {
          type: "string",
          description: "Service name to check (optional - auto-detects from git context if not provided)",
        },
        from: {
          type: "string",
          description: "Time range to analyze (e.g., '1h', '6h', '24h'). Default: 1h",
        },
      },
    },
  },
  {
    name: "datadog_deploy",
    description: "Validate deployment safety before deploying. Checks for active incidents, high error rates, and alerting monitors to determine if it's safe to deploy.",
    inputSchema: {
      type: "object",
      properties: {
        service: {
          type: "string",
          description: "Service name to validate (optional - auto-detects from git context)",
        },
        environment: {
          type: "string",
          description: "Environment to check (e.g., 'production', 'staging')",
        },
      },
    },
  },
  {
    name: "datadog_apm",
    description: "Query APM trace analytics for a service. Returns traces, error rates, latency percentiles (p50, p95, p99), and throughput metrics.",
    inputSchema: {
      type: "object",
      properties: {
        service: {
          type: "string",
          description: "Service name to query traces for",
        },
        from: {
          type: "string",
          description: "Time range (e.g., '1h', '24h', '7d'). Default: 1h",
        },
        status: {
          type: "string",
          description: "Filter by status: 'error', 'ok', or 'all'. Default: all",
          enum: ["error", "ok", "all"],
        },
        resource: {
          type: "string",
          description: "Filter by resource name (e.g., 'GET /api/users')",
        },
      },
      required: ["service"],
    },
  },
  {
    name: "datadog_logs",
    description: "Search and retrieve logs with filtering. Supports full-text search, status filtering, and time ranges.",
    inputSchema: {
      type: "object",
      properties: {
        query: {
          type: "string",
          description: "Log search query (e.g., 'error', 'status:500', 'service:api AND error')",
        },
        service: {
          type: "string",
          description: "Filter by service name",
        },
        from: {
          type: "string",
          description: "Time range (e.g., '1h', '24h', '7d'). Default: 1h",
        },
        status: {
          type: "string",
          description: "Filter by log status: 'error', 'warn', 'info', 'debug'",
          enum: ["error", "warn", "info", "debug"],
        },
        limit: {
          type: "number",
          description: "Maximum number of logs to return (default: 100, max: 1000)",
        },
      },
    },
  },
  {
    name: "datadog_incidents",
    description: "Manage Datadog incidents. List active incidents, create new incidents, update status, or close incidents.",
    inputSchema: {
      type: "object",
      properties: {
        action: {
          type: "string",
          description: "Action to perform: 'list', 'create', 'update', or 'close'",
          enum: ["list", "create", "update", "close"],
        },
        status: {
          type: "string",
          description: "Filter incidents by status (for list action): 'active', 'stable', 'resolved'",
          enum: ["active", "stable", "resolved"],
        },
        service: {
          type: "string",
          description: "Filter by service name (for list action)",
        },
        title: {
          type: "string",
          description: "Incident title (for create action)",
        },
        severity: {
          type: "string",
          description: "Incident severity (for create action): 'SEV-1' through 'SEV-5'",
          enum: ["SEV-1", "SEV-2", "SEV-3", "SEV-4", "SEV-5"],
        },
        incident_id: {
          type: "string",
          description: "Incident ID (for update/close actions)",
        },
        new_status: {
          type: "string",
          description: "New status for update action",
          enum: ["active", "stable", "resolved"],
        },
      },
      required: ["action"],
    },
  },
];

/**
 * Build CLI arguments from tool input
 */
function buildCliArgs(toolName: string, input: Record<string, any>): string[] {
  const args: string[] = [];

  // Handle incidents special case with action
  if (toolName === "datadog_incidents") {
    const action = input.action;
    if (action === "list") {
      if (input.status) args.push("--status", input.status);
      if (input.service) args.push("--service", input.service);
    } else if (action === "create") {
      if (input.title) args.push("--title", input.title);
      if (input.severity) args.push("--severity", input.severity);
      if (input.service) args.push("--service", input.service);
    } else if (action === "update") {
      if (input.incident_id) args.push("--id", input.incident_id);
      if (input.new_status) args.push("--status", input.new_status);
    } else if (action === "close") {
      if (input.incident_id) args.push("--id", input.incident_id);
    }
    return args;
  }

  // Standard parameter mapping
  for (const [key, value] of Object.entries(input)) {
    if (value !== undefined && value !== null) {
      // Convert camelCase to kebab-case for CLI flags
      const flag = key.replace(/([A-Z])/g, "-$1").toLowerCase();
      args.push(`--${flag}`, String(value));
    }
  }

  return args;
}

/**
 * Format CLI response for better readability
 */
function formatResponse(command: string, data: any): string {
  if (data.raw) {
    return data.output;
  }

  // Pretty-print JSON with context
  const formatted = JSON.stringify(data, null, 2);
  return `Datadog ${command} results:\n\n${formatted}`;
}

/**
 * Main server setup
 */
async function main() {
  const server = new Server(
    {
      name: "datadog-mcp-server",
      version: "0.1.0",
    },
    {
      capabilities: {
        tools: {},
      },
    }
  );

  // List available tools
  server.setRequestHandler(ListToolsRequestSchema, async () => {
    return {
      tools: TOOLS,
    };
  });

  // Execute tool calls
  server.setRequestHandler(CallToolRequestSchema, async (request) => {
    const { name, arguments: args } = request.params;

    try {
      // Extract command from tool name (datadog_health -> health)
      const command = name.replace("datadog_", "");

      // Build CLI arguments
      const cliArgs = buildCliArgs(name, args || {});

      // Execute command
      const result = await executeDDCommand(command, cliArgs);

      // Format response
      const formattedResult = formatResponse(command, result);

      return {
        content: [
          {
            type: "text",
            text: formattedResult,
          },
        ],
      };
    } catch (error: any) {
      return {
        content: [
          {
            type: "text",
            text: `Error executing ${name}: ${error.message}`,
          },
        ],
        isError: true,
      };
    }
  });

  // Start server
  const transport = new StdioServerTransport();
  await server.connect(transport);

  console.error("Datadog MCP Server running on stdio");
  console.error(`CLI binary: ${CLI_BINARY}`);
  console.error(`Tools available: ${TOOLS.length}`);
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
