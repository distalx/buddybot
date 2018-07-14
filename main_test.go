package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServePIS(t *testing.T) {
	var reqStr = `{
		"token": "Jhj5dZrVaK7ZwHHjRyZWjbDl",
		"challenge": "3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P",
		"type": "url_verification"
  }`

	Routes()

	r, err := http.NewRequest(http.MethodPost, "/events-endpoint", bytes.NewBufferString(reqStr))
	if err != nil {
		t.Fatal("should be able to create request without error, got:", err)
	}

	w := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Error("should return an HTTP 200 response, got:", w.Code)
	}
}
