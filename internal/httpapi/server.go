package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/kiteyuan/nexa-pulse-bot/internal/config"
	"github.com/kiteyuan/nexa-pulse-bot/internal/ports"
)

type Server struct {
	Accounts ports.Accounts
	Feeds    ports.Feeds
	Themes   ports.Themes
	Content  ports.Content
	Cfg      config.Config
	TG       ports.TelegramAdmin
	Public   fs.FS
	Admin    fs.FS
	limit    *limiter
}

func (s *Server) PublicHandler() http.Handler {
	s.ensureLimit()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/themes", s.rate(60, s.listPublicThemes))
	mux.HandleFunc("GET /api/items", s.rate(60, s.listItems))
	mux.HandleFunc("GET /api/items/{id}", s.rate(60, s.getItem))
	mux.HandleFunc("GET /rss.xml", s.rate(30, s.rssFeed))
	if s.Cfg.Dev {
		mux.HandleFunc("/", devPage("http://127.0.0.1:5173"))
	} else {
		mux.Handle("/", spa(s.Public))
	}
	return secure(mux, s.Cfg.ImageBaseURL, false)
}

func (s *Server) AdminHandler() http.Handler {
	s.ensureLimit()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/accounts", s.listAccounts)
	protected.HandleFunc("POST /api/accounts", s.createAccount)
	protected.HandleFunc("DELETE /api/accounts/{id}", s.deleteAccount)
	protected.HandleFunc("POST /api/accounts/{id}/login", s.rate(8, s.startLogin))
	protected.HandleFunc("GET /api/accounts/{id}/login", s.loginStatus)
	protected.HandleFunc("GET /api/channels", s.listChannels)
	protected.HandleFunc("POST /api/channels/sync", s.syncChannels)
	protected.HandleFunc("PATCH /api/channels/{id}", s.patchChannel)
	protected.HandleFunc("GET /api/feeds", s.listFeeds)
	protected.HandleFunc("POST /api/feeds", s.createFeed)
	protected.HandleFunc("PATCH /api/feeds/{id}", s.patchFeed)
	protected.HandleFunc("DELETE /api/feeds/{id}", s.deleteFeed)
	protected.HandleFunc("GET /api/themes", s.listThemes)
	protected.HandleFunc("POST /api/themes", s.createTheme)
	protected.HandleFunc("PATCH /api/themes/{id}", s.patchTheme)
	protected.HandleFunc("POST /api/themes/{id}/move", s.moveTheme)
	protected.HandleFunc("DELETE /api/themes/{id}", s.deleteTheme)
	protected.HandleFunc("POST /api/themes/{id}/sources", s.bindThemeSource)
	protected.HandleFunc("DELETE /api/themes/{id}/sources", s.unbindThemeSource)
	protected.HandleFunc("GET /api/settings", s.getSettings)
	protected.HandleFunc("PUT /api/settings", s.putSettings)
	protected.HandleFunc("GET /api/logs", s.logs)
	protected.HandleFunc("GET /api/inbox", s.listInbox)
	protected.HandleFunc("PATCH /api/inbox/{id}", s.patchInbox)
	mux.Handle("/api/", s.auth(protected))
	if s.Cfg.Dev {
		mux.HandleFunc("/", devPage("http://127.0.0.1:5174"))
	} else {
		mux.Handle("/", spa(s.Admin))
	}
	return secure(mux, "", true)
}

func devPage(where string) http.HandlerFunc {
	body := "开发模式不在这里提供页面。打开 " + where + "\n"
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, body)
	}
}

func (s *Server) ensureLimit() {
	if s.limit == nil {
		s.limit = newLimiter()
	}
}

func (s *Server) rate(perMinute int, next http.HandlerFunc) http.HandlerFunc {
	if s.Cfg.Dev {
		return next
	}
	return s.limit.wrap(perMinute, next)
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if !tokensMatch(got, s.Cfg.AdminToken) {
			if s.limit != nil && !s.limit.allow(clientIP(r)+":auth", 20) {
				writeErr(w, http.StatusTooManyRequests, "请求过多")
				return
			}
			writeErr(w, http.StatusUnauthorized, "未授权")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func secure(next http.Handler, imageBase string, allowDataImages bool) http.Handler {
	img := "'self'"
	if allowDataImages {
		img += " data:"
	}
	if u, err := url.Parse(strings.TrimSpace(imageBase)); err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" {
		img += " " + u.Scheme + "://" + u.Host
	}
	policy := "default-src 'self'; img-src " + img + "; style-src 'self'; script-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", policy)
		next.ServeHTTP(w, r)
	})
}

func spa(content fs.FS) http.Handler {
	if content == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "ui not built", http.StatusNotFound)
		})
	}
	files := http.FileServer(http.FS(content))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(content, path); err != nil {
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func tokensMatch(got, want string) bool {
	a := sha256.Sum256([]byte(got))
	b := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request", "path", r.URL.Path, "err", err)
	writeErr(w, http.StatusInternalServerError, "内部错误")
}

func readJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	return nil
}
