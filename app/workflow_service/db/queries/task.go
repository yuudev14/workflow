package queries

const GET_TASK_BY_WORKFLOW_ID = `SELECT * from tasks WHERE workflow_id = $1`
