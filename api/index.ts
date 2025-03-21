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
