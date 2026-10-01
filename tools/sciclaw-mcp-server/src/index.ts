import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import { timingSafeEqual } from "node:crypto";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { z } from "zod";
import { allPages } from "./pagination.js";
import { phaseOnePath } from "./phase-file.js";

type Config = {
  giteaApiBaseUrl: string;
  giteaToken: string;
  giteaOwner: string;
  giteaRepository: string;
  mcpApiKey: string;
};

type GiteaIssue = {
  id: number;
  index: number;
  title: string;
  html_url: string;
};

type ProjectColumn = { id: number; title: string };
type GiteaContent = { type: string; content?: string; encoding?: string; sha: string };

const required = (name: string): string => {
  const value = process.env[name];
  if (!value) throw new Error(`${name} must be set`);
  return value;
};

function apiBaseUrl(value: string): string {
  const url = new URL(value);
  if (!url.pathname.endsWith("/")) url.pathname += "/";
  return url.toString();
}

const config: Config = {
  giteaApiBaseUrl: apiBaseUrl(required("GITEA_API_BASE_URL")),
  giteaToken: required("GITEA_TOKEN"),
  giteaOwner: required("GITEA_OWNER"),
  giteaRepository: required("GITEA_REPOSITORY"),
  mcpApiKey: required("MCP_API_KEY"),
};

const segment = (value: string | number) => encodeURIComponent(String(value));
const filePath = (value: string) => value.split("/").map(segment).join("/");

async function gitea<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(new URL(path, config.giteaApiBaseUrl), {
    ...init,
    headers: {
      Authorization: `token ${config.giteaToken}`,
      Accept: "application/json",
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });

  if (!response.ok) throw new Error(`Gitea request failed (${response.status}): ${await response.text()}`);
  return response.status === 204 ? (undefined as T) : (await response.json()) as T;
}

async function giteaList<T>(path: string): Promise<T[]> {
  return allPages(async (page) => {
    const separator = path.includes("?") ? "&" : "?";
    return gitea<T[]>(`${path}${separator}page=${page}&limit=100`);
  });
}

