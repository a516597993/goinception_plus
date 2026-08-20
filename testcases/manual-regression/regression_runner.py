#!/usr/bin/env python3
"""Data-driven manual-regression companion for goInception Plus.

The runner never treats an unautomated destructive/environmental scenario as
passed. Such cases are emitted as MANUAL in JSON/CSV/Markdown reports.
"""

from __future__ import annotations

import argparse
import csv
import dataclasses
import datetime as dt
import json
import pathlib
import re
import sys
import time
import traceback
from typing import Any, Iterable

try:
    import pymysql
    from pymysql.constants import CLIENT
except ImportError as exc:  # pragma: no cover - environment guard
    raise SystemExit("PyMySQL is required: python -m pip install PyMySQL") from exc


ROOT = pathlib.Path(__file__).resolve().parent
LEGACY_COLUMNS = [
    "order_id", "stage", "error_level", "stage_status", "error_message", "sql",
    "affected_rows", "sequence", "backup_dbname", "execute_time", "sqlsha1", "backup_time",
]


@dataclasses.dataclass
class Case:
    case_id: str
    title: str
    sql: str | None = None
    expected_level: int | None = None
    expected_stage: str | None = "CHECKED"
    protocol_error: int | None = None
    variables: dict[str, str] = dataclasses.field(default_factory=dict)
    rule_id: str | None = None
    versions: tuple[int, ...] = (5, 8)
    manual_reason: str | None = None
    query_kind: str = "audit"
    verify_sql: str | None = None
    verify_value: Any = None


@dataclasses.dataclass
class Result:
    case_id: str
    title: str
    status: str
    duration_ms: int
    expected: str
    actual: str
    target_version: str
    rule_id: str = ""
    rows: list[list[Any]] = dataclasses.field(default_factory=list)
    columns: list[str] = dataclasses.field(default_factory=list)
    error: str = ""


def read_doc_cases(path: pathlib.Path, pattern: str) -> dict[str, str]:
    text = path.read_text(encoding="utf-8")
    out: dict[str, str] = {}
    for line in text.splitlines():
        m = re.match(pattern, line)
        if m:
            cells = [c.strip() for c in line.strip().strip("|").split("|")]
            if len(cells) >= 2:
                out[m.group(1)] = cells[1].replace("`", "")
    return out


