import { Hono } from "hono";
import { getConnInfo, handle } from "hono/vercel";
import { fetchDefaultBranch } from "../utils/fetchDefaultBranch";
import { trimTrailingSlash } from "hono/trailing-slash";
import { fetchRepoTree } from "../utils/fetchRepoTree";
import { buildTree } from "../utils/buildTree";
import { rateLimiter } from "hono-rate-limiter";
import { RedisStore } from "@hono-rate-limiter/redis";
import { kv } from "@vercel/kv";

export const config = {
  runtime: "edge",
};

const app = new Hono();

app.use(trimTrailingSlash());

app.get("/", (c) => {
  const explanation = `
Git Tree (gtree)
-----------------------------

This is a simple utility service that displays the directory structure of any public GitHub repository
in a tree-like format, similar to the Linux 'tree' command.

Usage:
GET /:owner/:repo
GET /:owner/:repo/:branch

Parameters:
- owner: GitHub username or organization name (required)
- repo: Repository name (required)
- branch: Branch name (optional, defaults to the repository's default branch)

Examples:
- /henilmalaviya/gtree         # Shows the default branch tree for henilmalaviya/gtree
- /henilmalaviya/gtree/main    # Shows the 'main' branch tree for henilmalaviya/gtree

Potential Use Cases:
- Enhancing LLM understanding of repository structure by providing a clear tree view
- Quick overview of repository structure without cloning
- Documentation generation for project layouts
- Comparing directory structures across branches

The output format mimics the Linux 'tree' command, making it easy to visualize the repository's
file hierarchy. This service fetches data directly from the GitHub API and generates the tree view
on-demand.

Note: This service only works with public repositories due to GitHub API restrictions.
	`.trim();

  return c.text(explanation);
});

app.use(
  rateLimiter({
    // 60 seconds
    windowMs: 1000 * 60,
    limit: 100,
    message: "Too many requests, we are detecting abuse.",
    keyGenerator(c) {
      const info = getConnInfo(c);
      return info.remote.address || "unknown";
    },
    store: new RedisStore({ client: kv }),
  })
);

app.get("/:owner/:repo/:branch?", async (c) => {
  let { owner, repo, branch } = c.req.param();

  if (!branch) {
    branch = await fetchDefaultBranch(owner, repo);
  }

  const { tree } = await fetchRepoTree(owner, repo, branch!);

  const treeString = buildTree(tree, owner, repo, branch!);

  // Set cache headers for Vercel Edge
  c.header("Cache-Control", "s-maxage=600, stale-while-revalidate=60");

  return c.text(treeString);
});

export default handle(app);
