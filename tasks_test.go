package twelvelabs

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestTasksService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			t.Errorf("Expected to request /tasks, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		fmt.Fprint(w, `[{"_id": "task1", "status": "pending", "video_id": "video1", "index_id": "index1", "created_at": "2024-01-01T00:00:00Z"}]`)
	}))
	defer server.Close()

	client, _ := NewClient(&ClientOptions{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	tasks, err := client.Tasks.List()
	if err != nil {
		t.Fatalf("Error listing tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "task1" {
		t.Errorf("Expected task ID task1, got %s", tasks[0].ID)
	}
}

func TestTasksService_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			t.Errorf("Expected to request /tasks, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		// In a real test, you'd parse multipart form data here
		fmt.Fprint(w, `{"_id": "new_task", "status": "pending", "video_id": "", "index_id": "index1", "created_at": "2024-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	client, _ := NewClient(&ClientOptions{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	// Create a dummy file for testing
	dummyFile, err := os.CreateTemp("", "test_video_*.mp4")
	if err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}
	defer os.Remove(dummyFile.Name())
	defer dummyFile.Close()

	_, err = dummyFile.WriteString("dummy video content")
	if err != nil {
		t.Fatalf("Failed to write to dummy file: %v", err)
	}

	createReq := &TasksCreateRequest{
		IndexID:   "index1",
		VideoFile: dummyFile.Name(),
	}

	task, err := client.Tasks.Create(createReq)
	if err != nil {
		t.Fatalf("Error creating task: %v", err)
	}

	if task.ID != "new_task" {
		t.Errorf("Expected task ID new_task, got %s", task.ID)
	}
}

func TestTasksService_Retrieve(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/task123" {
			t.Errorf("Expected to request /tasks/task123, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		fmt.Fprint(w, `{"_id": "task123", "status": "ready", "video_id": "video123", "index_id": "index1", "created_at": "2024-01-01T00:00:00Z"}`)
	}))
	defer server.Close()

	client, _ := NewClient(&ClientOptions{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	task, err := client.Tasks.Retrieve("task123")
	if err != nil {
		t.Fatalf("Error retrieving task: %v", err)
	}

	if task.ID != "task123" {
		t.Errorf("Expected task ID task123, got %s", task.ID)
	}
	if task.Status != "ready" {
		t.Errorf("Expected task status ready, got %s", task.Status)
	}
}

func TestTasksService_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/task_to_delete" {
			t.Errorf("Expected to request /tasks/task_to_delete, got %s", r.URL.Path)
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

	err := client.Tasks.Delete("task_to_delete")
	if err != nil {
		t.Fatalf("Error deleting task: %v", err)
	}
}

func TestTasksService_WaitForDone(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			fmt.Fprint(w, `{"_id": "wait_task", "status": "pending", "video_id": "", "index_id": "index1", "created_at": "2024-01-01T00:00:00Z"}`)
		} else {
			fmt.Fprint(w, `{"_id": "wait_task", "status": "ready", "video_id": "video_done", "index_id": "index1", "created_at": "2024-01-01T00:00:00Z"}`)
		}
	}))
	defer server.Close()

	client, _ := NewClient(&ClientOptions{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	var receivedStatus string
	completedTask, err := client.Tasks.WaitForDone("wait_task", 1*time.Millisecond, func(t *Task) {
		receivedStatus = t.Status
	})

	if err != nil {
		t.Fatalf("Error waiting for task: %v", err)
	}

	if completedTask.Status != "ready" {
		t.Errorf("Expected task status ready, got %s", completedTask.Status)
	}
	if receivedStatus != "ready" {
		t.Errorf("Expected callback to receive status ready, got %s", receivedStatus)
	}
}