def phase_catalog() -> list[Case]:
    titles = read_doc_cases(ROOT / "PHASE_1_7_CASES.md", r"^\|\s*(P[1-7]-\d{3})\s*\|")
    auto: dict[str, Case] = {}

    def add(case_id: str, **kwargs: Any) -> None:
        auto[case_id] = Case(case_id, titles.get(case_id, case_id), **kwargs)

    # Phase 1: request/parser boundaries.
    add("P1-001", sql="USE gip_manual; SELECT id FROM users WHERE id=1;", expected_level=0)
    add("P1-002", sql="UPDATE users SET age=21 WHERE id=1;", protocol_error=1064, query_kind="missing_commit")
    add("P1-003", sql="SELECT id FROM users WHERE id=1;", protocol_error=1064, query_kind="duplicate_start")
    add("P1-004", sql="SELECT id FROM users WHERE id=1;", protocol_error=1064, query_kind="reversed_markers")
    add("P1-005", sql="SELECT 'inception_magic_commit; still text' AS marker FROM users WHERE id=1;", expected_level=0)
    add("P1-006", sql="/* inception_magic_commit; */ SELECT id FROM users WHERE id=1;", expected_level=0)
    add("P1-007", sql="SELECT id FROM users WHERE id=1;", protocol_error=1064, query_kind="unknown_directive")
    add("P1-008", sql="SELECT id FROM users WHERE id=1;", protocol_error=1064, query_kind="invalid_port")
    add("P1-009", sql="SELECT id FROM users WHERE id=1;", protocol_error=1064, query_kind="backup_without_execute")
    add("P1-010", sql="SELECT 'a; b\\\\c' AS text_value FROM users WHERE id=1; SELECT id FROM users WHERE id=2;", expected_level=0)
    add("P1-011", sql="UPDTE users SET age=1;", expected_level=2)
    add("P1-012", sql="PREPARE s FROM 'SELECT 1';", expected_level=2)
    add("P1-014", sql="USE gip_manual; USE gip_missing; SELECT id FROM users WHERE id=1;", expected_level=2)

    # Phase 2: metadata and snapshot. Relaxed rule configuration in the runner
    # keeps these focused on projection/snapshot behavior.
    add("P2-003", sql="ALTER TABLE users ADD COLUMN p2_no_db INT DEFAULT 0;", expected_level=2, query_kind="no_database")
    add("P2-004", sql="USE gip_missing;", expected_level=2)
    add("P2-005", sql="CREATE TABLE p2_t(id BIGINT PRIMARY KEY, v VARCHAR(16) NOT NULL DEFAULT ''); ALTER TABLE p2_t ADD COLUMN score INT NOT NULL DEFAULT 0; INSERT INTO p2_t(id,v,score) VALUES(1,'n',1);", expected_level=0)
    add("P2-006", sql="CREATE TABLE p2_rename(id BIGINT PRIMARY KEY, score INT NOT NULL DEFAULT 0); RENAME TABLE p2_rename TO p2_renamed; UPDATE p2_renamed SET score=2 WHERE id=1;", expected_level=0)
    add("P2-007", sql="CREATE TABLE p2_drop(id BIGINT PRIMARY KEY); DROP TABLE p2_drop; SELECT id FROM p2_drop;", expected_level=2)
    add("P2-008", sql="CREATE TABLE p2_alter(id BIGINT PRIMARY KEY,v VARCHAR(16) NOT NULL DEFAULT ''); ALTER TABLE p2_alter ADD COLUMN score INT NOT NULL DEFAULT 0, CHANGE COLUMN v name VARCHAR(32) NOT NULL DEFAULT '', ADD KEY idx_name(name); INSERT INTO p2_alter(id,name,score) VALUES(1,'n',1);", expected_level=0)
    add("P2-009", sql="CREATE TABLE p2_like LIKE users;", expected_level=0)
    add("P2-011", sql="CREATE TABLE p2_idx(id BIGINT PRIMARY KEY,v INT NOT NULL DEFAULT 0); CREATE INDEX idx_v ON p2_idx(v); DROP INDEX idx_v ON p2_idx;", expected_level=0)

    # Phase 3 rules.
    add("P3-001", sql="INSERT INTO users(username,age,amount) VALUES('phase3',18,1.00);", expected_level=0)
    add("P3-002", sql="INSERT INTO users VALUES(99,'p3',18,1.00,NULL,NULL,NOW(6));", expected_level=1, variables={"check_insert_field": "true"}, rule_id="GIP-DML-SF-007")
    add("P3-003", sql="INSERT INTO users(username,age) VALUES('p31',1),('p32',2);", expected_level=1, variables={"max_insert_rows": "1"}, rule_id="GIP-DML-IM-003")
    add("P3-004", sql="UPDATE users SET age=1; DELETE FROM orders;", expected_level=2, variables={"check_dml_where": "true"}, rule_id="GIP-DML-SF-001")
    add("P3-005", sql="UPDATE users SET age=1 WHERE id>0 ORDER BY id LIMIT 1;", expected_level=2, variables={"check_dml_limit": "true", "check_dml_orderby": "true"})
    add("P3-006", sql="SELECT u.id FROM users u JOIN orders o;", expected_level=2, rule_id="GIP-DML-SF-005")
    add("P3-007", sql="SELECT * FROM users WHERE id=1;", expected_level=1, variables={"enable_select_star": "false"}, rule_id="GIP-DML-SF-006")
    add("P3-008", sql="SELECT id FROM users ORDER BY RAND();", expected_level=2, rule_id="GIP-DML-SF-008")
    add("P3-009", sql="SELECT id FROM type_matrix WHERE int_col='1';", expected_level=2, variables={"check_implicit_type_conversion": "true"}, rule_id="GIP-DML-SF-010")
    add("P3-010", sql="UPDATE users SET missing_column=1 WHERE id=1;", expected_level=2, rule_id="GIP-DML-MD-001")
    add("P3-011", sql="UPDATE users SET age=age+1 WHERE id IN(1,2);", expected_level=2, variables={"max_update_rows": "1"}, rule_id="GIP-DML-IM-002")
    add("P3-014", sql="UPDATE users u JOIN orders o ON o.user_id=u.id SET u.age=u.age+1,o.status=2 WHERE u.id=1 AND o.id=1;", expected_level=0)
    add("P3-015", sql="DELETE u,o FROM users u JOIN orders o ON o.user_id=u.id WHERE u.id=3;", expected_level=0)
    add("P3-017", sql="WITH c AS (SELECT id FROM users WHERE id=1) SELECT id FROM c;", expected_level=2, versions=(5,))

    # Phase 4 safe execution cases. Setup is restored before suite.
    add("P4-001", sql="INSERT INTO users(id,username,age) VALUES(90,'p4',1); UPDATE users SET age=2 WHERE id=90; DELETE FROM users WHERE id=90;", expected_level=0, expected_stage="EXECUTED", query_kind="execute")
    add("P4-002", sql="INSERT INTO users(id,username,age) VALUES(91,'p4-block',1); UPDATE users SET age=3;", expected_level=2, query_kind="execute", verify_sql="SELECT COUNT(*) FROM gip_manual.users WHERE id=91", verify_value=0)
    add("P4-003", sql="INSERT INTO users(id,username,age) VALUES(1,'duplicate',1); INSERT INTO users(id,username,age) VALUES(92,'must-stop',1);", expected_level=2, expected_stage="EXECUTED", query_kind="execute", verify_sql="SELECT COUNT(*) FROM gip_manual.users WHERE id=92", verify_value=0)
    add("P4-009", sql="UPDATE users SET age=99 WHERE id=1;", expected_level=2, query_kind="execute", manual_reason="Requires an isolated service started with audit_only=true")

    # Phase 5: automate a representative full backup batch. Environmental
    # fault injection remains MANUAL.
    add("P5-002", sql="INSERT INTO users(id,username,age,amount,payload,profile) VALUES(95,'backup-insert',1,1.25,X'0102','{\"a\":1}');", expected_level=0, expected_stage="BACKUP", query_kind="backup")
    add("P5-003", sql="UPDATE users SET age=21,amount=11.25 WHERE id=1;", expected_level=0, expected_stage="BACKUP", query_kind="backup")
    add("P5-004", sql="DELETE FROM users WHERE id=2;", expected_level=0, expected_stage="BACKUP", query_kind="backup")
    add("P5-006", sql="UPDATE no_key SET value_num=3 WHERE value_text='a';", expected_level=2, query_kind="backup", rule_id="GIP-DML-SF-002")
    add("P5-007", sql="UPDATE users SET payload=X'00FF10',amount=999999.99,profile=JSON_OBJECT('text','after'),created_at='2026-08-20 12:34:56.123456' WHERE id=1;", expected_level=0, expected_stage="BACKUP", query_kind="backup")
    add("P5-009", sql="UPDATE users u JOIN orders o ON o.user_id=u.id SET u.age=u.age+1,o.status=2 WHERE u.id=1 AND o.id=1;", expected_level=0, expected_stage="BACKUP", query_kind="backup")

    # Phase 6 management/protocol cases.
    add("P6-004", sql="SELECT VERSION()", expected_level=0, query_kind="management")
    add("P6-005", sql="SHOW DATABASES", expected_level=0, query_kind="management")
    add("P6-006", sql="SHOW TABLES FROM information_schema", expected_level=0, query_kind="management")
    add("P6-007", sql="SHOW COLUMNS FROM missing_table FROM information_schema", protocol_error=1146, query_kind="management")
    add("P6-009", sql="SHOW GRANTS", expected_level=0, query_kind="management")
    add("P6-010", sql="SELECT CONNECTION_ID()", expected_level=0, query_kind="management")
    add("P6-011", sql="SELECT 1", protocol_error=1235, query_kind="management")
    add("P6-013", sql="SELECT id FROM users WHERE id=1;", expected_level=0)
    add("P6-014", sql="inception show levels where value=2", expected_level=0, query_kind="management")
    add("P6-017", sql="KILL CONNECTION 4294967294", protocol_error=1094, query_kind="management")

    manual_default = "Requires controlled fault injection, a second connection/process, Archery UI/API, observability endpoint, Docker, or capacity measurements; follow PHASE_1_7_CASES.md"
    cases: list[Case] = []
    for case_id, title in titles.items():
        case = auto.get(case_id)
        if case is None:
            case = Case(case_id, title, manual_reason=manual_default)
        cases.append(case)
    return cases


