import os
import threading
import time
import pymysql

target_password = os.environ.get("GOINCEPTION_PLUS_TARGET_PASSWORD", "123456")
gateway_port = int(os.environ.get("GOINCEPTION_PLUS_GATEWAY_PORT", "4000"))
target = pymysql.connect(host="127.0.0.1", port=3306, user="root", password=target_password, autocommit=True, charset="utf8mb4")
with target.cursor() as cursor:
    cursor.execute("DROP DATABASE IF EXISTS gip_p56")
    cursor.execute("CREATE DATABASE gip_p56 CHARACTER SET utf8mb4")
    cursor.execute("DROP DATABASE IF EXISTS `127_0_0_1_3306_gip_p56`")
    cursor.execute("""CREATE TABLE gip_p56.t(
        id BIGINT PRIMARY KEY,
        v VARCHAR(32),
        amount DECIMAL(10,2),
        payload VARBINARY(16),
        doc JSON,
        changed_at DATETIME(6)
    )""")
    cursor.execute("""INSERT INTO gip_p56.t VALUES
        (1,'before',10.50,X'0102','{"state":"before"}','2026-08-18 10:00:00.123456')""")

query = f"""/*--host=127.0.0.1;--port=3306;--user=root;--password={target_password};--db=gip_p56;--check=1;--execute=1;--ignore-warnings=1;--backup=1;--sleep=200;--sleep_rows=100;*/
inception_magic_start;
INSERT INTO t VALUES(2,'inserted',20.25,X'0304','{{"state":"inserted"}}','2026-08-18 11:00:00.000001');
UPDATE t SET v='after',amount=99.99,payload=X'FF00',doc='{{"state":"after"}}',changed_at='2026-08-18 12:00:00.999999' WHERE id=1;
DELETE FROM t WHERE id=2;
inception_magic_commit;"""

gateway = pymysql.connect(host="127.0.0.1", port=gateway_port, user="archery", password=os.environ["GOINCEPTION_PLUS_PASSWORD"], charset="utf8mb4")
with gateway.cursor() as cursor:
    # Archery initialization command compatibility.
    cursor.execute("SET NAMES utf8mb4")
    cursor.execute("SET autocommit=1")
    cursor.execute("USE gip_p56")
    cursor.execute("SELECT VERSION()")
    assert "goInception-Plus" in cursor.fetchone()[0]
    cursor.execute("SHOW VARIABLES LIKE 'max_allowed_packet'")
    assert cursor.fetchone()[0] == "max_allowed_packet"
    cursor.execute("SHOW DATABASES")
    assert cursor.fetchall() == (("information_schema",),)
    cursor.execute("SHOW TABLES")
    assert cursor.fetchall() == ()
    cursor.execute("SHOW GRANTS")
    assert "GRANT USAGE" in cursor.fetchone()[0]
    cursor.execute("SELECT CONNECTION_ID()")
    assert cursor.fetchone()[0] >= 10000
    cursor.execute("inception show variables like 'check_dml_where'")
    assert cursor.fetchone()[0] == "check_dml_where"
    cursor.execute("inception set session check_dml_where=1")
    cursor.execute("inception show levels where value=2")
    assert cursor.fetchall()
    cursor.execute("inception get processlist")
    assert len(cursor.description) == 10
    cursor.execute(query)
    assert [column[0] for column in cursor.description] == [
        "order_id", "stage", "error_level", "stage_status", "error_message", "sql",
        "affected_rows", "sequence", "backup_dbname", "execute_time", "sqlsha1", "backup_time",
    ]
    records = cursor.fetchall()
    assert len(records) == 3
    assert all(row[1] == "BACKUP" and row[2] == 0 for row in records), records
    ddl_query = f"""/*--host=127.0.0.1;--port=3306;--user=root;--password={target_password};--db=gip_p56;--check=1;--execute=1;--backup=1;*/
inception_magic_start;
CREATE TABLE ddl_backup_test(id BIGINT PRIMARY KEY);
ALTER TABLE ddl_backup_test ADD COLUMN note VARCHAR(32);
inception_magic_commit;"""
    cursor.execute(ddl_query)
    ddl_records = cursor.fetchall()
    assert all(row[1] == "BACKUP" and row[2] <= 1 for row in ddl_records), ddl_records
    try:
        cursor.execute("SELECT 1")
        raise AssertionError("ordinary query should be rejected")
    except pymysql.err.NotSupportedError:
        pass
gateway.close()

