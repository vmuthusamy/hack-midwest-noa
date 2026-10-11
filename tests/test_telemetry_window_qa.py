"""Independent telemetry arithmetic QA: explicit journals, no worker/model calls."""
import json
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path
from inference_telemetry import MetricsStore

NOW = datetime(2026, 10, 11, 12, 17, 30, tzinfo=timezone.utc)
START = NOW - timedelta(hours=24)
OUTCOME_KEYS = {"accepted": "accepted", "rejected": "rejected", "timeout": "timeout", "cancelled": "cancelled", "provider_error": "providerError"}

def call(index, finished, latency, outcome="accepted"):
    return {"id": "qa-window-" + str(index), "startedAt": (NOW - timedelta(days=3)).isoformat(),
            "finishedAt": finished.isoformat() if isinstance(finished, datetime) else finished,
            "mode": "LIVE", "facilityId": "SC-DEMO-01", "outcome": outcome,
            "latencyMs": latency, "providerStarted": True, "metrics": {}, "validatorReasons": []}

class IndependentTelemetryWindowQA(unittest.TestCase):
    def snapshot(self, rows):
        with tempfile.TemporaryDirectory() as directory:
            journal = Path(directory) / "fixture.jsonl"
            journal.write_text("".join(json.dumps(row) + "\n" for row in rows))
            return MetricsStore(journal, Path(directory) / "missing-evaluations").snapshot(now=NOW)

    def test_completion_clock_bounds_and_offset_timestamps(self):
        rows = [call(1, START, 1000), call(2, NOW, 2000, "rejected"),
                call(3, START - timedelta(microseconds=1), 3000), call(4, NOW + timedelta(microseconds=1), 4000),
                call(5, None, 5000), call(6, START.astimezone(timezone(timedelta(hours=-5))), 6000, "cancelled")]
        app = self.snapshot(rows)["application"]
        self.assertEqual(app["windowStart"], START.isoformat())
        self.assertEqual(app["windowEnd"], NOW.isoformat())
        self.assertEqual(app["totals"]["calls"], 3)
        self.assertEqual(app["excludedUnknownTimestampCount"], 1)
        self.assertEqual(app["excludedFutureTimestampCount"], 1)
        self.assertEqual({row["id"] for row in app["recentCalls"]}, {"qa-window-1", "qa-window-2", "qa-window-6"})
        self.assertEqual(len(app["outcomeBuckets"]), 24)
        self.assertEqual(app["outcomeBuckets"][0]["accepted"], 1)
        self.assertEqual(app["outcomeBuckets"][0]["cancelled"], 1)
        self.assertEqual(app["outcomeBuckets"][-1]["rejected"], 1)

    def test_hourly_counts_use_entire_window_beyond_recent_table_cap(self):
        outcomes = list(OUTCOME_KEYS)
        rows = [call(i, START + timedelta(minutes=i * 40), (i + 1) * 1000, outcomes[i % 5]) for i in range(36)]
        app = self.snapshot(rows)["application"]
        self.assertEqual(app["totals"]["calls"], 36)
        self.assertEqual(len(app["recentCalls"]), 20)
        self.assertEqual(app["latencyMs"], {"sampleCount": 36, "p50": 18000, "p95": 35000, "method": "nearest-rank"})
        self.assertEqual(sum(sum(bucket[key] for key in OUTCOME_KEYS.values()) for bucket in app["outcomeBuckets"]), 36)
        for outcome, key in OUTCOME_KEYS.items():
            expected = sum(row["outcome"] == outcome for row in rows)
            self.assertEqual(app["totals"][key], expected)
            self.assertEqual(sum(bucket[key] for bucket in app["outcomeBuckets"]), expected)
        for index, bucket in enumerate(app["outcomeBuckets"]):
            self.assertEqual(datetime.fromisoformat(bucket["start"]), START + timedelta(hours=index))
            self.assertEqual(datetime.fromisoformat(bucket["end"]), START + timedelta(hours=index + 1))
        self.assertEqual(sum(bin_["count"] for bin_ in app["latencyHistogram"]), 36)

    def test_histogram_boundaries_and_unknown_latency_conserve_samples(self):
        values = [0, 4999.999, 5000, 14999.999, 15000, 29999.999, 30000, 59999.999, 60000, 89999.999, 90000, 120000, None, -1, True]
        app = self.snapshot([call(i, NOW - timedelta(minutes=1), value) for i, value in enumerate(values)])["application"]
        bins = app["latencyHistogram"]
        self.assertEqual([(b["minMs"], b["maxMs"]) for b in bins], [(0, 5000), (5000, 15000), (15000, 30000), (30000, 60000), (60000, 90000), (90000, None)])
        self.assertEqual([b["count"] for b in bins], [2] * 6)
        self.assertEqual(app["unknownLatencyCount"], 3)
        self.assertEqual(app["latencyMs"]["sampleCount"], 12)
        self.assertEqual(app["latencyMs"]["p50"], 29999.999)
        self.assertEqual(app["latencyMs"]["p95"], 120000)
        self.assertEqual(sum(b["count"] for b in bins) + app["unknownLatencyCount"], app["totals"]["calls"])
        self.assertIsNone(app["totals"]["totalTokens"])

    def test_empty_or_unmeasured_window_keeps_unknown_percentiles(self):
        for rows in [[], [call(1, NOW, None)], [call(1, START - timedelta(seconds=1), 1234)]]:
            with self.subTest(rows=len(rows)):
                snapshot = self.snapshot(rows)
                app = snapshot["application"]
                self.assertEqual(app["latencyMs"]["sampleCount"], 0)
                self.assertIsNone(app["latencyMs"]["p50"])
                self.assertIsNone(app["latencyMs"]["p95"])
                self.assertEqual(sum(b["count"] for b in app["latencyHistogram"]), 0)
                self.assertEqual(snapshot["evaluations"]["totals"]["samples"], 0)
