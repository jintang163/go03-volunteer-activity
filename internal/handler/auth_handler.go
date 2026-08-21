package handler

import (
	"net/http"

	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/respond"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Auth.Register(r.Context(), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Auth.Login(r.Context(), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.services.Auth.Logout(r.Context(), extractBearer(r))
	respond.NoContent(w)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Auth.Me(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateProfileRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.User.UpdateProfile(r.Context(), userFrom(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req model.ChangePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.services.User.ChangePassword(r.Context(), userFrom(r), extractBearer(r), req); err != nil {
		writeErr(w, err)
		return
	}
	respond.NoContent(w)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	f := model.UserFilter{Query: queryStr(r, "q")}
	if role, ok := model.ParseUserRole(queryStr(r, "role")); ok {
		f.Role = role
	}
	f.Status = model.UserStatus(queryStr(r, "status"))
	out, err := h.services.User.List(r.Context(), userFrom(r), f)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.User.Create(r.Context(), userFrom(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.Get(r.Context(), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) FreezeUser(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.SetStatus(r.Context(), userFrom(r), pathID(r), model.UserFrozen)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UnfreezeUser(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.SetStatus(r.Context(), userFrom(r), pathID(r), model.UserActive)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MyNotifications(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.Notifications(r.Context(), userFrom(r), parseBoolQuery(r, "unread", false))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ReadNotification(w http.ResponseWriter, r *http.Request) {
	if err := h.services.User.ReadNotification(r.Context(), userFrom(r), pathID(r)); err != nil {
		writeErr(w, err)
		return
	}
	respond.NoContent(w)
}

func (h *Handler) ReadAllNotifications(w http.ResponseWriter, r *http.Request) {
	n, err := h.services.User.ReadAll(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]int{"read": n})
}

func (h *Handler) MyHourLedgers(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.HourLedgers(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MyPointLedgers(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.PointLedgers(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MyCerts(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Cert.ListMine(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}
