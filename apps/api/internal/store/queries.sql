-- name: ListEntries :many
SELECT id, title, content, type, link_url, date, completed, color, tags, created_at
FROM entries
WHERE user_id = $1
  AND date >= $2
  AND date <= $3
ORDER BY date ASC, created_at ASC;
