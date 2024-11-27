-- name: CreateChirp :one
/* @param UserId uuid */
INSERT INTO chirps (id, created_at, updated_at, body, user_id) 
VALUES(
     gen_random_uuid(),
     NOW(),
     NOW(),
     $1,
     $2
) RETURNING *;

-- name: GetAllChirps :many
SELECT * from chirps
ORDER BY created_at ASC;

-- name: GetChirpById :one
SELECT * from chirps
WHERE id = $1;