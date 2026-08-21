package handler

import (
	"io/fs"
	"net/http"

	"go03-volunteer-activity/internal/auth"
	"go03-volunteer-activity/internal/middleware"
	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/respond"
	"go03-volunteer-activity/internal/service"
	"go03-volunteer-activity/internal/store"
)

type Handler struct {
	services *service.Services
	store    store.Store
	sessions *auth.SessionManager
	assets   fs.FS
}

func New(svc *service.Services, st store.Store, sessions *auth.SessionManager, assets fs.FS) *Handler {
	return &Handler{services: svc, store: st, sessions: sessions, assets: assets}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	authMw := middleware.RequireAuth(h.sessions, h.store)
	admin := middleware.Chain(authMw, middleware.RequireAdmin())
	org := middleware.Chain(authMw, middleware.RequireOrganizer())

	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.Handle("POST /api/auth/logout", authMw(http.HandlerFunc(h.Logout)))
	mux.Handle("GET /api/auth/me", authMw(http.HandlerFunc(h.Me)))
	mux.Handle("PUT /api/me/profile", authMw(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("PUT /api/me/password", authMw(http.HandlerFunc(h.ChangePassword)))

	mux.HandleFunc("GET /api/categories", h.Categories)
	mux.HandleFunc("GET /api/certificates/{code}/verify", h.VerifyCert)

	mux.Handle("GET /api/users", admin(http.HandlerFunc(h.ListUsers)))
	mux.Handle("POST /api/users", admin(http.HandlerFunc(h.CreateUser)))
	mux.Handle("GET /api/users/{id}", authMw(http.HandlerFunc(h.GetUser)))
	mux.Handle("POST /api/users/{id}/freeze", admin(http.HandlerFunc(h.FreezeUser)))
	mux.Handle("POST /api/users/{id}/unfreeze", admin(http.HandlerFunc(h.UnfreezeUser)))

	mux.Handle("GET /api/activities", authMw(http.HandlerFunc(h.ListActivities)))
	mux.Handle("POST /api/activities", org(http.HandlerFunc(h.CreateActivity)))
	mux.Handle("GET /api/activities/{id}", authMw(http.HandlerFunc(h.GetActivity)))
	mux.Handle("PUT /api/activities/{id}", org(http.HandlerFunc(h.UpdateActivity)))
	mux.Handle("POST /api/activities/{id}/publish", org(http.HandlerFunc(h.PublishActivity)))
	mux.Handle("POST /api/activities/{id}/close-signup", org(http.HandlerFunc(h.CloseSignup)))
	mux.Handle("POST /api/activities/{id}/start", org(http.HandlerFunc(h.StartActivity)))
	mux.Handle("POST /api/activities/{id}/complete", org(http.HandlerFunc(h.CompleteActivity)))
	mux.Handle("POST /api/activities/{id}/cancel", org(http.HandlerFunc(h.CancelActivity)))
	mux.Handle("POST /api/activities/{id}/refresh-code", org(http.HandlerFunc(h.RefreshCode)))
	mux.Handle("POST /api/activities/{id}/signup", authMw(http.HandlerFunc(h.Signup)))
	mux.Handle("GET /api/activities/{id}/signups", org(http.HandlerFunc(h.ListSignups)))
	mux.Handle("POST /api/activities/{id}/checkin", authMw(http.HandlerFunc(h.CheckIn)))
	mux.Handle("POST /api/activities/{id}/checkout", authMw(http.HandlerFunc(h.CheckOut)))
	mux.Handle("POST /api/activities/{id}/proxy-checkin", org(http.HandlerFunc(h.ProxyCheckIn)))
	mux.Handle("GET /api/activities/{id}/checkins", org(http.HandlerFunc(h.ListCheckIns)))
	mux.Handle("GET /api/activities/{id}/hours", org(http.HandlerFunc(h.ListActivityHours)))
	mux.Handle("POST /api/activities/{id}/feedback", authMw(http.HandlerFunc(h.Feedback)))

	mux.Handle("POST /api/signups/{id}/approve", org(http.HandlerFunc(h.ApproveSignup)))
	mux.Handle("POST /api/signups/{id}/reject", org(http.HandlerFunc(h.RejectSignup)))
	mux.Handle("POST /api/signups/{id}/cancel", authMw(http.HandlerFunc(h.CancelSignup)))
	mux.Handle("GET /api/me/signups", authMw(http.HandlerFunc(h.MySignups)))

	mux.Handle("POST /api/hours", org(http.HandlerFunc(h.SubmitHours)))
	mux.Handle("GET /api/hours/pending", org(http.HandlerFunc(h.PendingHours)))
	mux.Handle("GET /api/me/hours", authMw(http.HandlerFunc(h.MyHours)))
	mux.Handle("POST /api/hours/{id}/submit", org(http.HandlerFunc(h.SubmitHourReview)))
	mux.Handle("POST /api/hours/{id}/approve", org(http.HandlerFunc(h.ApproveHours)))
	mux.Handle("POST /api/hours/{id}/reject", org(http.HandlerFunc(h.RejectHours)))
	mux.Handle("POST /api/hours/{id}/correct", authMw(http.HandlerFunc(h.CorrectHours)))

	mux.Handle("GET /api/me/notifications", authMw(http.HandlerFunc(h.MyNotifications)))
	mux.Handle("POST /api/me/notifications/{id}/read", authMw(http.HandlerFunc(h.ReadNotification)))
	mux.Handle("POST /api/me/notifications/read-all", authMw(http.HandlerFunc(h.ReadAllNotifications)))
	mux.Handle("GET /api/me/hour-ledgers", authMw(http.HandlerFunc(h.MyHourLedgers)))
	mux.Handle("GET /api/me/point-ledgers", authMw(http.HandlerFunc(h.MyPointLedgers)))
	mux.Handle("GET /api/me/certificates", authMw(http.HandlerFunc(h.MyCerts)))

	mux.Handle("GET /api/teams", authMw(http.HandlerFunc(h.ListTeams)))
	mux.Handle("POST /api/teams", org(http.HandlerFunc(h.CreateTeam)))
	mux.Handle("GET /api/teams/{id}/members", authMw(http.HandlerFunc(h.TeamMembers)))
	mux.Handle("POST /api/teams/{id}/members", org(http.HandlerFunc(h.InviteMember)))

	mux.Handle("GET /api/stats", admin(http.HandlerFunc(h.GlobalStats)))
	mux.Handle("GET /api/stats/organizer", org(http.HandlerFunc(h.OrganizerStats)))
	mux.Handle("GET /api/audits", admin(http.HandlerFunc(h.ListAudits)))

	h.registerPageRoutes(mux)
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	respond.OK(w, model.HealthResponse{Status: "ok"})
}

func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	respond.OK(w, model.AllCategories())
}

func (h *Handler) VerifyCert(w http.ResponseWriter, r *http.Request) {
	c, u, err := h.services.Cert.Verify(r.Context(), r.PathValue("code"))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"certificate": c, "user": u})
}
