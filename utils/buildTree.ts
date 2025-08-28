import { TreeNode } from "./fetchRepoTree";

export function buildTree(
  treeData: TreeNode[],
  owner: string,
  repo: string,
  branch: string
): string {
  const treeMap = new Map<string, { children: string[]; isDir: boolean }>();
  const rootName = `${owner}/${repo}:${branch}`;

  treeMap.set(rootName, { children: [], isDir: true });

  treeData.forEach((item) => {
    const parts = item.path.split("/");
    let currentPath = rootName;

    parts.forEach((part, index) => {
      const fullPath =
        currentPath === rootName
          ? `${currentPath}/${part}`
          : `${currentPath}/${part}`;

      if (!treeMap.has(fullPath)) {
        treeMap.set(fullPath, {
          children: [],
          isDir: index < parts.length - 1 || item.type === "tree",
        });
      }

      if (!treeMap.get(currentPath)!.children.includes(part)) {
        treeMap.get(currentPath)!.children.push(part);
      }

      currentPath = fullPath;
    });
  });

  let output = `${rootName}\n`;
  const processed = new Set<string>();

  function buildLevel(path: string, prefix: string = ""): void {
    if (processed.has(path)) return;
    processed.add(path);

    const entry = treeMap.get(path)!;
    const children = entry.children.sort();

    children.forEach((child, index) => {
      const childPath = `${path}/${child}`;
      if (!treeMap.has(childPath)) return;

      const isLast = index === children.length - 1;
      const newPrefix = prefix + (isLast ? "    " : "│   ");
      const connector = isLast ? "└── " : "├── ";

      output += `${prefix}${connector}${child}${
        treeMap.get(childPath)!.isDir ? "/" : ""
      }\n`;
      buildLevel(childPath, newPrefix);
    });
  }

  buildLevel(rootName);

  const dirs =
    Array.from(treeMap.values()).filter((item) => item.isDir).length - 1;
  const files = Array.from(treeMap.values()).filter(
    (item) => !item.isDir
  ).length;
  output += `\n${dirs} directories, ${files} files`;

  return output;
}
