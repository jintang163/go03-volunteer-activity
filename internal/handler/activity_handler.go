package handler

import (
	"net/http"

	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/respond"
)

func (h *Handler) ListActivities(w http.ResponseWriter, r *http.Request) {
	f := model.ActivityFilter{
		Query:       queryStr(r, "q"),
		Category:    model.CategoryID(queryStr(r, "category")),
		Status:      model.ActivityStatus(queryStr(r, "status")),
		OrganizerID: queryStr(r, "organizer_id"),
		TeamID:      queryStr(r, "team_id"),
	}
	out, err := h.services.Activity.List(r.Context(), userFrom(r), f)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CreateActivity(w http.ResponseWriter, r *http.Request) {
	var req model.CreateActivityRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Activity.Create(r.Context(), userFrom(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) GetActivity(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Activity.Get(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UpdateActivity(w http.ResponseWriter, r *http.Request) {
	var req model.CreateActivityRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Activity.Update(r.Context(), userFrom(r), pathID(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) PublishActivity(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Activity.Publish(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CloseSignup(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Activity.CloseSignup(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) StartActivity(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Activity.Start(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CompleteActivity(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Activity.Complete(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CancelActivity(w http.ResponseWriter, r *http.Request) {
	var req model.CancelRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Activity.Cancel(r.Context(), userFrom(r), pathID(r), req.Reason)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) RefreshCode(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Activity.RefreshCode(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req model.SignupRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Signup.Signup(r.Context(), userFrom(r), pathID(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) ListSignups(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Signup.ListByActivity(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ApproveSignup(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Signup.Approve(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) RejectSignup(w http.ResponseWriter, r *http.Request) {
	var req model.RejectRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Signup.Reject(r.Context(), userFrom(r), pathID(r), req.Reason)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CancelSignup(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Signup.Cancel(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MySignups(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Signup.Mine(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) Feedback(w http.ResponseWriter, r *http.Request) {
	var req model.FeedbackRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Activity.SubmitFeedback(r.Context(), userFrom(r), pathID(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}
