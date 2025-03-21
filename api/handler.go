package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// Redis client
var rdb *redis.Client

// TTL for cache entries
var ttl time.Duration

// Initialize Redis client and TTL from environment variables
func init() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisDBStr := os.Getenv("REDIS_DB")
	redisDB := 0
	if redisDBStr != "" {
		var err error
		redisDB, err = strconv.Atoi(redisDBStr)
		if err != nil {
			panic("Invalid REDIS_DB: " + err.Error())
		}
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}

	ttlStr := os.Getenv("CACHE_TTL")
	if ttlStr == "" {
		ttl = 10 * time.Minute
	} else {
		ttlMin, err := strconv.Atoi(ttlStr)
		if err != nil {
			panic("Invalid CACHE_TTL: " + err.Error())
		}
		ttl = time.Duration(ttlMin) * time.Minute
	}
}

// RepoDetails holds the default branch from the GitHub API
type RepoDetails struct {
	DefaultBranch string `json:"default_branch"`
}

// ApiResponse holds the tree data from the GitHub API
type ApiResponse struct {
	Tree []TreeNode `json:"tree"`
}

// TreeNode represents a node in the repository tree
type TreeNode struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// fetchDefaultBranch fetches the default branch from GitHub API
func fetchDefaultBranch(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN not set")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	var repoDetails RepoDetails
	if err := json.Unmarshal(body, &repoDetails); err != nil {
		return "", fmt.Errorf("error parsing JSON: %w", err)
	}

	return repoDetails.DefaultBranch, nil
}

// fetchTree fetches the tree for a specific branch from GitHub API
func fetchTree(owner, repo, branch string) (ApiResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=true", owner, repo, branch)
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return ApiResponse{}, fmt.Errorf("GITHUB_TOKEN not set")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ApiResponse{}, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("error reading response: %w", err)
	}

	var data ApiResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return ApiResponse{}, fmt.Errorf("error parsing JSON: %w", err)
	}

	return data, nil
}

// parseTree builds a tree-like string from the API response
func parseTree(tree []TreeNode) string {
	var lines []string
	for _, node := range tree {
		lines = append(lines, node.Path)
	}
	return strings.Join(lines, "\n")
}

// getDefaultBranch gets the default branch from cache or API
func getDefaultBranch(ctx context.Context, owner, repo string, noCache bool) (string, error) {
	key := fmt.Sprintf("default_branch:%s:%s", owner, repo)
	if !noCache {
		val, err := rdb.Get(ctx, key).Result()
		if err == nil {
			return val, nil
		} else if err != redis.Nil {
			return "", fmt.Errorf("redis error: %w", err)
		}
	}

	branch, err := fetchDefaultBranch(owner, repo)
	if err != nil {
		return "", err
	}

	if err := rdb.Set(ctx, key, branch, ttl).Err(); err != nil {
		return "", fmt.Errorf("redis set error: %w", err)
	}
	return branch, nil
}

// getTreeString gets the built tree string from cache or API
func getTreeString(ctx context.Context, owner, repo, branch string, noCache bool) (string, error) {
	key := fmt.Sprintf("tree:%s:%s:%s", owner, repo, branch)
	if !noCache {
		val, err := rdb.Get(ctx, key).Result()
		if err == nil {
			return val, nil
		} else if err != redis.Nil {
			return "", fmt.Errorf("redis error: %w", err)
		}
	}

	fetchedTree, err := fetchTree(owner, repo, branch)
	if err != nil {
		return "", err
	}

	treeOutput := parseTree(fetchedTree.Tree)
	if err := rdb.Set(ctx, key, treeOutput, ttl).Err(); err != nil {
		return "", fmt.Errorf("redis set error: %w", err)
	}
	return treeOutput, nil
}

// Handler processes the HTTP request
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "Invalid path. Use /owner/repo or /owner/repo/branch", http.StatusBadRequest)
		return
	}

	// Check for nocache query parameter
	noCache := strings.ToLower(r.URL.Query().Get("nocache")) == "true"

	owner := parts[0]
	repo := parts[1]
	var branch string
	var err error

	if len(parts) == 2 {
		branch, err = getDefaultBranch(ctx, owner, repo, noCache)
	} else {
		branch = parts[2]
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get default branch: %v", err), http.StatusInternalServerError)
		return
	}

	treeOutput, err := getTreeString(ctx, owner, repo, branch, noCache)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get tree: %v", err), http.StatusInternalServerError)
		return
	}

	repoName := fmt.Sprintf("%s/%s (%s)", owner, repo, branch)
	fullOutput := repoName + "\n" + treeOutput

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, fullOutput)
}
