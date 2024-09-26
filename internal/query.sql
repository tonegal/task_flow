-- name: GetMaxTaskID :one
SELECT MAX(ID) FROM TASK_T
LIMIT 1;

-- name: CreateTask :one
INSERT INTO TASK_T (
    ID, TYPE, VALUE, STATE, CREATION_TIME, LAST_UPDATE_TIME
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetTaskValue :one
SELECT ID, value FROM TASK_T
WHERE ID = $1 LIMIT 1;

-- name: UpdateTaskState :one
UPDATE TASK_T
    SET
    STATE = $2,
    LAST_UPDATE_TIME = $3
WHERE ID = $1
RETURNING *;
