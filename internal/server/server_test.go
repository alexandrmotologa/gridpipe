package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	hub := NewSSEHub()
	RegisterRoutes(mux, hub)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res map[string]any
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["status"] != "ok" || res["service"] != "gridpipe" {
		t.Errorf("unexpected health response: %+v", res)
	}
}

func TestUploadAndCleanCSV(t *testing.T) {
	mux := http.NewServeMux()
	hub := NewSSEHub()
	RegisterRoutes(mux, hub)

	// Prepare multipart CSV upload
	csvData := "Item,Date,Amount\nA100,15/01/2025,$120.50\nB200,2025-02-18,$300.00\nTotal,,$420.50\n"
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test_report.csv")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write([]byte(csvData))
	_ = writer.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, uploadReq)
	if w.Code != http.StatusOK {
		t.Fatalf("upload failed with code %d: %s", w.Code, w.Body.String())
	}

	// Now call /api/clean
	cleanReq := httptest.NewRequest(http.MethodPost, "/api/clean", bytes.NewBufferString("{}"))
	wClean := httptest.NewRecorder()
	mux.ServeHTTP(wClean, cleanReq)

	if wClean.Code != http.StatusOK {
		t.Fatalf("clean failed with code %d: %s", wClean.Code, wClean.Body.String())
	}

	var cleanRes map[string]any
	if err := json.NewDecoder(wClean.Body).Decode(&cleanRes); err != nil {
		t.Fatalf("failed to decode clean response: %v", err)
	}

	if cleanRes["status"] != "success" {
		t.Errorf("expected success status: %+v", cleanRes)
	}
}
