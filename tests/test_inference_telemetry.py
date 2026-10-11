"""Telemetry tests use isolated journals and stub providers, never paid inference."""
import asyncio
import json
import os
import tempfile
import time
import unittest
from concurrent import futures
from datetime import datetime, timedelta, timezone
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import grpc
from google.protobuf.json_format import MessageToDict

from agent_service.server import EvidenceService, pb, rpc
from inference_telemetry import CURRENT_MEASUREMENTS, InferenceRejected, MetricsStore, capture_result_metrics, normalize_summary, normalized_record, percentiles, timestamp, validator_code

SDK_SUMMARY = {"total_cycles": 2, "accumulated_usage": {"inputTokens": 100, "outputTokens": 40, "totalTokens": 140},
               "tool_usage": {"evidence_reference": {"tool_info": {"input_params": {"secret": "private-prompt"}},
                               "execution_stats": {"call_count": 1, "success_count": 1, "error_count": 0, "total_time": .02}}},
               "traces": [{"prompt": "private-prompt", "api_key": "sk-secret-key"}]}

def proposal_context():
    return {"proposed_posture":"closure", "effective_at":"2026-10-11T08:00:00-04:00", "policy_version":"illustrative-operating-policy-v1",
            "policy_suggestion":"telework", "policy_rationale":"Applicable winter warning supports eligible telework for manager review.", "evidence_adequacy":"Warning supports operating review; local access is unverified."}

