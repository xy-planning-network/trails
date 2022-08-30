package main

import (
	"net/http"
	"os"
	"testing"

	"github.com/xy-planning-network/trails/ranger"
)

func Test(t *testing.T) {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	go main()

	for _, tc := range []struct {
		name     string
		input    string
		expected int
	}{
		{"root", "/", http.StatusOK},
		{"not-found", "/not-found", http.StatusNotFound},
		{"broken-500", "/broken", http.StatusInternalServerError},
		{"incorrect-200", "/incorrect", http.StatusOK},
		{"authed-200", "/authed", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := http.Get("http://" + ranger.DefaultHost + ranger.DefaultPort + tc.input)
			if err != nil {
				t.Fatal(err)
			}

			if actual.StatusCode != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, actual.StatusCode)
			}
		})
	}

	if err := p.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
}
