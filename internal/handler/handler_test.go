package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/slymn08183/insider-assessment/internal/cache"
	"github.com/slymn08183/insider-assessment/internal/model"
	"github.com/slymn08183/insider-assessment/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupTestRouter — Sets up router with real Redis and PG for testing.
// Requires PG and Redis running in Docker.
func setupTestRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// PostgreSQL
	db, err := gorm.Open(postgres.Open("postgres://postgres:postgres@localhost:5432/insider?sslmode=disable"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	db.AutoMigrate(&model.Event{})

	// Clean events table before each test
	db.Exec("DELETE FROM events")

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	rdb.FlushDB(t.Context())

	queue := cache.NewEventQueue(rdb)
	repo := repository.NewEventRepository(db)

	r := gin.New()

	eventHandler := NewEventHandler(queue)
	metricsHandler := NewMetricsHandler(repo)

	eventHandler.RegisterRoutes(r)
	metricsHandler.RegisterRoutes(r)

	return r
}

func TestCreateEvent(t *testing.T) {
	r := setupTestRouter(t)

	event := model.Event{
		EventName: "product_view",
		UserID:    "user_123",
		Timestamp: 1723475612,
		Channel:   "web",
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "accepted" {
		t.Errorf("expected status 'accepted', got '%v'", resp["status"])
	}
	if resp["event_hash"] == nil || resp["event_hash"] == "" {
		t.Error("expected event_hash in response")
	}
}

func TestCreateEventDuplicate(t *testing.T) {
	r := setupTestRouter(t)

	event := model.Event{
		EventName: "product_view",
		UserID:    "user_123",
		Timestamp: 1723475612,
	}
	body, _ := json.Marshal(event)

	// First request — 202
	req1 := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusAccepted {
		t.Errorf("first request: expected 202, got %d", w1.Code)
	}

	// Same request again — 409
	body2, _ := json.Marshal(event)
	req2 := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("duplicate request: expected 409, got %d, body: %s", w2.Code, w2.Body.String())
	}
}

func TestCreateEventValidation(t *testing.T) {
	r := setupTestRouter(t)

	// Missing event_name
	event := map[string]interface{}{
		"user_id":   "user_123",
		"timestamp": 1723475612,
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing event_name: expected 400, got %d", w.Code)
	}

	// Timestamp too old
	event2 := map[string]interface{}{
		"event_name": "test",
		"user_id":    "user_123",
		"timestamp":  1000000000, // 2001
	}
	body2, _ := json.Marshal(event2)

	req2 := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("old timestamp: expected 400, got %d, body: %s", w2.Code, w2.Body.String())
	}
}

func TestBulkCreate(t *testing.T) {
	r := setupTestRouter(t)

	events := []model.Event{
		{EventName: "view", UserID: "user_1", Timestamp: 1723475612},
		{EventName: "click", UserID: "user_2", Timestamp: 1723475613},
		{EventName: "view", UserID: "user_1", Timestamp: 1723475612}, // duplicate
	}
	body, _ := json.Marshal(events)

	req := httptest.NewRequest(http.MethodPost, "/events/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// 2 accepted, 1 duplicate
	if resp["accepted"] != float64(2) {
		t.Errorf("expected 2 accepted, got %v", resp["accepted"])
	}
	if resp["duplicates"] != float64(1) {
		t.Errorf("expected 1 duplicate, got %v", resp["duplicates"])
	}
}
