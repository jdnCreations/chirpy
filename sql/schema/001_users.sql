-- +goose Up
CREATE TABLE users (
  id UUID PRIMARY KEY NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  email TEXT NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE IF EXISTS users;