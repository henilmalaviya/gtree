import { octokit } from "./github";

export async function fetchDefaultBranch(owner: string, repo: string) {
  const response = await octokit.request(`GET /repos/${owner}/${repo}`);

  if (response.status !== 200) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  const data = response.data;

  return data.default_branch || "main";
}
