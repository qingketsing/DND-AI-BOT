package router

import (
	"net/http"
	"strings"

	"DND-AI-BOT/internal/transport/http/handler"
)

// NewRouter 创建最小 HTTP 路由器，并将请求分发到会话处理器。
func NewRouter(
	sessionHandler *handler.SessionHandler,
	gameStateHandler *handler.GameStateHandler,
	encounterHandler *handler.EncounterHandler,
	authHandler *handler.AuthHandler,
	authMiddleware func(http.Handler) http.Handler,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/register", authHandler.Register)
	mux.HandleFunc("/auth/login", authHandler.Login)
	mux.HandleFunc("/auth/logout", authHandler.Logout)
	mux.Handle("/auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))

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
		if strings.HasSuffix(r.URL.Path, "/game-state/stats") {
			gameStateHandler.UpdateStats(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/game-state/items/add") {
			gameStateHandler.AddItem(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/game-state/items/remove") {
			gameStateHandler.RemoveItem(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/game-state/gold/add") {
			gameStateHandler.AddGold(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/game-state/gold/spend") {
			gameStateHandler.SpendGold(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/game-state/scene") {
			gameStateHandler.SetScene(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/game-state/quests") {
			gameStateHandler.UpsertQuest(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/game-state") {
			if r.Method == http.MethodPost {
				gameStateHandler.CreateGameState(w, r)
				return
			}
			gameStateHandler.GetGameState(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/encounter/damage") {
			encounterHandler.ApplyDamage(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/encounter/heal") {
			encounterHandler.Heal(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/encounter/turn/advance") {
			encounterHandler.AdvanceTurn(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/encounter/effects/add") {
			encounterHandler.AddEffect(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/encounter/effects/remove") {
			encounterHandler.RemoveEffect(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/encounter/can-act") {
			encounterHandler.CanAct(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/encounter") {
			if r.Method == http.MethodPost {
				encounterHandler.CreateEncounter(w, r)
				return
			}
			encounterHandler.GetEncounter(w, r)
			return
		}

		sessionHandler.GetSession(w, r)
	})

	return mux
}
