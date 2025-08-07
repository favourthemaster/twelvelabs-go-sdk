package twelvelabs

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIndexesService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/indexes" {
			t.Errorf("Expected to request /indexes, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		fmt.Fprint(w, `[{"_id": "index1", "index_name": "test_index", "models": [], "created_at": "2024-01-01T00:00:00Z"}]`)
	}))
	defer server.Close()

	client, _ := NewClient(&ClientOptions{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	indexes, err := client.Indexes.List()
	if err != nil {
		t.Fatalf("Error listing indexes: %v", err)
	}

	if len(indexes) != 1 {
		t.Errorf("Expected 1 index, got %d", len(indexes))
	}
	if indexes[0].ID != "index1" {
		t.Errorf("Expected index ID index1, got %s", indexes[0].ID)
	}
}

func TestIndexesService_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/indexes" {
			t.Errorf("Expected to request /indexes, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		fmt.Fprint(w, `{"_id": "new_index", "index_name": "new_test_index", "models": [], "created_at": "2024-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	client, _ := NewClient(&ClientOptions{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	createReq := &IndexesCreateRequest{
		IndexName: "new_test_index",
		Models:    []Model{},
	}

	index, err := client.Indexes.Create(createReq)
	if err != nil {
		t.Fatalf("Error creating index: %v", err)
	}

	if index.ID != "new_index" {
		t.Errorf("Expected index ID new_index, got %s", index.ID)
	}
}

func TestIndexesService_Retrieve(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/indexes/index123" {
			t.Errorf("Expected to request /indexes/index123, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		fmt.Fprint(w, `{"_id": "index123", "index_name": "retrieved_index", "models": [], "created_at": "2024-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	client, _ := NewClient(&ClientOptions{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	index, err := client.Indexes.Retrieve("index123")
	if err != nil {
		t.Fatalf("Error retrieving index: %v", err)
	}

	if index.ID != "index123" {
		t.Errorf("Expected index ID index123, got %s", index.ID)
	}
}

func TestIndexesService_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/indexes/index_to_delete" {
			t.Errorf("Expected to request /indexes/index_to_delete, got %s", r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _ := NewClient(&ClientOptions{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	err := client.Indexes.Delete("index_to_delete")
	if err != nil {
		t.Fatalf("Error deleting index: %v", err)
	}
}
