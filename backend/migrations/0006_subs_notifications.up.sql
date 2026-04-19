CREATE TABLE subscriptions (
  user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  node_id    uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, node_id)
);
CREATE INDEX subscriptions_node_idx ON subscriptions(node_id);

CREATE TABLE notifications (
  id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  node_id    uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  event_type text        NOT NULL,
  payload    jsonb       NOT NULL DEFAULT '{}'::jsonb,
  read_at    timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_unread_idx
  ON notifications(user_id, created_at DESC)
  WHERE read_at IS NULL;
