-- +goose Up
CREATE TABLE chirps (
  id UUID PRIMARY KEY NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  body TEXT NOT NULL,
  user_id UUID REFERENCES users(id) on delete CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS chirps;