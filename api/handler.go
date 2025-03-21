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

func fetchTree(owner, repo string) (ApiResponse, error) {
	URL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/main?recursive=true", owner, repo)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return ApiResponse{}, fmt.Errorf("GITHUB_TOKEN not set")
	}

	// prepare
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	// set token
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}

	// make request
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
		return ApiResponse{}, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// handle status code
	if resp.StatusCode != http.StatusOK {
		return ApiResponse{}, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	// read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("error reading response: %w", err)
	}

	// parse json
	var data ApiResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return ApiResponse{}, fmt.Errorf("error parsing JSON: %w", err)
	}

	return data, nil
}

func parseTree(tree []TreeNode) string {
	// Step 1: Build a map of directory paths to their direct children
	dirMap := make(map[string][]TreeNode)
	for _, node := range tree {
		parent := ""
		if idx := strings.LastIndex(node.Path, "/"); idx != -1 {
			parent = node.Path[:idx] // Extract parent directory
		}
		dirMap[parent] = append(dirMap[parent], node)
	}

	// Step 2: Generate the tree string starting from the root
	return buildTree("", "", dirMap)
}

func buildTree(dir string, prefix string, dirMap map[string][]TreeNode) string {
	// Get children of the current directory
	children, ok := dirMap[dir]
	if !ok {
		return "" // No children, return empty string
	}

	// Step 3: Sort children by base name for alphabetical order
	sort.Slice(children, func(i, j int) bool {
		return path.Base(children[i].Path) < path.Base(children[j].Path)
	})

	// Step 4: Build the tree string
	var result strings.Builder
	for i, child := range children {
		isLast := i == len(children)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		// Get the base name and append "/" for directories
		name := path.Base(child.Path)
		if child.Type == "tree" {
			name += "/"
		}

		// Write the current node's line
		result.WriteString(prefix + connector + name + "\n")

		// If it's a directory, recurse into it with updated prefix
		if child.Type == "tree" {
			childPrefix := prefix
			if isLast {
				childPrefix += "    " // No vertical line if last
			} else {
				childPrefix += "│   " // Continue vertical line
			}
			result.WriteString(buildTree(child.Path, childPrefix, dirMap))
		}
	}

	return result.String()
}

func Handler(w http.ResponseWriter, r *http.Request) {

	// removing first "/"
	path := strings.TrimPrefix(r.URL.Path, "/")

	// split by "/"
	parts := strings.Split(path, "/")

	// pattern check (owner/repo)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {

		// extract
		owner := parts[0]
		repo := parts[1]

		fetchedTree, err := fetchTree(owner, repo)
		if err != nil {

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		treeOutput := parseTree(fetchedTree.Tree)
		repoName := owner + "/" + repo
		fullOutput := repoName + "\n" + treeOutput

		// Send response as plain text
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, fullOutput)
	} else {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}
