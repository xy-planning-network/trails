package main

import (
	"net/http"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	go main()

	actual, err := http.Get("http://localhost:8080/")
	if err != nil {
		t.Fatal(err)
	}

	if actual.StatusCode != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, actual.StatusCode)
	}

	actual, err = http.Get("http://localhost:8080/shutdown")
	if err != nil {
		t.Fatal(err)
	}

	if actual.StatusCode != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, actual.StatusCode)
	}

	actual, err = http.Get("http://localhost:8080/")
	if err == nil || !strings.Contains(err.Error(), "connect: connection refused") {
		t.Errorf(`expected "connect: connection refused" in error, got %q`, err.Error())
	}
}
