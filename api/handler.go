package handler

import (
	"fmt"
	"net/http"
	"strings"
)

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

		fmt.Fprintf(w, "Owner: %s, Repo: %s", owner, repo)
	} else {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}
