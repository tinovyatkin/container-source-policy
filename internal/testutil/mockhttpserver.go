package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
)

// mockFile holds content and optional headers for a file
type mockFile struct {
	content []byte
	headers map[string]string
}

// MockHTTPServer is a test HTTP server that serves files with deterministic content
type MockHTTPServer struct {
	Server   *httptest.Server
	files    map[string]*mockFile // path -> file data
	requests []string             // tracks all requests made to the server
	mu       sync.Mutex
}

// NewMockHTTPServer creates a new mock HTTP server
func NewMockHTTPServer() *MockHTTPServer {
	ms := &MockHTTPServer{
		files: make(map[string]*mockFile),
	}

	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := r.Method + " " + r.URL.Path
		ms.mu.Lock()
		ms.requests = append(ms.requests, req)
		file, ok := ms.files[r.URL.Path]
		ms.mu.Unlock()

		if !ok {
			http.NotFound(w, r)
			return
		}

		// Set custom headers if any
		for k, v := range file.headers {
			w.Header().Set(k, v)
		}

		// For HEAD requests, only return headers
		if r.Method == http.MethodHead {
			// Compute SHA256 and return as ETag (simulating raw.githubusercontent.com behavior)
			hash := sha256.Sum256(file.content)
			etag := hex.EncodeToString(hash[:])
			w.Header().Set("ETag", etag)
			w.Header().Set("Content-Length", strconv.Itoa(len(file.content)))
			w.WriteHeader(http.StatusOK)
			return
		}

		// For GET requests, return the content
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(file.content)
	}))

	return ms
}

// Close shuts down the mock HTTP server
func (ms *MockHTTPServer) Close() {
	ms.Server.Close()
}

// AddFile adds a file with deterministic content to the mock server
// Returns the sha256 checksum of the content
func (ms *MockHTTPServer) AddFile(path, content string) string {
	return ms.AddFileWithHeaders(path, content, nil)
}

// AddFileWithHeaders adds a file with deterministic content and custom HTTP headers
// Returns the sha256 checksum of the content
func (ms *MockHTTPServer) AddFileWithHeaders(path, content string, headers map[string]string) string {
	contentBytes := []byte(content)
	hash := sha256.Sum256(contentBytes)
	checksum := "sha256:" + hex.EncodeToString(hash[:])

	ms.mu.Lock()
	ms.files[path] = &mockFile{
		content: contentBytes,
		headers: headers,
	}
	ms.mu.Unlock()

	return checksum
}

// Requests returns all requests made to the server since last reset
func (ms *MockHTTPServer) Requests() []string {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	result := make([]string, len(ms.requests))
	copy(result, ms.requests)
	return result
}

// ResetRequests clears the tracked requests
func (ms *MockHTTPServer) ResetRequests() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.requests = nil
}

// HasRequest checks if a request matching the pattern was made
func (ms *MockHTTPServer) HasRequest(pattern string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	for _, req := range ms.requests {
		if strings.Contains(req, pattern) {
			return true
		}
	}
	return false
}

// URL returns the base URL of the mock HTTP server
func (ms *MockHTTPServer) URL() string {
	return ms.Server.URL
}
