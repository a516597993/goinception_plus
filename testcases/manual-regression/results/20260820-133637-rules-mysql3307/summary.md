# rules regression summary

- Target: `5.7.44-log`
- Generated: `2026-08-20T13:36:41.946115+08:00`
- PASS=68, FAIL=0, ERROR=0, MANUAL=8, SKIP=0

| Case | Status | Rule | Duration(ms) | Expected | Actual |
|---|---|---|---:|---|---|
| GIP-DDL-CT-001 | PASS | GIP-DDL-CT-001 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-001'] |
| GIP-COR-ST-001 | PASS | GIP-COR-ST-001 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-COR-ST-001'] |
| GIP-COR-PS-001 | MANUAL | GIP-COR-PS-001 | 0 | manual verification | Requires a stable parser-warning corpus; no known deterministic warning SQL is currently exposed |
| GIP-COR-PS-002 | PASS | GIP-COR-PS-002 | 6 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-COR-PS-002'] |
| GIP-COR-SF-001 | MANUAL | GIP-COR-SF-001 | 0 | manual verification | Requires service restart with audit_only=true |
| GIP-META-CN-001 | MANUAL | GIP-META-CN-001 | 0 | manual verification | Requires invalid credentials/network or permission fault injection |
| GIP-META-CN-002 | MANUAL | GIP-META-CN-002 | 0 | manual verification | Version-specific case is covered by phase/version suite; run separately against 5.7 |
| GIP-META-DB-001 | PASS | GIP-META-DB-001 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-META-DB-001'] |
| GIP-META-DB-002 | PASS | GIP-META-DB-002 | 4 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-META-DB-002'] |
| GIP-DDL-MD-001 | PASS | GIP-DDL-MD-001 | 6 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-MD-001'] |
| GIP-DDL-CT-002 | PASS | GIP-DDL-CT-002 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-002'] |
| GIP-DDL-CT-003 | PASS | GIP-DDL-CT-003 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-003'] |
| GIP-DDL-CT-004 | PASS | GIP-DDL-CT-004 | 6 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-004'] |
| GIP-DDL-CT-005 | PASS | GIP-DDL-CT-005 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-005'] |
| GIP-DDL-CT-006 | PASS | GIP-DDL-CT-006 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-006'] |
| GIP-DDL-CT-007 | PASS | GIP-DDL-CT-007 | 6 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-007'] |
| GIP-DDL-CT-008 | PASS | GIP-DDL-CT-008 | 4 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-008'] |
| GIP-DDL-CT-009 | PASS | GIP-DDL-CT-009 | 4 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-009'] |
| GIP-DDL-CT-010 | PASS | GIP-DDL-CT-010 | 6 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-010'] |
| GIP-DDL-CT-011 | PASS | GIP-DDL-CT-011 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-011'] |
| GIP-DDL-CT-012 | PASS | GIP-DDL-CT-012 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-012'] |
| GIP-DDL-CT-013 | PASS | GIP-DDL-CT-013 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-013'] |
| GIP-DDL-CT-014 | PASS | GIP-DDL-CT-014 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-014'] |
| GIP-DDL-CT-015 | PASS | GIP-DDL-CT-015 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-015'] |
| GIP-DDL-CT-016 | PASS | GIP-DDL-CT-016 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-016'] |
| GIP-DDL-CT-017 | PASS | GIP-DDL-CT-017 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-017'] |
| GIP-DDL-CT-018 | PASS | GIP-DDL-CT-018 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-018'] |
| GIP-DDL-CT-019 | PASS | GIP-DDL-CT-019 | 6 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-019'] |
| GIP-DDL-CT-020 | PASS | GIP-DDL-CT-020 | 4 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-020'] |
| GIP-DDL-CT-021 | PASS | GIP-DDL-CT-021 | 7 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-021'] |
| GIP-DDL-CT-022 | PASS | GIP-DDL-CT-022 | 6 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-022'] |
| GIP-DDL-CT-023 | PASS | GIP-DDL-CT-023 | 4 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-023'] |
| GIP-DDL-CT-024 | PASS | GIP-DDL-CT-024 | 7 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-024'] |
| GIP-DDL-CT-025 | PASS | GIP-DDL-CT-025 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-025'] |
| GIP-DDL-CT-026 | PASS | GIP-DDL-CT-026 | 6 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-026'] |
| GIP-DDL-CT-027 | PASS | GIP-DDL-CT-027 | 27 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-027'] |
| GIP-DDL-CT-028 | PASS | GIP-DDL-CT-028 | 7 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-028'] |
| GIP-DDL-CT-029 | PASS | GIP-DDL-CT-029 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-029'] |
| GIP-DDL-CT-030 | PASS | GIP-DDL-CT-030 | 164 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-030'] |
| GIP-DDL-CT-031 | PASS | GIP-DDL-CT-031 | 158 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-031'] |
| GIP-DDL-CT-032 | PASS | GIP-DDL-CT-032 | 7 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-032'] |
| GIP-DDL-CT-033 | PASS | GIP-DDL-CT-033 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-033'] |
| GIP-DDL-CT-034 | PASS | GIP-DDL-CT-034 | 7 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-034'] |
| GIP-DDL-CT-035 | PASS | GIP-DDL-CT-035 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-035'] |
| GIP-DDL-CT-036 | PASS | GIP-DDL-CT-036 | 4 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-036'] |
| GIP-DDL-CT-037 | PASS | GIP-DDL-CT-037 | 7 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-CT-037'] |
| GIP-DDL-AT-001 | PASS | GIP-DDL-AT-001 | 162 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED', 'CHECKED'], rows=2; observed_rules=['GIP-DDL-AT-001'] |
| GIP-DDL-AT-002 | PASS | GIP-DDL-AT-002 | 168 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-AT-002'] |
| GIP-DDL-AT-003 | PASS | GIP-DDL-AT-003 | 172 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-AT-003'] |
| GIP-DDL-AT-004 | PASS | GIP-DDL-AT-004 | 169 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-AT-004'] |
| GIP-DDL-SF-001 | PASS | GIP-DDL-SF-001 | 8 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-SF-001'] |
| GIP-DDL-SF-002 | PASS | GIP-DDL-SF-002 | 159 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-SF-002'] |
| GIP-DDL-SF-003 | PASS | GIP-DDL-SF-003 | 160 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-SF-003'] |
| GIP-DDL-SF-004 | PASS | GIP-DDL-SF-004 | 7 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-SF-004'] |
| GIP-DDL-SF-005 | PASS | GIP-DDL-SF-005 | 7 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-SF-005'] |
| GIP-DDL-SF-006 | PASS | GIP-DDL-SF-006 | 5 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-SF-006'] |
| GIP-DDL-SF-007 | PASS | GIP-DDL-SF-007 | 4 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-SF-007'] |
| GIP-DDL-SF-008 | PASS | GIP-DDL-SF-008 | 178 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-SF-008'] |
| GIP-DDL-SF-009 | PASS | GIP-DDL-SF-009 | 7 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DDL-SF-009'] |
| GIP-DML-MD-001 | PASS | GIP-DML-MD-001 | 171 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-MD-001'] |
| GIP-DML-SF-001 | PASS | GIP-DML-SF-001 | 170 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-001'] |
| GIP-DML-SF-002 | PASS | GIP-DML-SF-002 | 159 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-002'] |
| GIP-DML-SF-003 | PASS | GIP-DML-SF-003 | 161 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-003'] |
| GIP-DML-SF-004 | PASS | GIP-DML-SF-004 | 158 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-004'] |
| GIP-DML-SF-005 | PASS | GIP-DML-SF-005 | 337 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-005'] |
| GIP-DML-SF-006 | PASS | GIP-DML-SF-006 | 163 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-006'] |
| GIP-DML-SF-007 | PASS | GIP-DML-SF-007 | 159 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-007'] |
| GIP-DML-SF-008 | PASS | GIP-DML-SF-008 | 159 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-008'] |
| GIP-DML-SF-009 | PASS | GIP-DML-SF-009 | 160 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-009'] |
| GIP-DML-SF-010 | PASS | GIP-DML-SF-010 | 159 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-SF-010'] |
| GIP-DML-IM-001 | MANUAL | GIP-DML-IM-001 | 0 | manual verification | Requires EXPLAIN permission/failure injection |
| GIP-DML-IM-002 | PASS | GIP-DML-IM-002 | 169 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-IM-002'] |
| GIP-DML-IM-003 | PASS | GIP-DML-IM-003 | 243 | level=2, stage=CHECKED | max_level=2, stages=['CHECKED'], rows=1; observed_rules=['GIP-DML-IM-003'] |
| GIP-EXE-RN-001 | MANUAL | GIP-EXE-RN-001 | 0 | manual verification | Requires execute-mode duplicate key/lock/deadlock scenario |
| GIP-BAK-BL-001 | MANUAL | GIP-BAK-BL-001 | 0 | manual verification | Requires changing ROW Binlog prerequisites or replication permissions |
| GIP-BAK-RN-001 | MANUAL | GIP-BAK-RN-001 | 0 | manual verification | Requires backup store/binlog capture fault injection |
