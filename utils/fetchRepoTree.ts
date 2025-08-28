export type TreeNode = {
  path: string;
  type: string;
};

export type ApiResponse = {
  tree: TreeNode[];
};

export async function fetchRepoTree(
  owner: string,
  repo: string,
  branch: string
) {
  const response = await fetch(
    `https://api.github.com/repos/${owner}/${repo}/git/trees/${branch}?recursive=true`
  );

  if (response.status !== 200) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  const data = await response.json();

  return data as ApiResponse;
}
