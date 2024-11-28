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
ORDER BY 
  CASE WHEN $1 = 'desc' then created_at END DESC,
  CASE WHEN $1 = 'asc' then created_at END ASC;

-- name: GetChirpsByUserId :many
SELECT * from chirps
WHERE user_id = $1
ORDER BY 
  CASE WHEN $2 = 'desc' then created_at END DESC,
  CASE WHEN $2 = 'asc' then created_at END ASC;

-- name: GetChirpById :one
SELECT * from chirps
WHERE id = $1;

-- name: DeleteChirpById :exec
DELETE FROM chirps WHERE id = $1;