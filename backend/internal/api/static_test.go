package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Shahribonu/budget-planner/backend/internal/api"
)

func TestStaticApp(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>app</html>"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "assets", "index-abc.js"), []byte("js"), 0o644)
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken, StaticDir: dir}, nil))
	defer srv.Close()

	get := func(path string) *http.Response {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	res := get("/")
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || string(body) != "<html>app</html>" || res.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("index = %d %q %q", res.StatusCode, body, res.Header.Get("Cache-Control"))
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff")
	}
	if res := get("/assets/index-abc.js"); res.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Errorf("asset cache = %q", res.Header.Get("Cache-Control"))
	}
	// The API stays behind auth, not shadowed by the file server.
	if res := get("/api/me"); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("/api/me = %d", res.StatusCode)
	}
}
