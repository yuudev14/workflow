package queries

const INSERT_WORKFLOW = `
INSERT INTO workflows (name, description, trigger_type) 
VALUES ($1, $2, $3) 
RETURNING *`

const UPDATE_WORKFLOW = `
UPDATE workflows
SET %v
WHERE id = ($1)
RETURNING *`