RULE_SQL: dict[str, tuple[str, dict[str, str]]] = {
    "GIP-COR-ST-001": ("PREPARE s FROM 'SELECT 1';", {}),
    "GIP-COR-PS-002": ("UPDTE users SET age=1;", {}),
    "GIP-META-DB-001": ("USE gip_missing;", {}),
    "GIP-META-DB-002": ("ALTER TABLE users ADD p_no_db INT;", {}),
    "GIP-DDL-MD-001": ("ALTER TABLE missing_t ADD c INT;", {}),
    "GIP-DDL-CT-001": ("CREATE TABLE r001(a INT);", {"check_primary_key": "true"}),
    "GIP-DDL-CT-002": ("CREATE TABLE r002(id BIGINT PRIMARY KEY);", {"check_table_comment": "true"}),
    "GIP-DDL-CT-003": ("CREATE TABLE r003(id BIGINT PRIMARY KEY) COMMENT='r003';", {"check_column_comment": "true"}),
    "GIP-DDL-CT-004": ("CREATE TABLE r004(a INT,b INT,c INT);", {"max_column_count": "2"}),
    "GIP-DDL-CT-005": ("CREATE TABLE r005(id INT) ENGINE=MyISAM;", {"required_engine": "innodb", "enable_set_engine": "true", "support_engine": "innodb,myisam"}),
    "GIP-DDL-CT-006": ("CREATE TABLE r006(id INT,id BIGINT);", {}),
    "GIP-DDL-CT-007": ("CREATE TABLE r007(id INT,KEY idx_a(id),KEY idx_a(id));", {}),
    "GIP-DDL-CT-008": ("CREATE TABLE r008(id INT,KEY idx_x(missing));", {}),
    "GIP-DDL-CT-009": ("CREATE TABLE `bad-name`(id INT);", {"check_identifier": "true"}),
    "GIP-DDL-CT-010": ("CREATE TABLE r010(`select` INT);", {"enable_identifer_keyword": "true"}),
    "GIP-DDL-CT-011": ("CREATE TABLE r011(id INT,KEY bad_name(id));", {"check_index_prefix": "true", "index_prefix": "idx_"}),
    "GIP-DDL-CT-012": ("CREATE TABLE r012(id INT,UNIQUE KEY bad_u(id));", {"check_index_prefix": "true", "uniq_index_prefix": "uniq_"}),
    "GIP-DDL-CT-013": ("CREATE TABLE r013(a INT,b INT,c INT,KEY idx_abc(a,b,c));", {"max_key_parts": "2"}),
    "GIP-DDL-CT-014": ("CREATE TABLE r014(a INT,b INT,c INT,PRIMARY KEY(a,b,c));", {"max_primary_key_parts": "2"}),
    "GIP-DDL-CT-015": ("CREATE TABLE r015(a INT,b INT,c INT,KEY idx_a(a),KEY idx_b(b),KEY idx_c(c));", {"max_keys": "2"}),
    "GIP-DDL-CT-016": ("CREATE TABLE r016(id VARCHAR(16) PRIMARY KEY);", {"enable_pk_columns_only_int": "true"}),
    "GIP-DDL-CT-017": ("CREATE TABLE r017(id DECIMAL(10,0) AUTO_INCREMENT PRIMARY KEY);", {"check_autoincrement_datatype": "true"}),
    "GIP-DDL-CT-018": ("CREATE TABLE r018(id BIGINT AUTO_INCREMENT PRIMARY KEY);", {"enable_autoincrement_unsigned": "true"}),
    "GIP-DDL-CT-019": ("CREATE TABLE r019(seq BIGINT AUTO_INCREMENT PRIMARY KEY);", {}),
    "GIP-DDL-CT-020": ("CREATE TABLE r020(id BIGINT AUTO_INCREMENT PRIMARY KEY) AUTO_INCREMENT=100;", {}),
    "GIP-DDL-CT-021": ("CREATE TABLE r021(id BIGINT PRIMARY KEY);", {"must_have_columns": "created_at"}),
    "GIP-DDL-CT-022": ("CREATE TABLE r022(id INT) DEFAULT CHARSET=utf8mb4;", {"enable_set_charset": "false"}),
    "GIP-DDL-CT-023": ("CREATE TABLE r023(id INT) DEFAULT CHARSET=latin1;", {"enable_set_charset": "true", "support_charset": "utf8mb4"}),
    "GIP-DDL-CT-024": ("CREATE TABLE r024(name VARCHAR(20) CHARACTER SET latin1);", {"enable_column_charset": "false"}),
    "GIP-DDL-CT-025": ("CREATE TABLE r025(id INT PRIMARY KEY,t TEXT DEFAULT 'x');", {"enable_blob_type": "true"}),
    "GIP-DDL-CT-026": ("CREATE TABLE r026(id INT PRIMARY KEY,ts TIMESTAMP);", {"enable_timestamp_type": "false"}),
    "GIP-DDL-CT-027": ("CREATE TABLE r027(id INT PRIMARY KEY,doc JSON);", {"enable_json_type": "false"}),
    "GIP-DDL-CT-028": ("CREATE TABLE r028(id INT PRIMARY KEY,name VARCHAR(20) NULL);", {"enable_nullable": "false"}),
    "GIP-DDL-CT-029": ("CREATE TABLE r029(id INT PRIMARY KEY,content TEXT NOT NULL);", {"enable_blob_type": "true"}),
    "GIP-DDL-CT-030": ("ALTER TABLE users ADD COLUMN r030 INT NOT NULL;", {}),
    "GIP-DDL-CT-031": ("ALTER TABLE users ADD COLUMN r031 DATETIME NOT NULL;", {}),
    "GIP-DDL-CT-032": ("CREATE TABLE r032(id INT PRIMARY KEY,code CHAR(16));", {"max_char_length": "8"}),
    "GIP-DDL-CT-033": ("CREATE TABLE r033(id INT PRIMARY KEY,t1 TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,t2 TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP);", {"check_timestamp_count": "true"}),
    "GIP-DDL-CT-034": ("CREATE TABLE r034(id INT PRIMARY KEY,a INT,b INT,KEY idx_a(a),KEY idx_ab(a,b));", {"check_index_column_repeat": "true"}),
    "GIP-DDL-CT-035": ("CREATE TABLE r035(id INT PRIMARY KEY,dt DATETIME DEFAULT '0000-00-00 00:00:00');", {"sql_mode": "TRADITIONAL"}),
    "GIP-DDL-CT-036": ("CREATE TABLE r036(id INT PRIMARY KEY,state ENUM('a','b'));", {"enable_enum_set_bit": "false"}),
    "GIP-DDL-CT-037": ("CREATE TABLE r037(id INT PRIMARY KEY,content TEXT);", {"enable_blob_type": "false"}),
    "GIP-DDL-AT-001": ("ALTER TABLE users ADD c1 INT DEFAULT 0; ALTER TABLE users ADD c2 INT DEFAULT 0;", {}),
    "GIP-DDL-AT-002": ("ALTER TABLE users CHANGE COLUMN age user_age INT NOT NULL DEFAULT 0;", {}),
    "GIP-DDL-AT-003": ("ALTER TABLE users MODIFY age INT NOT NULL DEFAULT 0 FIRST;", {}),
    "GIP-DDL-AT-004": ("ALTER TABLE users MODIFY username VARCHAR(8) NOT NULL DEFAULT '';", {"check_column_type_change": "true"}),
    "GIP-DDL-SF-001": ("DROP DATABASE gip_manual;", {"enable_drop_database": "false"}),
    "GIP-DDL-SF-002": ("DROP TABLE users;", {"enable_drop_table": "false"}),
    "GIP-DDL-SF-003": ("TRUNCATE TABLE users;", {"enable_truncate_table": "false"}),
    "GIP-DDL-SF-004": ("CREATE TABLE r_fk(id BIGINT PRIMARY KEY,user_id BIGINT,FOREIGN KEY(user_id) REFERENCES users(id));", {"enable_foreign_key": "false"}),
    "GIP-DDL-SF-005": ("CREATE TABLE r_part(id BIGINT PRIMARY KEY) PARTITION BY HASH(id) PARTITIONS 2;", {"enable_partition_table": "false"}),
    "GIP-DDL-SF-006": ("ALTER DATABASE gip_manual CHARACTER SET utf8mb4;", {"enable_set_charset": "false"}),
    "GIP-DDL-SF-007": ("ALTER DATABASE gip_manual COLLATE utf8mb4_general_ci;", {"enable_set_collation": "false"}),
    "GIP-DDL-SF-008": ("ALTER TABLE users ENGINE=InnoDB;", {"enable_set_engine": "false"}),
    "GIP-DDL-SF-009": ("CREATE VIEW v_rule AS SELECT id,username FROM users;", {}),
    "GIP-DML-MD-001": ("UPDATE users SET missing_column=1 WHERE id=1;", {}),
    "GIP-DML-SF-001": ("UPDATE users SET age=1;", {"check_dml_where": "true"}),
    "GIP-DML-SF-002": ("UPDATE no_key SET value_num=3 WHERE value_text='a';", {}),
    "GIP-DML-SF-003": ("UPDATE users SET age=1 WHERE id>0 LIMIT 1;", {"check_dml_limit": "true"}),
    "GIP-DML-SF-004": ("UPDATE users SET age=1 WHERE id>0 ORDER BY id LIMIT 1;", {"check_dml_orderby": "true"}),
    "GIP-DML-SF-005": ("SELECT u.id FROM users u JOIN orders o;", {}),
    "GIP-DML-SF-006": ("SELECT * FROM users WHERE id=1;", {"enable_select_star": "false"}),
    "GIP-DML-SF-007": ("INSERT INTO users VALUES(99,'r',1,1.00,NULL,NULL,NOW(6));", {"check_insert_field": "true"}),
    "GIP-DML-SF-008": ("SELECT id FROM users ORDER BY RAND();", {}),
    "GIP-DML-SF-009": ("SELECT id FROM users WHERE id=1 AND 1;", {}),
    "GIP-DML-SF-010": ("SELECT id FROM type_matrix WHERE int_col='1';", {"check_implicit_type_conversion": "true"}),
    "GIP-DML-IM-002": ("UPDATE users SET age=age+1 WHERE id IN(1,2);", {"max_update_rows": "1"}),
    "GIP-DML-IM-003": ("INSERT INTO users(username,age) VALUES('r1',1),('r2',2);", {"max_insert_rows": "1"}),
}