function result(data: unknown) {
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

function createMcpServer(): McpServer {
  const server = new McpServer({ name: "sciclaw-mcp-server", version: "0.1.0" });
  const registerTool = server.registerTool.bind(server) as (
    name: string,
    definition: { description: string; inputSchema: z.ZodRawShape },
    callback: (arguments_: Record<string, any>) => Promise<ReturnType<typeof result>>,
  ) => unknown;

  registerTool(
    "move_work_item_phase",
    {
      description: "Move a Gitea issue representing an AtomicQMS work item from one named project-column phase to another.",
      inputSchema: {
        organization: z.string().min(1),
        projectId: z.number().int().positive(),
        owner: z.string().min(1),
        repository: z.string().min(1),
        issueNumber: z.number().int().positive(),
        fromPhase: z.string().min(1),
        toPhase: z.string().min(1),
      } as z.ZodRawShape,
    },
    async ({ organization, projectId, owner, repository, issueNumber, fromPhase, toPhase }) => {
      if (fromPhase === toPhase) throw new Error("fromPhase and toPhase must differ");
      const prefix = `orgs/${segment(organization)}/projects/${segment(projectId)}`;
      const columns = await giteaList<ProjectColumn>(`${prefix}/columns`);
      const source = columns.find((column) => column.title === fromPhase);
      const destination = columns.find((column) => column.title === toPhase);
      if (!source) throw new Error(`Project phase not found: ${fromPhase}`);
      if (!destination) throw new Error(`Project phase not found: ${toPhase}`);

      const issue = await gitea<GiteaIssue>(`repos/${segment(owner)}/${segment(repository)}/issues/${segment(issueNumber)}`);
      const sourceIssues = await giteaList<GiteaIssue>(`${prefix}/columns/${segment(source.id)}/issues`);
      if (!sourceIssues.some((candidate) => candidate.id === issue.id)) {
        throw new Error(`Issue #${issueNumber} is not in phase ${fromPhase}`);
      }

      await gitea<void>(`${prefix}/issues/${segment(issue.id)}/move`, {
        method: "POST",
        body: JSON.stringify({ column_id: destination.id }),
      });
      return result({ issueNumber, fromPhase, toPhase, moved: true });
    },
  );

  registerTool(
    "move_file_to_phase_1",
    {
      description: "Move one exact file in the configured repository from phase_0/ to its matching path under phase_1/ on a specified branch.",
      inputSchema: {
        branch: z.string().min(1),
        filePath: z.string().min(1),
      } as z.ZodRawShape,
    },
    async ({ branch, filePath: sourcePath }) => {
      const destinationPath = phaseOnePath(sourcePath);
      const repositoryPath = `repos/${segment(config.giteaOwner)}/${segment(config.giteaRepository)}/contents`;
      const source = await gitea<GiteaContent>(`${repositoryPath}/${filePath(sourcePath)}?ref=${segment(branch)}`);
      if (source.type !== "file" || source.encoding !== "base64" || source.content === undefined) {
        throw new Error("filePath must identify a regular file");
      }

      const message = `Move ${sourcePath} to ${destinationPath}`;
      await gitea<void>(`${repositoryPath}/${filePath(destinationPath)}`, {
        method: "POST",
        body: JSON.stringify({ branch, content: source.content, message }),
      });
      try {
        // ponytail: two Gitea content operations; use the Git data API if atomic moves matter.
        await gitea<void>(`${repositoryPath}/${filePath(sourcePath)}`, {
          method: "DELETE",
          body: JSON.stringify({ branch, message, sha: source.sha }),
        });
      } catch (error) {
        throw new Error(`Created ${destinationPath}, but could not delete ${sourcePath}: ${String(error)}`);
      }
      return result({ sourcePath, destinationPath, branch, moved: true });
    },
  );

  registerTool(
    "create_change_request",
    {
      description: "Create a Gitea pull request (AtomicQMS change request) from an existing branch.",
      inputSchema: {
        owner: z.string().min(1),
        repository: z.string().min(1),
        title: z.string().min(1),
        head: z.string().min(1),
        base: z.string().min(1),
        body: z.string().optional(),
        reviewers: z.array(z.string().min(1)).optional(),
      } as z.ZodRawShape,
    },
    async ({ owner, repository, title, head, base, body, reviewers }) => {
      const changeRequest = await gitea<{ number: number; html_url: string }>(
        `repos/${segment(owner)}/${segment(repository)}/pulls`,
        {
          method: "POST",
          body: JSON.stringify({ title, head, base, body, reviewers }),
        },
      );
      return result({ number: changeRequest.number, url: changeRequest.html_url, created: true });
    },
  );

  return server;
}

function authorized(request: IncomingMessage): boolean {
  const header = request.headers.authorization;
  const expected = `Bearer ${config.mcpApiKey}`;
  if (!header || header.length !== expected.length) return false;
  return timingSafeEqual(Buffer.from(header), Buffer.from(expected));
}

function respond(response: ServerResponse, statusCode: number, body: string): void {
  response.writeHead(statusCode, { "Content-Type": "application/json" });
  response.end(body);
}

const server = createServer(async (request, response) => {
  if (request.url === "/health") return respond(response, 200, JSON.stringify({ status: "ok" }));
  if (request.url !== "/mcp") return respond(response, 404, JSON.stringify({ error: "Not found" }));
  if (!authorized(request)) return respond(response, 401, JSON.stringify({ error: "Unauthorized" }));

  const mcpServer = createMcpServer();
  const transport = new StreamableHTTPServerTransport({ sessionIdGenerator: undefined });
  await mcpServer.connect(transport);
  await transport.handleRequest(request, response);
});

const port = Number(process.env.PORT ?? 3000);
server.listen(port, () => console.log(`SciClaw MCP server listening on http://127.0.0.1:${port}/mcp`));
