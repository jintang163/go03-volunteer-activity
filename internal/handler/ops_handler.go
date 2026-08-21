package handler

import (
	"net/http"

	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/respond"
)

func (h *Handler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var req model.CheckInRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.CheckIn.SelfCheckIn(r.Context(), userFrom(r), pathID(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) CheckOut(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.CheckIn.CheckOut(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ProxyCheckIn(w http.ResponseWriter, r *http.Request) {
	var req model.ProxyCheckInRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.CheckIn.ProxyCheckIn(r.Context(), userFrom(r), pathID(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) ListCheckIns(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.CheckIn.List(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) SubmitHours(w http.ResponseWriter, r *http.Request) {
	var req model.SubmitHoursRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Hours.SubmitManual(r.Context(), userFrom(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) PendingHours(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Hours.Pending(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MyHours(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Hours.Mine(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ListActivityHours(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Hours.ByActivity(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) SubmitHourReview(w http.ResponseWriter, r *http.Request) {
	var req model.ReviewHoursRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Hours.SubmitAutoForReview(r.Context(), userFrom(r), pathID(r), req.BreakMinutes)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ApproveHours(w http.ResponseWriter, r *http.Request) {
	var req model.ReviewHoursRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Hours.Approve(r.Context(), userFrom(r), pathID(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) RejectHours(w http.ResponseWriter, r *http.Request) {
	var req model.RejectRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Hours.Reject(r.Context(), userFrom(r), pathID(r), req.Reason)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CorrectHours(w http.ResponseWriter, r *http.Request) {
	var req model.CorrectHoursRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Hours.RequestCorrection(r.Context(), userFrom(r), pathID(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Team.ListMine(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTeamRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Team.Create(r.Context(), userFrom(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) TeamMembers(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Team.Members(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) InviteMember(w http.ResponseWriter, r *http.Request) {
	var req model.InviteMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := h.services.Team.Invite(r.Context(), userFrom(r), pathID(r), req.Username)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) GlobalStats(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Stats.Global(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) OrganizerStats(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Stats.Organizer(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ListAudits(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Audit.List(r.Context(), userFrom(r), queryStr(r, "target_type"), queryStr(r, "target_id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}
