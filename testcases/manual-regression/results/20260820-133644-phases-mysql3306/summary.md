# phases regression summary

- Target: `8.0.39`
- Generated: `2026-08-20T13:36:45.726868+08:00`
- PASS=53, FAIL=0, ERROR=0, MANUAL=48, SKIP=1

| Case | Status | Rule | Duration(ms) | Expected | Actual |
|---|---|---|---:|---|---|
| P1-001 | PASS |  | 10 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED', 'CHECKED'], rows=2 |
| P1-002 | PASS |  | 1 | protocol error 1064 | protocol_error=1064, message=(1064, 'request must contain inception_magic_start, SQL, and inception_magic_commit') |
| P1-003 | PASS |  | 0 | protocol error 1064 | protocol_error=1064, message=(1064, 'duplicate or misplaced inception control marker') |
| P1-004 | PASS |  | 1 | protocol error 1064 | protocol_error=1064, message=(1064, 'first statement must be inception_magic_start') |
| P1-005 | PASS |  | 8 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED'], rows=1 |
| P1-006 | PASS |  | 9 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED'], rows=1 |
| P1-007 | PASS |  | 0 | protocol error 1064 | protocol_error=1064, message=(1064, 'unknown inception directive "not_exists"') |
| P1-008 | PASS |  | 0 | protocol error 1064 | protocol_error=1064, message=(1064, 'invalid target port "70000"') |
| P1-009 | PASS |  | 0 | protocol error 1064 | protocol_error=1064, message=(1064, 'backup requires execute=1') |
| P1-010 | PASS |  | 9 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED', 'CHECKED'], rows=2 |
| P1-011 | PASS |  | 4 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P1-012 | PASS |  | 2 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P1-013 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P1-014 | PASS |  | 11 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED', 'CHECKED', 'CHECKED'], rows=3 |
| P2-001 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P2-002 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P2-003 | PASS |  | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P2-004 | PASS |  | 3 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P2-005 | PASS |  | 5 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED', 'CHECKED', 'CHECKED'], rows=3 |
| P2-006 | PASS |  | 6 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED', 'CHECKED', 'CHECKED'], rows=3 |
| P2-007 | PASS |  | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED', 'CHECKED', 'CHECKED'], rows=3 |
| P2-008 | PASS |  | 5 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED', 'CHECKED', 'CHECKED'], rows=3 |
| P2-009 | PASS |  | 11 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED'], rows=1 |
| P2-010 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P2-011 | PASS |  | 6 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED', 'CHECKED', 'CHECKED'], rows=3 |
| P2-012 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P2-013 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P2-014 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P3-001 | PASS |  | 9 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED'], rows=1 |
| P3-002 | PASS | GIP-DML-SF-007 | 10 | level=1, stage=CHECKED | max_level=1, stages=['CHECKED'], rows=1 |
| P3-003 | PASS | GIP-DML-IM-003 | 9 | level=1, stage=CHECKED | max_level=1, stages=['CHECKED'], rows=1 |
| P3-004 | PASS | GIP-DML-SF-001 | 15 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED', 'CHECKED'], rows=2 |
| P3-005 | PASS |  | 9 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P3-006 | PASS | GIP-DML-SF-005 | 13 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P3-007 | PASS | GIP-DML-SF-006 | 9 | level=1, stage=CHECKED | max_level=1, stages=['CHECKED'], rows=1 |
| P3-008 | PASS | GIP-DML-SF-008 | 10 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P3-009 | PASS | GIP-DML-SF-010 | 11 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P3-010 | PASS | GIP-DML-MD-001 | 8 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P3-011 | PASS | GIP-DML-IM-002 | 10 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P3-012 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P3-013 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P3-014 | PASS |  | 15 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED'], rows=1 |
| P3-015 | PASS |  | 14 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED'], rows=1 |
| P3-016 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P3-017 | SKIP |  | 0 | level=2, stage=CHECKED | not applicable to MySQL major 8 |
| P4-001 | PASS |  | 17 | level=0, stage=EXECUTED | max_level=0, stages=['EXECUTED', 'EXECUTED', 'EXECUTED'], rows=3 |
| P4-002 | PASS |  | 8 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED', 'CHECKED'], rows=2; verify=0, want=0 |
| P4-003 | PASS |  | 11 | level=2, stage=EXECUTED | max_level=2, stages=['EXECUTED', 'CHECKED'], rows=2; verify=0, want=0 |
| P4-004 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P4-005 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P4-006 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P4-007 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P4-008 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P4-009 | MANUAL |  | 0 | manual verification | Requires an isolated service started with audit_only=true |
| P4-010 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P5-001 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P5-002 | PASS |  | 50 | level=0, stage=BACKUP | max_level=0, stages=['BACKUP'], rows=1 |
| P5-003 | PASS |  | 42 | level=0, stage=BACKUP | max_level=0, stages=['BACKUP'], rows=1 |
| P5-004 | PASS |  | 39 | level=0, stage=BACKUP | max_level=0, stages=['BACKUP'], rows=1 |
| P5-005 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P5-006 | PASS | GIP-DML-SF-002 | 12 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1 |
| P5-007 | PASS |  | 42 | level=0, stage=BACKUP | max_level=0, stages=['BACKUP'], rows=1 |
| P5-008 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P5-009 | PASS |  | 54 | level=0, stage=BACKUP | max_level=0, stages=['BACKUP'], rows=1 |
| P5-010 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P5-011 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P5-012 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P5-013 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P5-014 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P5-015 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-001 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-002 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-003 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-004 | PASS |  | 1 | level=0, stage=CHECKED | columns=['VERSION()'], rows=1 |
| P6-005 | PASS |  | 0 | level=0, stage=CHECKED | columns=['Database'], rows=1 |
| P6-006 | PASS |  | 0 | level=0, stage=CHECKED | columns=['Tables_in_information_schema'], rows=0 |
| P6-007 | PASS |  | 0 | protocol error 1146 | protocol_error=1146, message=(1146, "Table 'information_schema.missing_table' doesn't exist") |
| P6-008 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-009 | PASS |  | 0 | level=0, stage=CHECKED | columns=['Grants for archery@%'], rows=1 |
| P6-010 | PASS |  | 0 | level=0, stage=CHECKED | columns=['CONNECTION_ID()'], rows=1 |
| P6-011 | PASS |  | 0 | protocol error 1235 | protocol_error=1235, message=(1235, 'only inception audit requests and goInception management commands are supported') |
| P6-012 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-013 | PASS |  | 6 | level=0, stage=CHECKED | max_level=0, stages=['CHECKED'], rows=1 |
| P6-014 | PASS |  | 1 | level=0, stage=CHECKED | columns=['Name', 'Value', 'Desc'], rows=24 |
| P6-015 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-016 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-017 | PASS |  | 0 | protocol error 1094 | protocol_error=1094, message=(1094, 'Unknown thread id: 4294967294') |
| P6-018 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-019 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P6-020 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-001 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-002 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-003 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-004 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-005 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-006 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-007 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-008 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-009 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-010 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-011 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
| P7-012 | MANUAL |  | 0 | manual verification | Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md |