class TelemetryMeasurementsTests(unittest.TestCase):
    def window_record(self, finished, latency=100, outcome="accepted", tokens=None):
        return normalized_record({"outcome": outcome, "finishedAt": finished.isoformat() if isinstance(finished, datetime) else finished,
                                  "latencyMs": latency, "metrics": {"totalTokens": tokens}})

    def test_24_hour_charts_and_percentiles_use_every_call_not_recent_twenty(self):
        now = datetime(2026, 10, 11, 2, 37, 19, tzinfo=timezone.utc)
        store = MetricsStore(memory_only=True)
        outcomes = ("accepted", "rejected", "timeout", "cancelled", "provider_error")
        store.records = [self.window_record(now-timedelta(minutes=i+1), i*1000, outcomes[i % 5], 10) for i in range(30)]
        store.records.extend([self.window_record(now-timedelta(hours=25), 999999, tokens=999999),
                              self.window_record(None), self.window_record(now+timedelta(seconds=1))])
        app = store.snapshot(now=now)["application"]
        self.assertEqual(app["windowStart"], (now-timedelta(hours=24)).isoformat())
        self.assertEqual(app["windowEnd"], now.isoformat())
        self.assertEqual(app["totals"]["calls"], 30)
        self.assertEqual(app["totals"]["totalTokens"], 300)
        self.assertEqual(app["latencyMs"], {"sampleCount": 30, "p50": 14000, "p95": 28000, "method": "nearest-rank"})
        self.assertEqual(len(app["recentCalls"]), 20)
        self.assertEqual(app["recentCalls"][0]["latencyMs"], 0)
        self.assertEqual(app["recentCalls"][-1]["latencyMs"], 19000)
        self.assertEqual(len(app["outcomeBuckets"]), 24)
        self.assertEqual(sum(sum(bucket[key] for key in ("accepted", "rejected", "timeout", "cancelled", "providerError")) for bucket in app["outcomeBuckets"]), 30)
        self.assertEqual(sum(bin["count"] for bin in app["latencyHistogram"]), 30)
        self.assertEqual([app["totals"][key] for key in ("accepted", "rejected", "timeout", "cancelled", "providerError")], [6]*5)
        self.assertEqual(app["excludedUnknownTimestampCount"], 1)
        self.assertEqual(app["excludedFutureTimestampCount"], 1)
        self.assertEqual(len(store.records), 33)  # Windowing never deletes persisted history.

    def test_rolling_hour_boundaries_include_exact_window_start_and_end(self):
        now = datetime(2026, 10, 11, 2, 37, 19, tzinfo=timezone.utc)
        start = now-timedelta(hours=24)
        store = MetricsStore(memory_only=True)
        store.records = [self.window_record(start), self.window_record(start+timedelta(hours=1)),
                         self.window_record(now.astimezone(timezone(timedelta(hours=-4)))),
                         self.window_record(start-timedelta(microseconds=1)),
                         self.window_record(now+timedelta(microseconds=1)),
                         self.window_record("not-a-date"), self.window_record("0001-01-01T00:00:00+14:00")]
        app = store.snapshot(now=now)["application"]
        self.assertEqual(app["totals"]["calls"], 3)
        self.assertEqual([app["outcomeBuckets"][i]["accepted"] for i in (0, 1, 23)], [1, 1, 1])
        self.assertEqual(app["outcomeBuckets"][0]["start"], start.isoformat())
        self.assertEqual(app["outcomeBuckets"][1]["start"], app["outcomeBuckets"][0]["end"])
        self.assertEqual(app["outcomeBuckets"][-1]["end"], now.isoformat())
        self.assertEqual(app["excludedUnknownTimestampCount"], 2)
        self.assertEqual(app["excludedFutureTimestampCount"], 1)
        with self.assertRaises(ValueError):
            store.snapshot(now=now.replace(tzinfo=None))

    def test_latency_histogram_edges_and_unknown_values_conserve_samples(self):
        now = datetime(2026, 10, 11, tzinfo=timezone.utc)
        store = MetricsStore(memory_only=True)
        samples = [0, 4999.999, 5000, 14999.999, 15000, 30000, 60000, 90000, 100000, None, -1, float("nan"), True]
        store.records = [self.window_record(now, latency) for latency in samples]
        app = store.snapshot(now=now)["application"]
        self.assertEqual([bin["count"] for bin in app["latencyHistogram"]], [2, 2, 1, 1, 1, 2])
        self.assertIsNone(app["latencyHistogram"][-1]["maxMs"])
        self.assertEqual(app["latencyMs"]["sampleCount"], 9)
        self.assertEqual(app["unknownLatencyCount"], 4)
        self.assertEqual(sum(bin["count"] for bin in app["latencyHistogram"])+app["unknownLatencyCount"], app["totals"]["calls"])

    def test_empty_window_has_no_fabricated_usage_and_keeps_evaluations_separate(self):
        now = datetime(2026, 10, 11, tzinfo=timezone.utc)
        with tempfile.TemporaryDirectory() as temp:
            (Path(temp)/"model-test.json").write_text(json.dumps([{"case":"recorded", "output":{}, "elapsed_seconds":3}]))
            store = MetricsStore(memory_only=True, evaluation_directory=temp)
            store.records = [self.window_record(now-timedelta(days=2), 1000, tokens=400)]
            snapshot = store.snapshot(now=now)
        app = snapshot["application"]
        self.assertEqual(app["totals"]["calls"], 0)
        self.assertIsNone(app["totals"]["totalTokens"])
        self.assertIsNone(app["latencyMs"]["p95"])
        self.assertEqual(app["latencyMs"]["sampleCount"], 0)
        self.assertEqual(sum(bin["count"] for bin in app["latencyHistogram"]), 0)
        self.assertTrue(all(bucket["accepted"] == 0 for bucket in app["outcomeBuckets"]))
        self.assertEqual(snapshot["evaluations"]["totals"]["samples"], 1)
        self.assertEqual(snapshot["evaluations"]["latencyMs"]["p95"], 3000)

    def test_sdk_summary_disallows_tool_inputs_traces_and_unknown_tool_names(self):
        summary = {**SDK_SUMMARY, "tool_usage": {**SDK_SUMMARY["tool_usage"], "sk-secret-key": {"execution_stats": {"call_count": 12}}}}
        metrics = normalize_summary(summary)
        self.assertEqual((metrics["cycles"], metrics["totalTokens"], metrics["toolCalls"]), (2, 140, 1))
        self.assertEqual(metrics["tools"][0]["totalTimeMs"], 20)
        self.assertNotIn("private-prompt", json.dumps(metrics))
        self.assertNotIn("sk-secret-key", json.dumps(metrics))
        self.assertIsNone(normalize_summary(None)["totalTokens"])
        self.assertIsNone(normalize_summary({"total_cycles": 1, "accumulated_usage": {"inputTokens": 0, "outputTokens": 0, "totalTokens": 0}})["totalTokens"])
        self.assertIsNone(normalize_summary({"accumulated_usage": {"totalTokens": float("nan")}})["totalTokens"])

    def test_fixed_validator_codes_never_export_exception_text(self):
        self.assertEqual(validator_code(ValueError("Agent cited an unknown source: sk-secret-key")), "unknown_source")
        self.assertEqual(validator_code(ValueError("provider details sk-secret-key")), "output_contract_violation")
        self.assertEqual(str(InferenceRejected("sk-secret-key")), "output_contract_violation")

    def test_cycle_durations_are_actual_bounded_numeric_milliseconds_and_persist(self):
        collector = {}
        token = CURRENT_MEASUREMENTS.set(collector)
        summary = {**SDK_SUMMARY, "average_cycle_time": 1.750123, "total_duration": 3.500246}
        try:
            capture_result_metrics(SimpleNamespace(metrics=SimpleNamespace(get_summary=lambda: summary, cycle_durations=[1.000123, 2.500123, "private-prompt", float("nan"), -1])))
        finally:
            CURRENT_MEASUREMENTS.reset(token)
        metrics = collector["metrics"]
        self.assertEqual(metrics["cycleLatencyMs"], [1000.123, 2500.123])
        self.assertEqual(metrics["avgCycleLatencyMs"], 1750.123)
        self.assertEqual(metrics["totalCycleLatencyMs"], 3500.246)
        self.assertEqual(len(normalize_summary(summary, [1] * 100)["cycleLatencyMs"]), 32)
        self.assertIsNone(normalize_summary({})["cycleLatencyMs"])
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "calls.jsonl"
            MetricsStore(path).record(started_at=timestamp(), duration_ms=3600, mode="LIVE", facility_id="SC-DEMO-01", outcome="accepted", measurements=collector)
            persisted = MetricsStore(path).snapshot()["application"]["recentCalls"][0]["metrics"]
            self.assertEqual(persisted["cycleLatencyMs"], metrics["cycleLatencyMs"])
            self.assertEqual(persisted["avgCycleLatencyMs"], metrics["avgCycleLatencyMs"])
            self.assertNotIn("private-prompt", json.dumps(persisted))

    def test_journal_persists_and_reapplies_allowlist_on_read(self):
        with tempfile.TemporaryDirectory() as temp:
            journal = Path(temp) / "calls.jsonl"
            store = MetricsStore(journal, Path(temp) / "evaluations")
            store.record(started_at=timestamp(), duration_ms=1234, mode="LIVE", facility_id="SC-DEMO-01", outcome="rejected", model_id="Qwen/model", reasons=["unknown_source", "sk-secret-key"], measurements={"providerStarted": True, "metrics": normalize_summary(SDK_SUMMARY)})
            row = json.loads(journal.read_text())
            row.update(prompt="private-prompt", provider_error="sk-secret-key")
            row["metrics"]["tools"][0]["input"] = "private-prompt"
            journal.write_text(json.dumps(row) + "\nnot-json\n")
            restarted = MetricsStore(journal, Path(temp) / "evaluations").snapshot()
            self.assertEqual(restarted["application"]["totals"]["calls"], 1)
            self.assertEqual(restarted["application"]["totals"]["guardrailCatches"], 1)
            self.assertEqual(restarted["application"]["totals"]["totalTokens"], 140)
            self.assertEqual(restarted["application"]["skippedJournalLines"], 1)
            self.assertNotIn("private-prompt", json.dumps(restarted))
            self.assertNotIn("sk-secret-key", json.dumps(restarted))

    def test_persistence_failure_is_visible_and_does_not_invent_usage(self):
        with tempfile.TemporaryDirectory() as temp:
            store = MetricsStore(Path(temp), Path(temp) / "missing")
            store.record(started_at=timestamp(), duration_ms=10, mode="LIVE", facility_id="SC-DEMO-01", outcome="provider_error")
            app = store.snapshot()["application"]
            self.assertEqual(app["persistenceState"], "unavailable")
            self.assertEqual(app["totals"]["providerError"], 1)
            self.assertIsNone(app["totals"]["totalTokens"])
            self.assertEqual(app["totals"]["callsWithTokenUsage"], 0)

    def test_historical_evaluations_are_separate_and_not_quality_passes(self):
        root = Path(__file__).resolve().parents[1]
        snapshot = MetricsStore(memory_only=True, evaluation_directory=root / "docs/evaluation-results").snapshot()
        self.assertEqual(snapshot["application"]["totals"]["calls"], 0)
        self.assertEqual(snapshot["evaluations"]["totals"], {"samples": 10, "returned": 8, "failed": 2, "groupCoveragePasses": 8, "guardrailCatches": 2})
        self.assertIsNone(snapshot["evaluations"]["tokenUsage"])
        self.assertNotIn("output", snapshot["evaluations"])
        examples = snapshot["evaluations"]["guardrailExamples"]
        self.assertEqual(len(examples), 2)
        self.assertEqual({example["case"]: example["reasonCodes"] for example in examples}, {"injection": ["delivery_state_claim"], "cancel": ["lifecycle_misstatement"]})
        self.assertTrue(all(example["runFile"].startswith("model") and example["latencyMs"] > 0 for example in examples))

    def test_guardrail_examples_never_copy_raw_reasons_outputs_or_prompts(self):
        with tempfile.TemporaryDirectory() as temp:
            rows = [{"case": "injection", "error_type": "ValueError", "validation_reasons": ["Agent cited an unknown source: sk-secret-key private-prompt"], "elapsed_seconds": 1.5, "prompt": "private-prompt"}]
            (Path(temp) / "model-example.json").write_text(json.dumps(rows))
            evaluations = MetricsStore(memory_only=True, evaluation_directory=temp).snapshot()["evaluations"]
            self.assertEqual(evaluations["guardrailExamples"][0]["reasonCodes"], ["unknown_source"])
            self.assertEqual(evaluations["guardrailExamples"][0]["latencyMs"], 1500)
            self.assertNotIn("private-prompt", json.dumps(evaluations))
            self.assertNotIn("sk-secret-key", json.dumps(evaluations))

    def test_percentiles_report_real_sample_size_and_unknown_empty_sample(self):
        self.assertEqual(percentiles([10, 20, 30, 40, 50])["p95"], 50)
        self.assertEqual(percentiles([10, 20, 30, 40, 50])["p50"], 30)
        self.assertEqual(percentiles([])["sampleCount"], 0)
        self.assertIsNone(percentiles([])["p95"])

    def test_invalid_disk_record_cannot_expose_arbitrary_metadata(self):
        self.assertIsNone(normalized_record({"outcome": "sk-secret-key"}))
        record = normalized_record({"outcome": "accepted", "modelId": "https://api.example/key", "startedAt": "sk-secret-key", "metrics": {"totalTokens": -1}})
        self.assertIsNone(record["modelId"])
        self.assertIsNone(record["startedAt"])
        self.assertIsNone(record["metrics"]["totalTokens"])

