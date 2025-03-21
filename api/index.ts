import { Hono } from "hono";
import { handle } from "hono/vercel";
import { fetchDefaultBranch } from "../utils/fetchDefaultBranch";
import { trimTrailingSlash } from "hono/trailing-slash";
import { fetchRepoTree } from "../utils/fetchRepoTree";
import { buildTree } from "../utils/buildTree";

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
- /henilmalaviya/gtree         # Shows the default branch tree for nexusog/link-api
- /henilmalaviya/gtree/main    # Shows the 'main' branch tree for nexusog/link-api

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

app.get("/:owner/:repo/:branch?", async (c) => {
  let { owner, repo, branch } = c.req.param();

  if (!branch) {
    branch = await fetchDefaultBranch(owner, repo);
  }

  const { tree } = await fetchRepoTree(owner, repo, branch!);

  const treeString = buildTree(tree, owner, repo, branch!);

  return c.text(treeString);
});

export default handle(app);