MANUAL_RULE_REASONS = {
    "GIP-COR-PS-001": "Requires a stable parser-warning corpus; no known deterministic warning SQL is currently exposed",
    "GIP-COR-SF-001": "Requires service restart with audit_only=true",
    "GIP-META-CN-001": "Requires invalid credentials/network or permission fault injection",
    "GIP-META-CN-002": "Version-specific case is covered by phase/version suite; run separately against 5.7",
    "GIP-DML-IM-001": "Requires EXPLAIN permission/failure injection",
    "GIP-EXE-RN-001": "Requires execute-mode duplicate key/lock/deadlock scenario",
    "GIP-BAK-BL-001": "Requires changing ROW Binlog prerequisites or replication permissions",
    "GIP-BAK-RN-001": "Requires backup store/binlog capture fault injection",
}


def rule_catalog() -> list[Case]:
    text = (ROOT / "RULE_CASES.md").read_text(encoding="utf-8")
    codes = list(dict.fromkeys(re.findall(r"GIP-[A-Z]+-[A-Z]+-\d{3}", text)))
    cases: list[Case] = []
    for code in codes:
        if code in MANUAL_RULE_REASONS:
            cases.append(Case(code, code, rule_id=code, manual_reason=MANUAL_RULE_REASONS[code]))
        elif code in RULE_SQL:
            sql, variables = RULE_SQL[code]
            if code == "GIP-DML-SF-002":
                kind = "backup"
            elif code == "GIP-META-DB-002":
                kind = "no_database"
            else:
                kind = "audit"
            cases.append(Case(code, code, sql=sql, expected_level=2, rule_id=code, variables=variables, query_kind=kind))
        else:
            cases.append(Case(code, code, rule_id=code, manual_reason="No safe deterministic automation; follow RULE_CASES.md"))
    return cases


