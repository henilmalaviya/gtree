import { Elysia } from "elysia";
import { fetchDefaultBranch } from "../utils/fetchDefaultBranch";
import { fetchRepoTree } from "../utils/fetchRepoTree";
import { buildTree } from "../utils/buildTree";

// Token Bucket rate limiter (burst + smooth refill) per IP
// Config: capacity (max burst), refillRate (tokens added per second)
const RATE_CAPACITY = 50; // allow short bursts
const REFILL_RATE = 100 / 60; // ~1.666 tokens/sec -> ~100 req/min sustained
type Bucket = { tokens: number; last: number };
const buckets = new Map<string, Bucket>();

function takeToken(ip: string): { allowed: boolean; remaining: number } {
  const now = Date.now();
  let b = buckets.get(ip);
  if (!b) {
    b = { tokens: RATE_CAPACITY, last: now };
    buckets.set(ip, b);
  }
  // Refill
  const elapsedSec = (now - b.last) / 1000;
  if (elapsedSec > 0) {
    b.tokens = Math.min(RATE_CAPACITY, b.tokens + elapsedSec * REFILL_RATE);
    b.last = now;
  }
  if (b.tokens >= 1) {
    b.tokens -= 1;
    return { allowed: true, remaining: Math.floor(b.tokens) };
  }
  return { allowed: false, remaining: Math.floor(b.tokens) };
}

const port = Bun.env.PORT;
if (!port) throw new Error("No port");

// In-memory cache for repo trees (owner:repo:branch) -> tree string
// 60 second TTL per key
type CacheEntry = { value: string; expires: number };
const TREE_CACHE_TTL_MS = 60_000;
const treeCache = new Map<string, CacheEntry>();

function getCache(key: string): string | null {
  const entry = treeCache.get(key);
  if (!entry) return null;
  if (Date.now() > entry.expires) {
    treeCache.delete(key);
    return null;
  }
  return entry.value;
}

function setCache(key: string, value: string) {
  treeCache.set(key, { value, expires: Date.now() + TREE_CACHE_TTL_MS });
}

const app = new Elysia()
  // Rate limit hook (runs early)
  .onRequest(({ request, set }) => {
    const ipHeader =
      request.headers.get("x-forwarded-for") ||
      request.headers.get("x-real-ip") ||
      "unknown";
    const ip = ipHeader.split(",")[0].trim();
    const { allowed, remaining } = takeToken(ip);
    // Set informative headers (not standardized but useful)
    set.headers["X-RateLimit-Limit"] = `${RATE_CAPACITY}`;
    set.headers["X-RateLimit-Remaining"] = `${remaining}`;
    // Rough reset time (seconds until full) for client insight
    const bucket = buckets.get(ip)!;
    const secondsUntilFull = (RATE_CAPACITY - bucket.tokens) / REFILL_RATE;
    set.headers["X-RateLimit-Reset"] = `${Math.ceil(secondsUntilFull)}`;
    if (!allowed) {
      set.status = 429;
      return "Too many requests, we are detecting abuse.";
    }
  })
  // Root explanation route
  .get("/", () => {
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
    return explanation;
  })
  // GET /:owner/:repo/:branch?  -> build tree
  .get("/:owner/:repo/:branch?", async ({ params, set }) => {
    try {
      const { owner, repo } = params as { owner: string; repo: string };
      let branch = (params as { branch?: string }).branch;

      if (!owner || !repo) {
        set.status = 400;
        return "owner and repo are required";
      }

      if (!branch) {
        branch = await fetchDefaultBranch(owner, repo);
      }

      const cacheKey = `${owner}:${repo}:${branch}`;
      const cached = getCache(cacheKey);
      if (cached) {
        set.headers["X-Cache"] = "HIT";
        set.headers["Cache-Control"] =
          "s-maxage=600, stale-while-revalidate=60";
        return cached;
      }

      const { tree } = await fetchRepoTree(owner, repo, branch!);
      const treeString = buildTree(tree, owner, repo, branch!);
      setCache(cacheKey, treeString);
      set.headers["X-Cache"] = "MISS";

      // Set caching headers (similar to Hono / Vercel Edge example)
      set.headers["Cache-Control"] = "s-maxage=600, stale-while-revalidate=60";
      return treeString;
    } catch (err: any) {
      set.status = 500;
      return `Error: ${err?.message || "unknown"}`;
    }
  })
  .listen(port);

console.log(
  `🦊 Elysia is running at ${app.server?.hostname}:${app.server?.port}`
);
