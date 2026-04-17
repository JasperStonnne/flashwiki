CREATE TABLE nodes (
  id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_id  uuid        REFERENCES nodes(id) ON DELETE RESTRICT,
  kind       text        NOT NULL CHECK (kind IN ('folder','doc')),
  title      text        NOT NULL,
  owner_id   uuid        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  tsv        tsvector
);
CREATE INDEX nodes_parent_idx         ON nodes(parent_id);
CREATE INDEX nodes_live_idx           ON nodes(parent_id) WHERE deleted_at IS NULL;
CREATE INDEX nodes_tsv_gin_idx        ON nodes USING GIN (tsv);
CREATE INDEX nodes_title_trgm_idx     ON nodes USING GIN (title gin_trgm_ops);

CREATE FUNCTION nodes_tsv_trigger() RETURNS trigger AS $$
BEGIN
  NEW.tsv := to_tsvector('chinese', coalesce(NEW.title, ''));
  NEW.updated_at := now();
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER nodes_tsv_upd
BEFORE INSERT OR UPDATE OF title ON nodes
FOR EACH ROW EXECUTE FUNCTION nodes_tsv_trigger();

CREATE TABLE node_permissions (
  id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id      uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  subject_type text        NOT NULL CHECK (subject_type IN ('user','group')),
  subject_id   uuid        NOT NULL,
  level        text        NOT NULL CHECK (level IN ('manage','edit','readable','none')),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (node_id, subject_type, subject_id)
);
CREATE INDEX node_permissions_node_idx    ON node_permissions(node_id);
CREATE INDEX node_permissions_subject_idx ON node_permissions(subject_type, subject_id);
