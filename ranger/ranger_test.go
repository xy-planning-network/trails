package ranger_test

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails/http/template"
	tt "github.com/xy-planning-network/trails/http/template/templatetest"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/ranger"
	"golang.org/x/exp/slog"
)

func TestMaintModeHandler(t *testing.T) {
	// Arrange
	b := new(bytes.Buffer)
	l := logger.New(slog.New(slog.HandlerOptions{AddSource: true}.NewTextHandler(b)))
	p := template.NewParser([]fs.FS{tt.NewMockFS(tt.NewMockFile("", nil))})
	handler := ranger.MaintModeHandler(p, l, "test@example.com")
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	// Act + Assert
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
	require.Equal(t, "600", rr.Result().Header.Get("Retry-After"))
	require.Equal(t, "", rr.Body.String())

	// Arrange -- Test POST w/ route & tmpl content
	msg := "Sorry for the inconvenience"
	p = template.NewParser([]fs.FS{tt.NewMockFS(tt.NewMockFile("tmpl/maintenance.tmpl", []byte(msg)))})
	handler = ranger.MaintModeHandler(p, l, "test@example.com")
	req, err = http.NewRequest("POST", "/maint-mode-test", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Act + Assert
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
	require.Equal(t, "600", rr.Result().Header.Get("Retry-After"))
	require.Equal(t, msg, rr.Body.String())
}
