package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type ProjectMemberResponse struct {
	ID            string  `json:"id"`
	ProjectID     string  `json:"project_id"`
	UserID        string  `json:"user_id"`
	CreatedAt     string  `json:"created_at"`
	UserName      string  `json:"user_name"`
	UserEmail     string  `json:"user_email"`
	UserAvatarURL *string `json:"user_avatar_url"`
}

type grantProjectMemberRequest struct {
	UserID string `json:"user_id"`
}

func projectMemberToResponse(row db.ListProjectMembersRow) ProjectMemberResponse {
	return ProjectMemberResponse{
		ID:            uuidToString(row.ID),
		ProjectID:     uuidToString(row.ProjectID),
		UserID:        uuidToString(row.UserID),
		CreatedAt:     timestampToString(row.CreatedAt),
		UserName:      row.UserName,
		UserEmail:     row.UserEmail,
		UserAvatarURL: textToPtr(row.UserAvatarUrl),
	}
}

func (h *Handler) loadProjectInWorkspace(w http.ResponseWriter, r *http.Request) (db.Project, bool) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	idUUID, ok := parseUUIDOrBadRequest(w, id, "project id")
	if !ok {
		return db.Project{}, false
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return db.Project{}, false
	}
	project, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{
		ID: idUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return db.Project{}, false
	}
	return project, true
}

// ListProjectMembers GET /api/projects/{id}/members
func (h *Handler) ListProjectMembers(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectInWorkspace(w, r)
	if !ok {
		return
	}
	rows, err := h.Queries.ListProjectMembers(r.Context(), project.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list project members")
		return
	}
	resp := make([]ProjectMemberResponse, len(rows))
	for i, row := range rows {
		resp[i] = projectMemberToResponse(row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": resp, "total": len(resp)})
}

// GrantProjectMember POST /api/projects/{id}/members
func (h *Handler) GrantProjectMember(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectInWorkspace(w, r)
	if !ok {
		return
	}
	var req grantProjectMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	userUUID, ok := parseUUIDOrBadRequest(w, req.UserID, "user_id")
	if !ok {
		return
	}
	workspaceID := uuidToString(project.WorkspaceID)
	if _, err := h.getWorkspaceMember(r.Context(), req.UserID, workspaceID); err != nil {
		writeError(w, http.StatusBadRequest, "user is not a workspace member")
		return
	}

	row, err := h.Queries.GrantProjectMember(r.Context(), db.GrantProjectMemberParams{
		ProjectID: project.ID,
		UserID:    userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusConflict, "project member already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to grant project member")
		return
	}

	// Re-read via List so response includes user name/email.
	members, err := h.Queries.ListProjectMembers(r.Context(), project.ID)
	if err != nil {
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":         uuidToString(row.ID),
			"project_id": uuidToString(row.ProjectID),
			"user_id":    uuidToString(row.UserID),
			"created_at": timestampToString(row.CreatedAt),
		})
		return
	}
	for _, m := range members {
		if m.UserID == userUUID {
			writeJSON(w, http.StatusCreated, projectMemberToResponse(m))
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         uuidToString(row.ID),
		"project_id": uuidToString(row.ProjectID),
		"user_id":    uuidToString(row.UserID),
		"created_at": timestampToString(row.CreatedAt),
	})
}

// RevokeProjectMember DELETE /api/projects/{id}/members/{userId}
func (h *Handler) RevokeProjectMember(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectInWorkspace(w, r)
	if !ok {
		return
	}
	userID := chi.URLParam(r, "userId")
	userUUID, ok := parseUUIDOrBadRequest(w, userID, "user id")
	if !ok {
		return
	}
	if err := h.Queries.RevokeProjectMember(r.Context(), db.RevokeProjectMemberParams{
		ProjectID: project.ID,
		UserID:    userUUID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke project member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
