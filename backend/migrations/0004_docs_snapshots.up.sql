CREATE TABLE doc_states (
  node_id            uuid        PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
  ydoc_state         bytea       NOT NULL,
  version            bigint      NOT NULL DEFAULT 0,
  markdown_plain     text        NOT NULL DEFAULT '',
  markdown_plain_tsv tsvector,
  last_compacted_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX doc_states_tsv_gin_idx ON doc_states USING GIN (markdown_plain_tsv);

CREATE FUNCTION doc_states_tsv_trigger() RETURNS trigger AS $$
BEGIN
  NEW.markdown_plain_tsv := to_tsvector('chinese', coalesce(NEW.markdown_plain, ''));
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER doc_states_tsv_upd
BEFORE INSERT OR UPDATE OF markdown_plain ON doc_states
FOR EACH ROW EXECUTE FUNCTION doc_states_tsv_trigger();

CREATE TABLE doc_updates (
  id         bigserial   PRIMARY KEY,
  node_id    uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  update     bytea       NOT NULL,
  client_id  text        NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX doc_updates_node_idx ON doc_updates(node_id, id);

CREATE TABLE doc_snapshots (
  id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id         uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  ydoc_state      bytea       NOT NULL,
  title           text        NOT NULL,
  created_by      uuid        NOT NULL REFERENCES users(id),
  trigger_reason  text        NOT NULL CHECK (trigger_reason IN ('manual','scheduled_daily')),
  created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX doc_snapshots_node_idx ON doc_snapshots(node_id, created_at DESC);
