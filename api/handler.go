package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
)

// RepoDetails holds the default branch from the GitHub API response
type RepoDetails struct {
	DefaultBranch string `json:"default_branch"`
}

type TreeNode struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	Size int    `json:"size"`
	Sha  string `json:"sha"`
	Url  string `json:"url"`
}

type ApiResponse struct {
	SHA  string     `json:"sha"`
	Url  string     `json:"url"`
	Tree []TreeNode `json:"tree"`
}

// fetchDefaultBranch gets the default branch for a repository
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

// fetchTree gets the tree for a specific branch
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

func getChildren(parent string, tree []TreeNode) []TreeNode {
	var children []TreeNode
	for _, node := range tree {
		dir := ""
		if strings.Contains(node.Path, "/") {
			dir = node.Path[:strings.LastIndex(node.Path, "/")]
		}
		if dir == parent {
			children = append(children, node)
		}
	}
	return children
}

func buildTree(dir string, prefix string, tree []TreeNode) string {
	children := getChildren(dir, tree)
	if len(children) == 0 {
		return ""
	}

	sort.Slice(children, func(i, j int) bool {
		return path.Base(children[i].Path) < path.Base(children[j].Path)
	})

	var output strings.Builder
	for i, child := range children {
		isLast := i == len(children)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		name := path.Base(child.Path)
		if child.Type == "tree" {
			name += "/"
		}

		output.WriteString(prefix + connector + name + "\n")

		if child.Type == "tree" {
			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			output.WriteString(buildTree(child.Path, newPrefix, tree))
		}
	}
	return output.String()
}

func parseTree(tree []TreeNode) string {
	return buildTree("", "", tree)
}

// Handler processes the HTTP request
func Handler(w http.ResponseWriter, r *http.Request) {
	// Remove leading "/" from the path
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	// Validate the path: must be owner/repo or owner/repo/branch
	if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "Invalid path. Use /owner/repo or /owner/repo/branch", http.StatusBadRequest)
		return
	}

	owner := parts[0]
	repo := parts[1]
	branch := ""

	// If branch isn’t provided, fetch the default
	if len(parts) == 3 {
		branch = parts[2]
	} else {
		var err error
		branch, err = fetchDefaultBranch(owner, repo)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch default branch: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Fetch the tree using the determined branch
	fetchedTree, err := fetchTree(owner, repo, branch)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch tree: %v", err), http.StatusInternalServerError)
		return
	}

	// Assuming parseTree exists and generates the tree output from fetchedTree.Tree
	treeOutput := parseTree(fetchedTree.Tree)
	repoName := fmt.Sprintf("%s/%s (%s)", owner, repo, branch)
	fullOutput := repoName + "\n" + treeOutput

	// Send the response as plain text
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, fullOutput)
}