class TelemetryGeneratorTests(unittest.IsolatedAsyncioTestCase):
    async def test_case_rationale_instructions_and_weather_context_use_actual_generator_path(self):
        from briefing import AGENT_INSTRUCTIONS, generate_briefing_async
        captured={}
        warning={"id":"test-warning", "event":"Flash Flood Warning", "applicability":"applicable", "lifecycle":"active", "severity":"Severe"}
        explanation="The test facility is covered by a recorded Flash Flood Warning. Propose closure at 8:00 AM instead of illustrative-policy telework; facility access conditions have not been verified."
        output={"changed_facts":[{"fact":"Recorded Flash Flood Warning covers the test facility.", "source_id":"test-warning"}],"source_references":["test-warning"],"evidence_gaps":["No prior approval supplied; simulated staff only."],"explanation":explanation,"draft_operational_messages":[]}
        result=SimpleNamespace(stop_reason="end_turn",message={"content":[{"text":json.dumps(output)}]},metrics=SimpleNamespace(get_summary=lambda:SDK_SUMMARY))
        class StubAgent:
            async def invoke_async(self,prompt):
                captured["input"]=json.loads(prompt)
                return result
        def factory(**kwargs):
            captured["instructions"]=kwargs["system_prompt"]
            return StubAgent()
        evidence={"mode":"REPLAY","evidence_only":True,"facility":{"id":"SC-DEMO-02", "name":"Test facility"},"lastSuccessfulRefresh":"2026-10-10T21:07:32Z","alerts":[warning],"gaps":["Facility access conditions unverified."],"case_context":proposal_context()}
        with patch.dict(os.environ,{"AI_BASE_URL":"https://example.invalid/v1","AI_API_KEY":"test-only-key","AI_MODEL_ID":"test-model"}),patch("strands.Agent",side_effect=factory):
            returned=await generate_briefing_async(evidence)
        self.assertEqual(captured["instructions"],AGENT_INSTRUCTIONS)
        self.assertIn("name the facility, describe the material supplied weather facts",captured["instructions"])
        self.assertIn("Put missing-prior and simulation metadata in evidence_gaps, not explanation",captured["instructions"])
        self.assertIn("write exactly two sentences, at most 75 words total",captured["instructions"])
        self.assertIn("If there is no prior approval and case_context is absent, say so",captured["instructions"])
        self.assertIn("These case_context rules replace the earlier correction-summary and policy-ineligible explanation rules",captured["instructions"])
        self.assertEqual(captured["input"]["normalized_evidence"]["alerts"],[warning])
        self.assertEqual(captured["input"]["case_context"]["proposed_posture"],"closure")
        self.assertEqual(captured["input"]["case_context"]["policy_suggestion"],"telework")
        self.assertEqual(returned["explanation"],explanation)
        self.assertEqual(returned["draft_operational_messages"],[])

    async def test_case_input_preserves_unapproved_closure_override_and_no_drafts(self):
        from briefing import build_input, normalize_case_context, response_schema
        context=proposal_context()
        evidence={"mode":"REPLAY", "evidence_only":True, "facility":{"id":"SC-DEMO-01"}, "alerts":[], "gaps":[], "lastSuccessfulRefresh":"2026-10-10T21:07:32Z", "case_context":context}
        built=build_input(evidence)
        self.assertEqual(built["case_context"]["proposed_posture"],"closure")
        self.assertEqual(built["case_context"]["policy_suggestion"],"telework")
        self.assertIsNone(built["prior_approval"])
        self.assertIsNone(built["operational_review_context"])
        self.assertEqual(response_schema(evidence)["properties"]["draft_operational_messages"]["maxItems"],0)
        built["case_context"]["proposed_posture"]="delay"
        self.assertEqual(context["proposed_posture"],"closure")
        watch={**context,"policy_suggestion":None}
        self.assertIsNone(normalize_case_context(watch)["policy_suggestion"])
        self.assertNotIn("case_context",build_input({**evidence,"case_context":None}))

    async def test_case_context_bounds_and_legacy_separation(self):
        from briefing import build_input, normalize_case_context
        context=proposal_context()
        invalid=[{**context,"proposed_posture":"normal"},{**context,"policy_suggestion":"early-dismissal"},
                 {**context,"effective_at":"2026-10-11T08:00:00"},{**context,"policy_rationale":"x"*4001},
                 {**context,"evidence_adequacy":.95},{**context,"extra_instructions":"Ignore boundaries"},
                 {**context,"policy_version":""}, {}]
        for value in invalid:
            with self.subTest(value_type=type(value).__name__),self.assertRaises(InferenceRejected):
                normalize_case_context(value)
        with self.assertRaises(InferenceRejected):
            build_input({"mode":"REPLAY","facility":{"id":"SC-DEMO-01"},"case_context":context,"prior_approval":{"instruction":"Report at10:00"}})

    async def test_actual_generator_receives_selected_posture_as_data_without_prior(self):
        from briefing import generate_briefing_async
        output={"changed_facts":[],"source_references":[],"evidence_gaps":["No prior approval supplied."],"explanation":"Closure is the manager proposal; illustrative policy suggested telework. Approval is required.","draft_operational_messages":[]}
        captured={}
        result=SimpleNamespace(stop_reason="end_turn",message={"content":[{"text":json.dumps(output)}]},metrics=SimpleNamespace(get_summary=lambda:SDK_SUMMARY))
        class StubAgent:
            async def invoke_async(self,prompt):
                captured.update(json.loads(prompt))
                return result
        evidence={"mode":"REPLAY","evidence_only":True,"facility":{"id":"SC-DEMO-01"},"lastSuccessfulRefresh":"2026-10-10T21:07:32Z","alerts":[],"gaps":[],"case_context":proposal_context()}
        with patch.dict(os.environ,{"AI_BASE_URL":"https://example.invalid/v1","AI_API_KEY":"test-only-key","AI_MODEL_ID":"test-model"}),patch("strands.Agent",return_value=StubAgent()):
            returned=await generate_briefing_async(evidence)
        self.assertEqual(captured["case_context"]["proposed_posture"],"closure")
        self.assertEqual(captured["case_context"]["policy_suggestion"],"telework")
        self.assertIsNone(captured["prior_approval"])
        self.assertEqual(returned["draft_operational_messages"],[])

    async def test_evidence_only_recording_requires_no_prior_and_no_drafts(self):
        from briefing import Briefing, build_input, response_schema, validate_output
        evidence = {"mode": "REPLAY", "evidence_only": True, "facility": {"id": "SC-DEMO-01"}, "lastSuccessfulRefresh": "2026-10-10T21:07:32Z", "alerts": [], "gaps": []}
        self.assertIsNone(build_input(evidence)["operational_review_context"])
        self.assertIsNone(build_input(evidence)["prior_approval"])
        self.assertEqual(response_schema(evidence)["properties"]["draft_operational_messages"]["maxItems"], 0)
        output = Briefing(changed_facts=[], source_references=[], evidence_gaps=["No prior approval supplied."], explanation="Recorded evidence only.", draft_operational_messages=[{"group": "arriving", "text": "Report at 10:00."}])
        with self.assertRaises(ValueError):
            validate_output(output, evidence)

    async def test_actual_generator_captures_sdk_metrics_before_validator_rejection(self):
        from briefing import generate_briefing_async
        output = {"changed_facts": [], "source_references": ["unknown-source"], "evidence_gaps": [], "explanation": "Test.", "draft_operational_messages": []}
        result = SimpleNamespace(stop_reason="end_turn", message={"content": [{"text": json.dumps(output)}]}, metrics=SimpleNamespace(get_summary=lambda: SDK_SUMMARY))
        class StubAgent:
            async def invoke_async(self, prompt):
                return result
        collector = {}
        token = CURRENT_MEASUREMENTS.set(collector)
        evidence = {"mode": "LIVE", "facility": {"id": "SC-DEMO-01"}, "lastSuccessfulRefresh": datetime.now(timezone.utc).isoformat(), "alerts": [{"id": "known-source"}], "gaps": []}
        try:
            with patch.dict(os.environ, {"AI_BASE_URL": "https://example.invalid/v1", "AI_API_KEY": "test-only-key", "AI_MODEL_ID": "test-model"}), patch("strands.Agent", return_value=StubAgent()):
                with self.assertRaises(InferenceRejected) as error:
                    await generate_briefing_async(evidence)
            self.assertEqual(error.exception.reason, "unknown_source")
            self.assertEqual(collector["metrics"]["totalTokens"], 140)
            self.assertTrue(collector["providerStarted"])
            self.assertNotIn("private-prompt", json.dumps(collector))
        finally:
            CURRENT_MEASUREMENTS.reset(token)

class TelemetryTransportTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.store = MetricsStore(Path(self.temp.name) / "calls.jsonl", Path(self.temp.name) / "missing-evaluations")
        async def accepted(evidence):
            CURRENT_MEASUREMENTS.get()["providerStarted"] = True
            capture_result_metrics(SimpleNamespace(metrics=SimpleNamespace(get_summary=lambda: SDK_SUMMARY)))
            return {"changed_facts": [], "source_references": [], "evidence_gaps": [], "explanation": "Stub transport output", "draft_operational_messages": []}
        self.service = EvidenceService(accepted, self.store)
        self.server = grpc.server(futures.ThreadPoolExecutor(max_workers=3))
        rpc.add_EvidenceBriefingServiceServicer_to_server(self.service, self.server)
        port = self.server.add_insecure_port("127.0.0.1:0")
        self.server.start()
        self.channel = grpc.insecure_channel("127.0.0.1:" + str(port))
        self.client = rpc.EvidenceBriefingServiceStub(self.channel)
    def tearDown(self):
        self.channel.close()
        self.server.stop(0).wait()
        self.temp.cleanup()
    def request(self):
        return pb.ExplainRequest(request_id="request-1", mode="LIVE", facility=pb.Facility(id="SC-DEMO-01"), evidence_timestamp=timestamp())
    def wait_for_record(self):
        end = time.monotonic() + 2
        while time.monotonic() < end:
            app = self.store.snapshot()["application"]
            if app["totals"]["calls"]:
                return app
            time.sleep(.01)
        self.fail("missing inference outcome record")
    def test_get_metrics_returns_real_call_and_does_not_count_dashboard_reads(self):
        with patch("agent_service.server.configured", return_value=True):
            self.client.Explain(self.request(), timeout=2)
        first = MessageToDict(self.client.GetMetrics(pb.MetricsRequest(), timeout=2).metrics)
        second = MessageToDict(self.client.GetMetrics(pb.MetricsRequest(), timeout=2).metrics)
        self.assertEqual(first["application"]["totals"]["calls"], 1)
        self.assertEqual(second["application"]["totals"]["calls"], 1)
        self.assertEqual(first["application"]["totals"]["accepted"], 1)
        self.assertEqual(first["application"]["recentCalls"][0]["metrics"]["totalTokens"], 140)

    def test_worker_preserves_hazard_end_separately_from_message_expiry(self):
        captured = {}
        async def evidence_only(evidence):
            captured.update(evidence)
            return {"changed_facts": [], "source_references": [], "evidence_gaps": [], "explanation": "Stub evidence response", "draft_operational_messages": []}
        self.service.generator = evidence_only
        request = self.request()
        request.alerts.add(id="winter-warning", event="Winter Storm Warning", onset="2026-10-10T22:13:00Z", ends="2026-10-12T12:00:00Z", expires="2026-10-11T13:15:00Z")
        with patch("agent_service.server.configured", return_value=True):
            self.client.Explain(request, timeout=2)
        self.assertEqual(captured["alerts"][0]["onset"], "2026-10-10T22:13:00Z")
        self.assertEqual(captured["alerts"][0]["ends"], "2026-10-12T12:00:00Z")
        self.assertEqual(captured["alerts"][0]["expires"], "2026-10-11T13:15:00Z")
    def test_early_rejections_are_counted_once_without_guardrail_inflation(self):
        with patch("agent_service.server.configured", return_value=False), self.assertRaises(grpc.RpcError):
            self.client.Explain(self.request(), timeout=2)
        app = self.wait_for_record()
        self.assertEqual(app["totals"]["calls"], 1)
        self.assertEqual(app["totals"]["rejected"], 1)
        self.assertEqual(app["totals"]["guardrailCatches"], 0)
        self.assertEqual(app["recentCalls"][0]["validatorReasons"], ["provider_unconfigured"])
    def test_invalid_request_is_counted_once(self):
        request = self.request()
        request.request_id = ""
        with patch("agent_service.server.configured", return_value=True), self.assertRaises(grpc.RpcError):
            self.client.Explain(request, timeout=2)
        self.assertEqual(self.wait_for_record()["recentCalls"][0]["validatorReasons"], ["request_incomplete"])
    def test_guardrail_and_provider_failures_are_distinct_and_sanitized(self):
        async def rejected(evidence):
            raise InferenceRejected("unknown_source")
        self.service.generator = rejected
        with patch("agent_service.server.configured", return_value=True), self.assertRaises(grpc.RpcError):
            self.client.Explain(self.request(), timeout=2)
        self.assertEqual(self.wait_for_record()["totals"]["guardrailCatches"], 1)
        async def provider_error(evidence):
            raise ValueError("private-prompt sk-secret-key provider response")
        self.service.generator = provider_error
        with patch("agent_service.server.configured", return_value=True), self.assertRaises(grpc.RpcError):
            self.client.Explain(self.request(), timeout=2)
        app = self.store.snapshot()["application"]
        self.assertEqual(app["totals"]["calls"], 2)
        self.assertEqual(app["totals"]["providerError"], 1)
        self.assertNotIn("sk-secret-key", json.dumps(app))
        self.assertNotIn("private-prompt", json.dumps(app))
    def test_deadline_records_timeout_and_releases_worker(self):
        async def slow(evidence):
            await asyncio.sleep(30)
        self.service.generator = slow
        with patch("agent_service.server.configured", return_value=True), self.assertRaises(grpc.RpcError):
            self.client.Explain(self.request(), timeout=.05)
        app = self.wait_for_record()
        self.assertEqual(app["totals"]["calls"], 1)
        self.assertEqual(app["totals"]["timeout"], 1)
        self.assertFalse(self.service.lock.locked())
    def test_caller_cancellation_records_cancelled_not_timeout(self):
        async def slow(evidence):
            await asyncio.sleep(30)
        self.service.generator = slow
        with patch("agent_service.server.configured", return_value=True):
            call = self.client.Explain.future(self.request(), timeout=3)
            end = time.monotonic() + 1
            while not self.service.lock.locked() and time.monotonic() < end:
                time.sleep(.01)
            call.cancel()
            app = self.wait_for_record()
        self.assertEqual(app["totals"]["cancelled"], 1)
        self.assertEqual(app["totals"]["timeout"], 0)
    def test_worker_busy_is_not_a_model_validation_catch(self):
        self.service.lock.acquire()
        try:
            with patch("agent_service.server.configured", return_value=True), self.assertRaises(grpc.RpcError):
                self.client.Explain(self.request(), timeout=2)
        finally:
            self.service.lock.release()
        app = self.wait_for_record()
        self.assertEqual(app["recentCalls"][0]["validatorReasons"], ["worker_busy"])
        self.assertEqual(app["totals"]["guardrailCatches"], 0)
    def test_evidence_only_replay_does_not_require_fabricated_prior_approval(self):
        request = self.request()
        request.mode, request.evidence_only = "REPLAY", True
        with patch("agent_service.server.configured", return_value=True):
            result = self.client.Explain(request, timeout=2)
        self.assertEqual(result.facility_id, "SC-DEMO-01")
        self.assertEqual(list(result.briefing.draft_operational_messages), [])
    def test_case_context_transport_is_bounded_and_evidence_only(self):
        from google.protobuf.json_format import ParseDict
        captured={}
        async def evidence_only(evidence):
            captured.update(evidence)
            return {"changed_facts":[],"source_references":[],"evidence_gaps":[],"explanation":"Stub evidence response","draft_operational_messages":[]}
        self.service.generator=evidence_only
        request=self.request()
        request.mode,request.evidence_only="REPLAY",True
        ParseDict(proposal_context(),request.case_context)
        with patch("agent_service.server.configured",return_value=True):
            self.client.Explain(request,timeout=2)
        self.assertEqual(captured["case_context"],proposal_context())
        request.evidence_only=False
        with patch("agent_service.server.configured",return_value=True),self.assertRaises(grpc.RpcError):
            self.client.Explain(request,timeout=2)
        request.evidence_only=True
        ParseDict({**proposal_context(),"proposed_posture":"normal"},request.case_context)
        with patch("agent_service.server.configured",return_value=True),self.assertRaises(grpc.RpcError):
            self.client.Explain(request,timeout=2)
    def test_correction_replay_strict_context_remains_and_cannot_mix_evidence_only(self):
        request = self.request()
        request.mode = "REPLAY"
        with patch("agent_service.server.configured", return_value=True), self.assertRaises(grpc.RpcError) as error:
            self.client.Explain(request, timeout=2)
        self.assertEqual(error.exception.code(), grpc.StatusCode.FAILED_PRECONDITION)
        request.evidence_only = True
        request.prior_approval.instruction = "Report at 10:00"
        with patch("agent_service.server.configured", return_value=True), self.assertRaises(grpc.RpcError) as error:
            self.client.Explain(request, timeout=2)
        self.assertEqual(error.exception.code(), grpc.StatusCode.FAILED_PRECONDITION)
