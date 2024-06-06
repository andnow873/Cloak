package usermanager

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"

	gmux "github.com/gorilla/mux"
)

const (
	DefaultSessionsCap = 4
	DefaultUpRate      = 10 * 1024 * 1024 * 1024               // 10Gbps
	DefaultDownRate    = 10 * 1024 * 1024 * 1024               // 10Gbps
	DefaultUpCredit    = 10 * 1024 * 1024 * 1024 * 1024 * 1024 // 10PB
	DefaultDownCredit  = 10 * 1024 * 1024 * 1024 * 1024 * 1024 // 10PB
	DefaultExpiryTime  = 4849059490                            // 2119-01-01 00:00:00 UTC
)

type APIRouter struct {
	*gmux.Router
	manager UserManager
}

func APIRouterOf(manager UserManager) *APIRouter {
	ret := &APIRouter{
		manager: manager,
	}
	ret.registerMux()
	return ret
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func (ar *APIRouter) registerMux() {
	ar.Router = gmux.NewRouter()
	ar.HandleFunc("/admin/users", ar.listAllUsersHlr).Methods("GET")
	ar.HandleFunc("/admin/users/{UID}", ar.getUserInfoHlr).Methods("GET")
	ar.HandleFunc("/admin/users/{UID}", ar.writeUserInfoHlr).Methods("POST")
	ar.HandleFunc("/admin/users/{UID}", ar.deleteUserHlr).Methods("DELETE")
	ar.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
	})
	ar.Use(corsMiddleware)
}

func (ar *APIRouter) listAllUsersHlr(w http.ResponseWriter, r *http.Request) {
	infos, err := ar.manager.ListAllUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := json.Marshal(infos)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(resp)
}

func (ar *APIRouter) getUserInfoHlr(w http.ResponseWriter, r *http.Request) {
	b64UID := gmux.Vars(r)["UID"]
	if b64UID == "" {
		http.Error(w, "UID cannot be empty", http.StatusBadRequest)
	}

	UID, err := base64.URLEncoding.DecodeString(b64UID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	uinfo, err := ar.manager.GetUserInfo(UID)
	if err == ErrUserNotFound {
		http.Error(w, ErrUserNotFound.Error(), http.StatusNotFound)
		return
	}
	resp, err := json.Marshal(uinfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(resp)
}

func (ar *APIRouter) writeUserInfoHlr(w http.ResponseWriter, r *http.Request) {
	b64UID := gmux.Vars(r)["UID"]
	if b64UID == "" {
		http.Error(w, "UID cannot be empty", http.StatusBadRequest)
		return
	}
	UID, err := base64.URLEncoding.DecodeString(b64UID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var uinfo UserInfo
	err = json.NewDecoder(r.Body).Decode(&uinfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !bytes.Equal(UID, uinfo.UID) {
		http.Error(w, "UID mismatch", http.StatusBadRequest)
	}

	if uinfo.SessionsCap == nil || *uinfo.SessionsCap == 0 {
		uinfo.SessionsCap = JustInt32(DefaultSessionsCap)
	}

	if uinfo.UpRate == nil || *uinfo.UpRate == 0 {
		uinfo.UpRate = JustInt64(DefaultUpRate)
	}

	if uinfo.DownRate == nil || *uinfo.DownRate == 0 {
		uinfo.DownRate = JustInt64(DefaultDownRate)
	}

	if uinfo.UpCredit == nil || *uinfo.UpCredit == 0 {
		uinfo.UpCredit = JustInt64(DefaultUpCredit)
	}

	if uinfo.DownCredit == nil || *uinfo.DownCredit == 0 {
		uinfo.DownCredit = JustInt64(DefaultDownCredit)
	}

	if uinfo.ExpiryTime == nil || *uinfo.ExpiryTime == 0 {
		uinfo.ExpiryTime = JustInt64(DefaultExpiryTime)
	}

	err = ar.manager.WriteUserInfo(uinfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusCreated)
}

func (ar *APIRouter) deleteUserHlr(w http.ResponseWriter, r *http.Request) {
	b64UID := gmux.Vars(r)["UID"]
	if b64UID == "" {
		http.Error(w, "UID cannot be empty", http.StatusBadRequest)
		return
	}
	UID, err := base64.URLEncoding.DecodeString(b64UID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = ar.manager.DeleteUser(UID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}
