USE gip_phase_test;

SELECT 'users-final' AS checkpoint, id, username, score
FROM users
ORDER BY id;

SELECT 'audit-blocked-id-901-must-be-zero' AS checkpoint, COUNT(*) AS actual
FROM users WHERE id = 901;

SELECT 'runtime-stop-id-902-must-be-zero' AS checkpoint, COUNT(*) AS actual
FROM users WHERE id = 902;

SELECT 'backup-fail-closed-score-must-remain-10.50' AS checkpoint, score AS actual
FROM users WHERE id = 1;

SELECT 'phase4-ddl-column' AS checkpoint, COLUMN_NAME, COLUMN_TYPE
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = 'gip_phase_test'
  AND TABLE_NAME = 'phase4_ddl'
ORDER BY ORDINAL_POSITION;
