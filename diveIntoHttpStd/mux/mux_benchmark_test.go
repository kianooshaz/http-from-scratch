package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// dummyHandler is a simple HTTP handler
func dummyHandler(w http.ResponseWriter, r *http.Request) {}

// -------------------- Setup functions --------------------

// setupFixed registers only fixed paths
func setupFixed() *http.ServeMux {
	mux := http.NewServeMux()
	for i := 1; i <= 30; i++ {
		path := "/fixed/path" + strconv.Itoa(i)
		mux.HandleFunc(path, dummyHandler)
	}
	return mux
}

// setupWildcard registers only "wildcard-like" paths
func setupWildcard() *http.ServeMux {
	mux := http.NewServeMux()
	for i := 1; i <= 30; i++ {
		path := "/users/" + strconv.Itoa(i) + "/profile"
		mux.HandleFunc(path, dummyHandler)
	}
	return mux
}

// setupMixed registers both fixed and wildcard-like paths
func setupMixed() *http.ServeMux {
	mux := http.NewServeMux()
	for i := 1; i <= 15; i++ {
		path := "/fixed/path" + strconv.Itoa(i)
		mux.HandleFunc(path, dummyHandler)
	}
	// wildcard-like paths
	for i := 1; i <= 15; i++ {
		path := "/users/" + strconv.Itoa(i) + "/profile"
		mux.HandleFunc(path, dummyHandler)
	}
	return mux
}

// -------------------- Benchmarks --------------------

// BenchmarkFixedOnly tests fixed paths from setupFixed
func BenchmarkFixedOnly(b *testing.B) {
	mux := setupFixed()
	req := httptest.NewRequest("GET", "/fixed/path1", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

// BenchmarkWildcardOnly tests wildcard-like paths from setupWildcard
func BenchmarkWildcardOnly(b *testing.B) {
	mux := setupWildcard()
	req := httptest.NewRequest("GET", "/users/7/profile", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

// BenchmarkMixedFixed tests a fixed path from setupMixed
func BenchmarkMixedFixed(b *testing.B) {
	mux := setupMixed()
	req := httptest.NewRequest("GET", "/fixed/path1", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

// BenchmarkMixedWildcard tests a wildcard-like path from setupMixed
func BenchmarkMixedWildcard(b *testing.B) {
	mux := setupMixed()
	req := httptest.NewRequest("GET", "/users/7/profile", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}
