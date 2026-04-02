package router

import (
	"net/http"
	"strings"

	"../handler"
)

// NewRouter 创建最小 HTTP 路由器，并将请求分发到会话处理器。
func NewRouter(sessionHandler *handler.SessionHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			sessionHandler.CreateSession(w, r)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/sessions/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/messages") {
			sessionHandler.SendMessage(w, r)
			return
		}

		sessionHandler.GetSession(w, r)
	})

	return mux
}
