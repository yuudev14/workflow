package queries

const INSERT_EDGES = `INSERT INTO tasks (destination_id, source_id) VALUES %v RETURNING *;`
