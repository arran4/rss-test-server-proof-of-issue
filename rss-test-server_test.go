package main

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIndexHandler(t *testing.T) {
	s := &Server{}
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.index)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expectedContent := "RSS Client Test Feed"
	if !strings.Contains(rr.Body.String(), expectedContent) {
		t.Errorf("handler returned unexpected body: got %v want it to contain %v",
			rr.Body.String(), expectedContent)
	}
}

func TestFeedHandler(t *testing.T) {
	s := &Server{}
	req, err := http.NewRequest("GET", "/feed.xml", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Important for baseURL to resolve properly without panicking
	req.Host = "localhost:8080"

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.feed)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/rss+xml") {
		t.Errorf("handler returned unexpected content type: got %v want application/rss+xml", contentType)
	}

	var rss RSS
	if err := xml.Unmarshal(rr.Body.Bytes(), &rss); err != nil {
		t.Fatalf("could not parse rss xml: %v", err)
	}

	if len(rss.Channel.Items) != 15 {
		t.Errorf("expected 15 items, got %d", len(rss.Channel.Items))
	}
}

func TestItemHandler(t *testing.T) {
	s := &Server{}
	req, err := http.NewRequest("GET", "/item/1760000000?offset=0", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "localhost:8080"

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.item)

	// Since item handler sleeps randomly, we might just want to let it do its thing
	// The sleep min is 400ms, max 4s, which is fine for tests unless it causes timeout
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expectedPrefix := "RSS client test item"
	if !strings.Contains(rr.Body.String(), expectedPrefix) {
		t.Errorf("handler returned unexpected body: got %v want it to contain %v",
			rr.Body.String(), expectedPrefix)
	}
	if !strings.Contains(rr.Body.String(), "unix_time: 1760000000") {
		t.Errorf("handler missing correct unix_time")
	}
}

func TestItemHandlerInvalidTimestamp(t *testing.T) {
	s := &Server{}
	req, err := http.NewRequest("GET", "/item/invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.item)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestRandomDelay(t *testing.T) {
	s := &Server{}
	min := 10 * time.Millisecond
	max := 50 * time.Millisecond
	delay := s.randomDelay(min, max)

	if delay < min || delay > max {
		t.Errorf("randomDelay(%v, %v) returned %v, which is out of bounds", min, max, delay)
	}

	// Test edge case min == max
	delay = s.randomDelay(max, max)
	if delay != max {
		t.Errorf("randomDelay(%v, %v) returned %v, expected %v", max, max, delay, max)
	}

	// Test edge case min > max
	delay = s.randomDelay(max, min)
	if delay != max {
		t.Errorf("randomDelay(%v, %v) returned %v, expected %v", max, min, delay, max)
	}
}
