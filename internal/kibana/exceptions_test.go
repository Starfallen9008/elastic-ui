package kibana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFindExceptionItemsReadsAllPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if r.URL.Query().Get("list_id") != "endpoint_trusted_apps" {
			t.Error("missing list scope")
		}
		count := 200
		if page == "2" {
			count = 1
		}
		rows := make([]ExceptionListItem, count)
		for i := range rows {
			rows[i].ID = fmt.Sprintf("%s-%d", page, i)
		}
		_ = json.NewEncoder(w).Encode(exceptionItemResponse{Data: rows, Total: 201})
	}))
	defer server.Close()
	client := &Client{baseURL: server.URL, http: server.Client()}
	items, err := client.FindExceptionItems(context.Background(), "endpoint_trusted_apps", "")
	if err != nil || len(items) != 201 {
		t.Fatalf("got %d items, %v", len(items), err)
	}
}

func TestFindExceptionItemsRejectsIncompletePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(exceptionItemResponse{Data: []ExceptionListItem{}, Total: 1})
	}))
	defer server.Close()
	client := &Client{baseURL: server.URL, http: server.Client()}
	if _, err := client.FindExceptionItems(context.Background(), "endpoint_trusted_apps", ""); err == nil {
		t.Fatal("expected incomplete-page error")
	}
}
