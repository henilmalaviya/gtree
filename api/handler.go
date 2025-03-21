package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type ApiResponse struct {
	SHA  string `json:"sha"`
	Url  string `json:"url"`
	Tree []struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		Size int    `json:"size"`
		Sha  string `json:"sha"`
		Url  string `json:"url"`
	} `json:"tree"`
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

		// send response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fetchedTree)

	} else {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}
