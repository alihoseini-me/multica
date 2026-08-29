package handler

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// accessibleProjectIDs returns (ids, allAccess).
// allAccess=true → no project filter (owner/admin/PAT/daemon/task_token/cloud_pat/agent).
func (h *Handler) accessibleProjectIDs(ctx context.Context, r *http.Request, workspaceID string) ([]pgtype.UUID, bool, error) {
	if middleware.DaemonIDFromContext(ctx) != "" || middleware.DaemonWorkspaceIDFromContext(ctx) != "" {
		return nil, true, nil
	}

	switch r.Header.Get("X-Actor-Source") {
	case "task_token", "cloud_pat", "pat":
		return nil, true, nil
	}

	userID := requestUserID(r)
	if userID != "" {
		if actorType, _ := h.resolveActor(r, userID, workspaceID); actorType == "agent" {
			return nil, true, nil
		}
	}

	role := ""
	if m, ok := middleware.MemberFromContext(ctx); ok {
		role = m.Role
	} else if userID != "" {
		if m, err := h.getWorkspaceMember(ctx, userID, workspaceID); err == nil {
			role = m.Role
		}
	}
	if role == "owner" || role == "admin" {
		return nil, true, nil
	}

	wsUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return nil, false, err
	}
	userUUID, err := util.ParseUUID(userID)
	if err != nil {
		return nil, false, err
	}
	ids, err := h.Queries.ListAccessibleProjectIds(ctx, db.ListAccessibleProjectIdsParams{
		WorkspaceID: wsUUID,
		UserID:      userUUID,
	})
	if err != nil {
		return nil, false, err
	}
	if ids == nil {
		ids = []pgtype.UUID{}
	}
	return ids, false, nil
}

// canAccessProject reports whether the caller may see/mutate the given project.
// A NULL/invalid projectID is only allowed for allAccess principals.
func (h *Handler) canAccessProject(ctx context.Context, r *http.Request, workspaceID string, projectID pgtype.UUID) bool {
	accessible, allAccess, err := h.accessibleProjectIDs(ctx, r, workspaceID)
	if err != nil {
		return false
	}
	if allAccess {
		return true
	}
	if !projectID.Valid {
		return false
	}
	for _, id := range accessible {
		if id.Valid && id.Bytes == projectID.Bytes {
			return true
		}
	}
	return false
}

// restrictProjectFilter intersects client project_ids with accessible set.
// For restricted members: force includeNoProject=false; if no accessible projects → empty=true (caller returns empty list).
func restrictProjectFilter(all bool, accessible []pgtype.UUID, requested []pgtype.UUID, includeNoProject bool) (ids []pgtype.UUID, includeNull bool, empty bool) {
	if all {
		return requested, includeNoProject, false
	}

	if len(requested) == 0 {
		if len(accessible) == 0 {
			return nil, false, true
		}
		return accessible, false, false
	}

	allowed := make(map[[16]byte]struct{}, len(accessible))
	for _, id := range accessible {
		if id.Valid {
			allowed[id.Bytes] = struct{}{}
		}
	}
	out := make([]pgtype.UUID, 0, len(requested))
	for _, id := range requested {
		if !id.Valid {
			continue
		}
		if _, ok := allowed[id.Bytes]; ok {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, false, true
	}
	return out, false, false
}
