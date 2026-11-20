package kibana

import (
	"context"
	"fmt"
	"net/url"
)

type Agent struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	PolicyID          string `json:"policy_id"`
	PolicyRevision    int    `json:"policy_revision"`
	PolicyRevisionIdx int    `json:"policy_revision_idx"`
	LastCheckin       string `json:"last_checkin"`
	LocalMetadata     struct {
		Host struct {
			Name string `json:"name"`
			OS   struct {
				Name     string `json:"name"`
				Platform string `json:"platform"`
				Full     string `json:"full"`
			} `json:"os"`
		} `json:"host"`
	} `json:"local_metadata"`
}

type AgentsResponse struct {
	Items []Agent `json:"items"`
	Total int     `json:"total"`
}

func (c *Client) ListAgents(ctx context.Context, perPage int) (*AgentsResponse, error) {
	if perPage <= 0 {
		perPage = 200
	}
	var out AgentsResponse
	path := fmt.Sprintf("/api/fleet/agents?perPage=%d", perPage)
	if err := c.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAgent(ctx context.Context, id string) (*Agent, error) {
	var out struct {
		Item Agent `json:"item"`
	}
	path := "/api/fleet/agents/" + url.PathEscape(id)
	if err := c.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out.Item, nil
}
