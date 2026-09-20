package imgbed

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUploadJoinsHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("uploadChannel") != "telegram" {
			t.Errorf("channel %s", r.URL.Query().Get("uploadChannel"))
		}
		_, _ = w.Write([]byte(`[{"src":"/file/abc.jpg"}]`))
	}))
	defer srv.Close()
	got, err := (&Client{BaseURL: srv.URL, HTTP: srv.Client()}).Upload(t.Context(), "a.jpg", []byte("jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if got != srv.URL+"/file/abc.jpg" {
		t.Fatal(got)
	}
}

func TestUploadRejectsForeignHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"src":"https://evil.example/x.jpg"}]`))
	}))
	defer srv.Close()
	_, err := (&Client{BaseURL: srv.URL, HTTP: srv.Client()}).Upload(t.Context(), "a.jpg", []byte("jpg"))
	if err == nil {
		t.Fatal("expected foreign host rejection")
	}
}

func TestUploadRequiresBaseURL(t *testing.T) {
	_, err := (&Client{}).Upload(t.Context(), "a.jpg", []byte("jpg"))
	if err == nil {
		t.Fatal("expected missing base url")
	}
}