# Authentication is mandatory on the protocol gateway.
try:
    pymysql.connect(host="127.0.0.1", port=gateway_port, user="archery", password="definitely-wrong")
    raise AssertionError("wrong gateway password should be rejected")
except pymysql.err.OperationalError as exc:
    assert exc.args[0] == 1045, exc

# Keep another MySQL connection writing the same table while the audited
# connection executes. The backup must contain only the audited connection's
# row event, proving that position range alone is not used as ownership.
stop_noise = threading.Event()
noise_errors = []
def write_noise():
    try:
        noise = pymysql.connect(host="127.0.0.1", port=3306, user="root", password=target_password, autocommit=True)
        with noise.cursor() as c:
            i = 1000
            while not stop_noise.is_set():
                c.execute("INSERT INTO gip_p56.t(id,v) VALUES(%s,'noise')", (i,))
                i += 1
        noise.close()
    except Exception as exc:
        noise_errors.append(exc)

writer = threading.Thread(target=write_noise, daemon=True)
writer.start()
time.sleep(0.05)
isolation_query = f"""/*--host=127.0.0.1;--port=3306;--user=root;--password={target_password};--db=gip_p56;--check=1;--execute=1;--backup=1;*/
inception_magic_start;
UPDATE t SET v='isolated' WHERE id=1 AND SLEEP(0.20)=0;
inception_magic_commit;"""
isolation_gateway = pymysql.connect(host="127.0.0.1", port=gateway_port, user="archery", password=os.environ["GOINCEPTION_PLUS_PASSWORD"])
with isolation_gateway.cursor() as cursor:
    cursor.execute(isolation_query)
    isolation_record = cursor.fetchone()
isolation_gateway.close()
stop_noise.set()
writer.join(timeout=2)
assert not noise_errors, noise_errors
assert isolation_record[1] == "BACKUP" and isolation_record[2] == 0, isolation_record
with target.cursor() as cursor:
    cursor.execute(
        "SELECT rollback_statement FROM `127_0_0_1_3306_gip_p56`.`t` WHERE opid_time=%s",
        (isolation_record[7],),
    )
    isolation_rollbacks = cursor.fetchall()
    assert len(isolation_rollbacks) == 1, isolation_rollbacks
    assert "WHERE `id`=1 LIMIT 1" in isolation_rollbacks[0][0], isolation_rollbacks
    cursor.execute(isolation_rollbacks[0][0])
    cursor.execute("DELETE FROM gip_p56.t WHERE id >= 1000")

# Roll back the batch in reverse statement order, and each statement's rows in reverse id order.
with target.cursor() as cursor:
    for record in reversed(records):
        cursor.execute(
            "SELECT rollback_statement FROM `127_0_0_1_3306_gip_p56`.`t` "
            "WHERE opid_time=%s ORDER BY id DESC",
            (record[7],),
        )
        for (rollback_sql,) in cursor.fetchall():
            cursor.execute(rollback_sql)
    cursor.execute("SELECT id,v,amount,HEX(payload),JSON_UNQUOTE(JSON_EXTRACT(doc,'$.state')),changed_at FROM gip_p56.t ORDER BY id")
    restored = cursor.fetchall()
    assert len(restored) == 1
    assert restored[0][0:5] == (1, "before", __import__("decimal").Decimal("10.50"), "0102", "before"), restored
    cursor.execute("SELECT COUNT(*) FROM `127_0_0_1_3306_gip_p56`.`$_$Inception_backup_information$_$`")
    assert cursor.fetchone()[0] == 6
    cursor.execute(
        "SELECT type FROM `127_0_0_1_3306_gip_p56`.`$_$Inception_backup_information$_$` "
        "WHERE opid_time IN (%s,%s) ORDER BY opid_time",
        (ddl_records[0][7], ddl_records[1][7]),
    )
    assert {row[0] for row in cursor.fetchall()} == {"CREATETABLE", "ALTERTABLE"}
    for record in reversed(ddl_records):
        cursor.execute(
            "SELECT rollback_statement FROM `127_0_0_1_3306_gip_p56`.`ddl_backup_test` "
            "WHERE opid_time=%s ORDER BY id DESC",
            (record[7],),
        )
        for (rollback_sql,) in cursor.fetchall():
            cursor.execute(rollback_sql)
    cursor.execute("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='gip_p56' AND TABLE_NAME='ddl_backup_test'")
    assert cursor.fetchone()[0] == 0
target.close()
print("phase5 ROW binlog backup/rollback and phase6 protocol smoke test passed")
