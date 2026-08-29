CREATE TABLE project_member (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(project_id, user_id)
);
CREATE INDEX project_member_user_id_idx ON project_member(user_id);

INSERT INTO project_member (project_id, user_id)
SELECT p.id, m.user_id FROM member m
JOIN project p ON p.workspace_id = m.workspace_id
ON CONFLICT DO NOTHING;
