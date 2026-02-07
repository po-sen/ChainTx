CREATE SCHEMA IF NOT EXISTS app;

CREATE TABLE IF NOT EXISTS app.bootstrap_metadata (
  key text PRIMARY KEY,
  value text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO app.bootstrap_metadata (key, value)
VALUES ('bootstrap_version', 'v1')
ON CONFLICT (key) DO NOTHING;
