package httpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/container"
)

// Editing the Query table must reach the URL, which is what gets sent.
func TestQueryTableEditsSyncURL(t *testing.T) {
	setupText(t)
	deps := container.Deps{Report: container.Reporter{Toast: func(string) {}}}
	req := domain.NewHTTPRequest("search")
	req.Spec.HTTP.URL = "https://example.com/search"
	c := Open(req, deps)
	http := c.req.Spec.HTTP

	container.AddKVRow(c.queryParams, c.queryParamsChanged)
	require.Len(t, c.queryParams.Rows, 1)
	row := c.queryParams.Rows[0]
	assert.Equal(t, "https://example.com/search", http.URL)

	row.Cells["key"] = "query"
	c.queryParams.OnCellChange(row.ID, "key", "query")
	row.Cells["value"] = "istanbul"
	c.queryParams.OnCellChange(row.ID, "value", "istanbul")
	assert.Equal(t, "https://example.com/search?query=istanbul", http.URL)
	assert.True(t, c.Dirty())

	c.queryParams.Rows[0].Selected = false
	c.queryParams.OnSelectionChange()
	assert.Equal(t, "https://example.com/search", http.URL)

	c.queryParams.Rows[0].Selected = true
	c.queryParams.OnSelectionChange()
	assert.Equal(t, "https://example.com/search?query=istanbul", http.URL)

	c.queryParams.Actions[0].OnClick(row.ID)
	assert.Empty(t, c.queryParams.Rows)
	assert.Equal(t, "https://example.com/search", http.URL)
}