class Harness:
    def __init__(self, args: argparse.Namespace):
        self.args = args
        self.target_version = "unknown"
        self.gateway = None
        self.target = None
        self.server_log = args.server_log

    def connect(self) -> None:
        self.target = pymysql.connect(
            host=self.args.target_host, port=self.args.target_port,
            user=self.args.target_user, password=self.args.target_password,
            charset="utf8mb4", autocommit=True,
            client_flag=CLIENT.MULTI_STATEMENTS,
        )
        with self.target.cursor() as cursor:
            cursor.execute("SELECT VERSION()")
            self.target_version = str(cursor.fetchone()[0])
        self.gateway = pymysql.connect(
            host=self.args.gateway_host, port=self.args.gateway_port,
            user=self.args.gateway_user, password=self.args.gateway_password,
            charset="utf8mb4", autocommit=True,
        )

    def close(self) -> None:
        for conn in (self.gateway, self.target):
            if conn:
                try:
                    conn.close()
                except Exception:
                    pass

    def execute_script(self, path: pathlib.Path) -> None:
        script = path.read_text(encoding="utf-8")
        with self.target.cursor() as cursor:
            cursor.execute(script)
            while cursor.nextset():
                pass

    def management(self, sql: str) -> tuple[list[str], list[list[Any]]]:
        with self.gateway.cursor() as cursor:
            cursor.execute(sql)
            columns = [d[0] for d in cursor.description] if cursor.description else []
            rows = [list(row) for row in cursor.fetchall()] if cursor.description else []
            return columns, rows

    def set_relaxed_policy(self) -> None:
        variables = {
            "check_primary_key": "false", "check_table_comment": "false",
            "check_column_comment": "false", "max_column_count": "0",
            "check_identifier": "false", "check_index_prefix": "false",
            "enable_set_charset": "true", "enable_set_collation": "true",
            "enable_set_engine": "true", "support_engine": "innodb,myisam",
            "enable_column_charset": "true", "enable_blob_type": "true",
            "enable_json_type": "true", "enable_nullable": "true",
            "enable_enum_set_bit": "true", "enable_timestamp_type": "true",
            "enable_foreign_key": "true", "enable_partition_table": "true",
            "enable_drop_table": "true", "enable_drop_database": "true",
            "enable_truncate_table": "true", "max_key_parts": "0",
            "max_primary_key_parts": "0", "max_keys": "0",
            "max_update_rows": "0", "max_insert_rows": "0",
            "check_dml_where": "false", "check_dml_limit": "false",
            "check_dml_orderby": "false", "check_insert_field": "false",
            "check_implicit_type_conversion": "false", "enable_select_star": "true",
            "must_have_columns": "", "max_char_length": "0",
        }
        for name, value in variables.items():
            try:
                self.management(f"inception set {name}={quote_management(value)}")
            except Exception:
                # A report should expose a case failure, but optional relaxation
                # must not abort the entire suite on config-version differences.
                pass

    def disable_all_rules(self, cases: Iterable[Case]) -> None:
        for code in sorted({c.rule_id for c in cases if c.rule_id}):
            try:
                self.management(f"inception set level {code}=0")
            except Exception:
                pass

    def request_text(self, case: Case) -> str:
        directives = (
            f"--host={self.args.target_host};--port={self.args.target_port};"
            f"--user={self.args.target_user};--password={self.args.target_password};"
            f"--db=gip_manual;--check=1;"
        )
        if case.query_kind == "execute":
            directives += "--execute=1;--backup=0;--ignore-warnings=1;"
        elif case.query_kind == "backup":
            directives += "--execute=1;--backup=1;--ignore-warnings=1;"
        else:
            directives += "--execute=0;--backup=0;"
        if case.query_kind == "unknown_directive":
            directives += "--not-exists=1;"
        if case.query_kind == "invalid_port":
            directives = directives.replace(f"--port={self.args.target_port}", "--port=70000")
        if case.query_kind == "backup_without_execute":
            directives = directives.replace("--execute=0;--backup=0", "--execute=0;--backup=1")
        header = f"/*{directives}*/\n"
        if case.query_kind == "missing_commit":
            return header + "inception_magic_start;\n" + (case.sql or "")
        if case.query_kind == "duplicate_start":
            return header + "inception_magic_start;\ninception_magic_start;\n" + (case.sql or "") + "\ninception_magic_commit;"
        if case.query_kind == "reversed_markers":
            return header + "inception_magic_commit;\n" + (case.sql or "") + "\ninception_magic_start;"
        if case.query_kind == "no_database":
            header = header.replace("--db=gip_manual;", "")
        return header + "inception_magic_start;\n" + (case.sql or "") + "\ninception_magic_commit;"

    def run_case(self, case: Case, suite: str) -> Result:
        started = time.perf_counter()
        major = 5 if self.target_version.startswith("5.7") else 8 if self.target_version.startswith("8.0") else 0
        expected = expectation(case)
        if case.manual_reason:
            return Result(case.case_id, case.title, "MANUAL", 0, expected, case.manual_reason, self.target_version, case.rule_id or "")
        if major not in case.versions:
            return Result(case.case_id, case.title, "SKIP", 0, expected, f"not applicable to MySQL major {major}", self.target_version, case.rule_id or "")
        try:
            if suite == "rules" and case.rule_id:
                self.management(f"inception set level {case.rule_id}=2")
            for name, value in case.variables.items():
                self.management(f"inception set {name}={quote_management(value)}")

            columns: list[str] = []
            rows: list[list[Any]] = []
            protocol_code = None
            protocol_message = ""
            log_offset = self.server_log.stat().st_size if self.server_log and self.server_log.exists() else 0
            try:
                if case.query_kind == "management":
                    columns, rows = self.management(case.sql or "")
                else:
                    with self.gateway.cursor() as cursor:
                        cursor.execute(self.request_text(case))
                        columns = [d[0] for d in cursor.description] if cursor.description else []
                        rows = [list(row) for row in cursor.fetchall()] if cursor.description else []
            except pymysql.MySQLError as exc:
                protocol_code = int(exc.args[0]) if exc.args else -1
                protocol_message = str(exc)

            observed_rules = self.read_rule_codes(log_offset) if suite == "rules" and case.rule_id else []
            ok, actual = self.evaluate(case, columns, rows, protocol_code, protocol_message)
            if ok and suite == "rules" and case.rule_id and self.server_log:
                rule_ok = case.rule_id in observed_rules
                ok = ok and rule_ok
                actual += f"; observed_rules={observed_rules}"
            if ok and case.verify_sql:
                with self.target.cursor() as cursor:
                    cursor.execute(case.verify_sql)
                    got = cursor.fetchone()[0]
                ok = got == case.verify_value
                actual += f"; verify={got!r}, want={case.verify_value!r}"
            return Result(
                case.case_id, case.title, "PASS" if ok else "FAIL",
                int((time.perf_counter() - started) * 1000), expected, actual,
                self.target_version, case.rule_id or "", sanitize_rows(rows), columns,
                protocol_message if protocol_code is not None else "",
            )
        except Exception as exc:
            return Result(
                case.case_id, case.title, "ERROR",
                int((time.perf_counter() - started) * 1000), expected,
                f"runner exception: {exc}", self.target_version, case.rule_id or "",
                error=traceback.format_exc(),
            )
        finally:
            if suite == "rules" and case.rule_id:
                try:
                    self.management(f"inception set level {case.rule_id}=0")
                except Exception:
                    pass

    def read_rule_codes(self, offset: int) -> list[str]:
        if not self.server_log:
            return []
        deadline = time.time() + 1.0
        text = ""
        while time.time() < deadline:
            try:
                with self.server_log.open("rb") as fh:
                    fh.seek(offset)
                    text = fh.read().decode("utf-8", errors="replace")
            except FileNotFoundError:
                text = ""
            if "audit_completed" in text or "audit_failed" in text:
                break
            time.sleep(0.05)
        codes: list[str] = []
        for line in text.splitlines():
            try:
                item = json.loads(line)
            except json.JSONDecodeError:
                continue
            if item.get("event") != "audit_completed":
                continue
            for code in str(item.get("rule_codes", "")).split(","):
                if code and code not in codes:
                    codes.append(code)
        return codes

    def evaluate(self, case: Case, columns: list[str], rows: list[list[Any]], protocol_code: int | None, protocol_message: str) -> tuple[bool, str]:
        if case.protocol_error is not None:
            return protocol_code == case.protocol_error, f"protocol_error={protocol_code}, message={protocol_message}"
        if protocol_code is not None:
            return False, f"unexpected protocol_error={protocol_code}, message={protocol_message}"
        if case.query_kind == "management":
            return True, f"columns={columns}, rows={len(rows)}"
        if columns != LEGACY_COLUMNS:
            return False, f"columns mismatch: {columns}"
        levels = [int(row[2]) for row in rows]
        stages = [str(row[1]) for row in rows]
        max_level = max(levels, default=0)
        level_ok = case.expected_level is None or max_level == case.expected_level
        stage_ok = case.expected_stage is None or case.expected_stage in stages
        return level_ok and stage_ok, f"max_level={max_level}, stages={stages}, rows={len(rows)}"


