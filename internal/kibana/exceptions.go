package kibana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type ExceptionEntry struct {
	Type     string `json:"type"`
	Field    string `json:"field"`
	Value    any    `json:"value"`
	Operator string `json:"operator"`
}

type ExceptionListItem struct {
	ID            string           `json:"id"`
	ItemID        string           `json:"item_id"`
	ListID        string           `json:"list_id"`
	Type          string           `json:"type"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	Entries       []ExceptionEntry `json:"entries"`
	NamespaceType string           `json:"namespace_type"`
	OSTypes       []string         `json:"os_types"`
	Tags          []string         `json:"tags"`
	CreatedAt     string           `json:"created_at"`
	CreatedBy     string           `json:"created_by"`
	UpdatedAt     string           `json:"updated_at"`
	UpdatedBy     string           `json:"updated_by"`
}

type ExceptionList struct {
	ID          string   `json:"id"`
	ListID      string   `json:"list_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	OS          []string `json:"os_types"`
	Tags        []string `json:"tags"`
	Immutable   bool     `json:"immutable"`
}

type exceptionListResponse struct {
	Data    []ExceptionList `json:"data"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	PerPage int             `json:"per_page"`
}

type exceptionItemResponse struct {
	Data    []ExceptionListItem `json:"data"`
	Total   int                 `json:"total"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
}

type exceptionItemSingle struct {
	Data ExceptionListItem `json:"data"`
}

func (c *Client) FindExceptionLists(ctx context.Context, listType string) ([]ExceptionList, error) {
	path := "/api/exception_lists/_find?page=1&per_page=100"
	if listType != "" {
		path += "&list_type=" + url.QueryEscape(listType)
	}
	var out exceptionListResponse
	if err := c.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) FindExceptionItems(ctx context.Context, listID string, listType string) ([]ExceptionListItem, error) {
	path := "/api/exception_lists/items/_find?per_page=200&namespace_type=agnostic"
	if listID != "" {
		path += "&list_id=" + url.QueryEscape(listID)
	}
	if listType != "" {
		path += "&list_type=" + url.QueryEscape(listType)
	}
	items := []ExceptionListItem{}
	for page := 1; ; page++ {
		var out exceptionItemResponse
		if err := c.Get(ctx, fmt.Sprintf("%s&page=%d", path, page), &out); err != nil {
			return nil, err
		}
		items = append(items, out.Data...)
		if len(items) >= out.Total {
			return items, nil
		}
		if len(out.Data) == 0 {
			return nil, fmt.Errorf("排除列表分页不完整，已读取 %d / %d 条", len(items), out.Total)
		}
	}
}

func (c *Client) FindEndpointListItems(ctx context.Context) ([]ExceptionListItem, error) {
	var out exceptionItemResponse
	path := "/api/endpoint_list/items/_find?page=1&per_page=200"
	if err := c.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) CreateExceptionItem(ctx context.Context, listID, name, description string, entries []ExceptionEntry, osTypes, tags []string) (*ExceptionListItem, error) {
	body := map[string]any{
		"list_id":        listID,
		"type":           "simple",
		"name":           name,
		"description":    description,
		"entries":        entries,
		"os_types":       osTypes,
		"tags":           tags,
		"namespace_type": "agnostic",
	}
	var raw json.RawMessage
	if err := c.Post(ctx, "/api/exception_lists/items", body, &raw); err != nil {
		return nil, err
	}
	var single exceptionItemSingle
	if json.Unmarshal(raw, &single) == nil && single.Data.ID != "" {
		return &single.Data, nil
	}
	var item ExceptionListItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *Client) CreateEndpointException(ctx context.Context, name, description string, entries []ExceptionEntry, osTypes, tags []string) (*ExceptionListItem, error) {
	body := map[string]any{
		"type":           "simple",
		"name":           name,
		"description":    description,
		"entries":        entries,
		"os_types":       osTypes,
		"tags":           tags,
		"namespace_type": "agnostic",
	}
	var out exceptionItemSingle
	if err := c.Post(ctx, "/api/endpoint_list/items", body, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *Client) DeleteExceptionItem(ctx context.Context, item ExceptionListItem) error {
	path := fmt.Sprintf("/api/exception_lists/items?id=%s&namespace_type=agnostic", url.QueryEscape(item.ID))
	return c.Delete(ctx, path, nil)
}

func (c *Client) DeleteEndpointListItem(ctx context.Context, itemID string) error {
	path := "/api/endpoint_list/items?item_id=" + url.QueryEscape(itemID) + "&namespace_type=agnostic"
	return c.Delete(ctx, path, nil)
}

func PolicyTag(policyID string) string {
	if strings.TrimSpace(policyID) == "" {
		return "policy:all"
	}
	return "policy:" + policyID
}
