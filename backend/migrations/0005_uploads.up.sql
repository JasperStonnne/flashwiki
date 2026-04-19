CREATE TABLE uploads (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id    uuid        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  node_id     uuid        REFERENCES nodes(id) ON DELETE SET NULL,
  filename    text        NOT NULL,
  stored_path text        NOT NULL,
  mime_type   text        NOT NULL,
  size_bytes  bigint      NOT NULL CHECK (size_bytes > 0),
  sha256      text        NOT NULL UNIQUE,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX uploads_node_idx  ON uploads(node_id) WHERE node_id IS NOT NULL;
CREATE INDEX uploads_owner_idx ON uploads(owner_id);
