import { Octokit } from "@octokit/core";

export const octokit = new Octokit({
  auth: Bun.env.GITHUB_TOKEN,
});
