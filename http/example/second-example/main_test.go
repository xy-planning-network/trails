package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/xy-planning-network/trails/ranger"
)

func Test(t *testing.T) {
	go main()

	actual, err := http.Get("http://" + ranger.DefaultHost + ranger.DefaultPort)
	if err != nil {
		t.Fatal(err)
	}

	if actual.StatusCode != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, actual.StatusCode)
	}

	actual, err = http.Get("http://" + ranger.DefaultHost + ranger.DefaultPort + "/broken")
	if err != nil {
		t.Fatal(err)
	}

	if actual.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, actual.StatusCode)
	}

	b, _ := io.ReadAll(actual.Body)
	if !bytes.Contains(b, []byte("replace the default error template")) {
		t.Errorf("expected to receive rendered tmpl/error.tmpl, but got trails/http/template/tmpl/error.tmpl")
	}

	actual, err = http.Get("http://" + ranger.DefaultHost + ranger.DefaultPort + "/shutdown")
	if err != nil {
		t.Fatal(err)
	}

	if actual.StatusCode != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, actual.StatusCode)
	}

	actual, err = http.Get("http://" + ranger.DefaultHost + ranger.DefaultPort)
	if err == nil || !strings.Contains(err.Error(), "connect: connection refused") {
		t.Errorf(`expected "connect: connection refused" in error, got %q`, err.Error())
	}
}