def quote_management(value: str) -> str:
    if re.fullmatch(r"-?\d+|true|false|on|off", value, re.I):
        return value
    return "'" + value.replace("'", "''") + "'"


def expectation(case: Case) -> str:
    if case.manual_reason:
        return "manual verification"
    if case.protocol_error is not None:
        return f"protocol error {case.protocol_error}"
    return f"level={case.expected_level}, stage={case.expected_stage}"


def sanitize_rows(rows: list[list[Any]]) -> list[list[Any]]:
    out = []
    for row in rows:
        clean = []
        for value in row:
            if isinstance(value, (bytes, bytearray)):
                clean.append("0x" + bytes(value).hex())
            elif isinstance(value, (dt.datetime, dt.date, dt.time)):
                clean.append(value.isoformat())
            else:
                clean.append(value)
        out.append(clean)
    return out


def write_reports(results: list[Result], directory: pathlib.Path, suite: str, version: str) -> None:
    directory.mkdir(parents=True, exist_ok=True)
    payload = [dataclasses.asdict(r) for r in results]
    (directory / "results.json").write_text(json.dumps(payload, ensure_ascii=False, indent=2, default=str), encoding="utf-8")
    with (directory / "summary.csv").open("w", encoding="utf-8-sig", newline="") as fh:
        writer = csv.writer(fh)
        writer.writerow(["case_id", "title", "status", "duration_ms", "target_version", "rule_id", "expected", "actual", "error"])
        for r in results:
            writer.writerow([r.case_id, r.title, r.status, r.duration_ms, r.target_version, r.rule_id, r.expected, r.actual, r.error])
    counts = {name: sum(r.status == name for r in results) for name in ("PASS", "FAIL", "ERROR", "MANUAL", "SKIP")}
    lines = [
        f"# {suite} regression summary", "", f"- Target: `{version}`",
        f"- Generated: `{dt.datetime.now().astimezone().isoformat()}`",
        "- " + ", ".join(f"{k}={v}" for k, v in counts.items()), "",
        "| Case | Status | Rule | Duration(ms) | Expected | Actual |", "|---|---|---|---:|---|---|",
    ]
    for r in results:
        esc = lambda s: str(s).replace("|", "\\|").replace("\n", " ")
        lines.append(f"| {r.case_id} | {r.status} | {r.rule_id} | {r.duration_ms} | {esc(r.expected)} | {esc(r.actual)} |")
    (directory / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--suite", choices=("phases", "rules"), required=True)
    parser.add_argument("--gateway-host", default="127.0.0.1")
    parser.add_argument("--gateway-port", type=int, default=4400)
    parser.add_argument("--gateway-user", default="archery")
    parser.add_argument("--gateway-password", required=True)
    parser.add_argument("--target-host", default="127.0.0.1")
    parser.add_argument("--target-port", type=int, required=True)
    parser.add_argument("--target-user", default="root")
    parser.add_argument("--target-password", required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--server-log", type=pathlib.Path)
    parser.add_argument("--skip-setup", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    cases = phase_catalog() if args.suite == "phases" else rule_catalog()
    harness = Harness(args)
    try:
        harness.connect()
        if not args.skip_setup:
            harness.execute_script(ROOT / "setup.sql")
        harness.set_relaxed_policy()
        if args.suite == "rules":
            harness.disable_all_rules(cases)
        results = []
        for case in cases:
            result = harness.run_case(case, args.suite)
            results.append(result)
            print(f"[{result.status:6}] {result.case_id} {result.actual}")
        write_reports(results, args.output, args.suite, harness.target_version)
        failed = sum(r.status in ("FAIL", "ERROR") for r in results)
        manual = sum(r.status == "MANUAL" for r in results)
        print(f"Completed: total={len(results)} failed={failed} manual={manual} output={args.output}")
        return 1 if failed else 0
    finally:
        harness.close()


if __name__ == "__main__":
    sys.exit(main())
