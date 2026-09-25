package container

import (
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
)

// SyncURLFromParams rebuilds the URL query string from query params.
func SyncURLFromParams(url string, queryParams []domain.KeyValue) string {
	if len(queryParams) == 0 {
		if strings.Contains(url, "?") {
			return strings.Split(url, "?")[0]
		}
		return url
	}
	if url == "" {
		return ""
	}
	base := url
	if idx := strings.Index(url, "?"); idx >= 0 {
		base = url[:idx]
	}
	encoded := domain.EncodeQueryParams(queryParams)
	if encoded == "" {
		return base
	}
	return base + "?" + encoded
}

// SyncParamsFromURL parses query and path params from a URL, preserving existing path values.
func SyncParamsFromURL(url string, existingPath []domain.KeyValue) (query []domain.KeyValue, path []domain.KeyValue) {
	if url == "" {
		return nil, nil
	}
	parts := strings.SplitN(url, "?", 2)
	if len(parts) == 2 {
		query = domain.ParseQueryParams(parts[1])
	}
	path = domain.ParsePathParams(parts[0])
	for _, param := range existingPath {
		for i, newParam := range path {
			if newParam.Key == param.Key {
				path[i].Value = param.Value
				path[i].Enable = param.Enable
				if param.ID != "" {
					path[i].ID = param.ID
				}
			}
		}
	}
	return query, path
}
