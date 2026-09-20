package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	items, err := s.Accounts.ListAccounts(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []kernel.Account{}
	}
	for i := range items {
		items[i].APIHash = mask(items[i].APIHash)
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		APIID   int    `json:"api_id"`
		APIHash string `json:"api_hash"`
		Phone   string `json:"phone"`
	}
	if err := readJSON(w, r, &body); err != nil || body.Name == "" || body.APIID == 0 || body.APIHash == "" {
		writeErr(w, http.StatusBadRequest, "需要 name、api_id、api_hash")
		return
	}
	acc, err := s.Accounts.CreateAccount(r.Context(), strings.TrimSpace(body.Name), body.APIID, strings.TrimSpace(body.APIHash), strings.TrimSpace(body.Phone))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无法创建账号")
		return
	}
	acc.APIHash = mask(acc.APIHash)
	writeJSON(w, http.StatusCreated, acc)
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	if s.TG != nil {
		s.TG.Forget(id)
	}
	if err := s.Accounts.DeleteAccount(r.Context(), id); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) startLogin(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	_ = readJSON(w, r, &body)
	s.TG.StartQR(id, body.Password)
	writeJSON(w, http.StatusAccepted, s.TG.Login(id))
}

func (s *Server) loginStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	writeJSON(w, http.StatusOK, s.TG.Login(id))
}

func (s *Server) listChannels(w http.ResponseWriter, r *http.Request) {
	var accountID int64
	if raw := r.URL.Query().Get("account_id"); raw != "" {
		accountID, _ = strconv.ParseInt(raw, 10, 64)
	}
	items, err := s.Accounts.ListChannels(r.Context(), accountID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []kernel.Channel{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) syncChannels(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AccountID int64 `json:"account_id"`
	}
	if err := readJSON(w, r, &body); err != nil || body.AccountID == 0 {
		writeErr(w, http.StatusBadRequest, "需要 account_id")
		return
	}
	n, err := s.TG.Sync(r.Context(), body.AccountID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"synced": n})
}

func (s *Server) patchChannel(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	var body struct {
		Disabled bool `json:"disabled"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效 JSON")
		return
	}
	if err := s.Accounts.SetChannelDisabled(r.Context(), id, body.Disabled); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
