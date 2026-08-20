import importlib.util
import pathlib
import sys
import tempfile
import unittest


HERE = pathlib.Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("gip_regression_runner", HERE / "regression_runner.py")
MODULE = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


class CatalogContractTest(unittest.TestCase):
    def test_phase_document_and_runner_are_one_to_one(self):
        cases = MODULE.phase_catalog()
        self.assertEqual(102, len(cases))
        self.assertEqual(len(cases), len({case.case_id for case in cases}))
        self.assertEqual(54, sum(not case.manual_reason for case in cases))

    def test_rule_document_and_runner_cover_catalog(self):
        cases = MODULE.rule_catalog()
        self.assertEqual(76, len(cases))
        self.assertEqual(len(cases), len({case.case_id for case in cases}))
        self.assertEqual(68, sum(not case.manual_reason for case in cases))
        self.assertEqual(set(MODULE.RULE_SQL) | set(MODULE.MANUAL_RULE_REASONS), {case.case_id for case in cases})

    def test_every_automated_case_has_an_assertion(self):
        for case in MODULE.phase_catalog() + MODULE.rule_catalog():
            if case.manual_reason:
                continue
            self.assertIsNotNone(case.sql, case.case_id)
            self.assertTrue(case.protocol_error is not None or case.expected_level is not None, case.case_id)

    def test_report_writes_all_formats(self):
        result = MODULE.Result("P0-001", "sample", "PASS", 1, "level=0", "level=0", "8.0.39")
        with tempfile.TemporaryDirectory() as temp:
            path = pathlib.Path(temp)
            MODULE.write_reports([result], path, "sample", "8.0.39")
            self.assertTrue((path / "results.json").is_file())
            self.assertTrue((path / "summary.csv").is_file())
            self.assertTrue((path / "summary.md").is_file())

    def test_rule_codes_are_read_from_structured_log(self):
        with tempfile.TemporaryDirectory() as temp:
            log = pathlib.Path(temp) / "server.log"
            log.write_text(
                '{"msg":"audit request completed","event":"audit_completed",'
                '"rule_codes":"GIP-DDL-CT-001,GIP-DDL-CT-003"}\n',
                encoding="utf-8",
            )
            args = type("Args", (), {"server_log": log})()
            harness = MODULE.Harness(args)
            self.assertEqual(["GIP-DDL-CT-001", "GIP-DDL-CT-003"], harness.read_rule_codes(0))


if __name__ == "__main__":
    unittest.main()
