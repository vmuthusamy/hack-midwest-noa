"""Allowlisted inference measurements; never persist prompts or provider error text."""
import contextvars
import json
import math
import os
import re
import threading
import uuid
from collections import Counter
from datetime import datetime, timedelta, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CURRENT_MEASUREMENTS = contextvars.ContextVar("inference_measurements", default=None)
OUTCOMES = {"accepted", "rejected", "timeout", "cancelled", "provider_error"}
VALIDATOR_CODES = {
    "markup", "quotes", "unknown_source", "missing_source", "missing_fact",
    "lifecycle_misstatement", "omitted_update", "unauthorized_drafts", "group_coverage",
    "delivery_state_claim", "missing_prior_instruction", "movement_directive",
    "contradictory_instruction", "safety_directive", "output_contract_violation",
    "incomplete_output", "evidence_input_invalid", "provider_unconfigured",
    "request_incomplete", "worker_busy",
}

class InferenceRejected(ValueError):
    """Fixed application reason, with no model/provider text attached."""
    def __init__(self, reason):
        self.reason = reason if reason in VALIDATOR_CODES else "output_contract_violation"
        super().__init__(self.reason)

def timestamp():
    return datetime.now(timezone.utc).isoformat()

def parsed_time(value):
    try:
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
        return parsed.astimezone(timezone.utc) if parsed.tzinfo else None
    except (AttributeError, TypeError, ValueError, OverflowError):
        return None

def safe_identifier(value):
    if isinstance(value, str) and re.fullmatch(r"[A-Za-z0-9_.:/-]{1,128}", value) and not value.startswith(("http:", "https:", "sk-")):
        return value
    return None

def number(value, integer=False):
    if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value) or value < 0:
        return None
    if integer and int(value) != value:
        return None
    return int(value) if integer else round(float(value), 3)

def milliseconds(value):
    return None if number(value) is None else number(value * 1000)

def numeric_latencies(values):
    if not isinstance(values, (list, tuple)):
        return None
    return [valid for value in values[:32] if (valid := number(value)) is not None]

def normalize_summary(summary, cycle_durations=None):
    """Select numeric SDK fields. Traces, tool input_params and responses are discarded."""
    summary = summary if isinstance(summary, dict) else {}
    usage = summary.get("accumulated_usage") or {}
    usage = usage if isinstance(usage, dict) else {}
    # SDK initializes these counters to zero even when provider usage is absent.
    # A nonempty model request cannot prove zero tokens from that default.
    if all(usage.get(key, 0) == 0 for key in ("inputTokens", "outputTokens", "totalTokens")):
        usage = {}
    tools = []
    tool_usage = summary.get("tool_usage") or {}
    if isinstance(tool_usage, dict):
        for name, raw in sorted(tool_usage.items(), key=lambda item: str(item[0]))[:20]:
            # The current application grants exactly this read-only tool.
            if name != "evidence_reference" or not isinstance(raw, dict):
                continue
            stats = raw.get("execution_stats") or {}
            if not isinstance(stats, dict):
                continue
            duration = number(stats.get("total_time"))
            tools.append({"name": name, "calls": number(stats.get("call_count"), True),
                          "successes": number(stats.get("success_count"), True),
                          "errors": number(stats.get("error_count"), True),
                          "totalTimeMs": None if duration is None else round(duration * 1000, 3)})
    cycles = number(summary.get("total_cycles"), True)
    durations = cycle_durations[:32] if isinstance(cycle_durations, (list, tuple)) else None
    return {"cycles": cycles, "cycleLatencyMs": None if durations is None else [ms for duration in durations if (ms := milliseconds(duration)) is not None],
            "avgCycleLatencyMs": milliseconds(summary.get("average_cycle_time")),
            "totalCycleLatencyMs": milliseconds(summary.get("total_duration")),
            "inputTokens": number(usage.get("inputTokens"), True),
            "outputTokens": number(usage.get("outputTokens"), True),
            "totalTokens": number(usage.get("totalTokens"), True),
            "toolCalls": sum(t["calls"] for t in tools if t["calls"] is not None) if isinstance(summary.get("tool_usage"), dict) else None,
            "tools": tools}

def capture_result_metrics(result):
    collector = CURRENT_MEASUREMENTS.get()
    if collector is None:
        return
    try:
        collector["metrics"] = normalize_summary(result.metrics.get_summary(), getattr(result.metrics, "cycle_durations", None))
    except Exception:
        collector["metrics"] = normalize_summary(None)

def mark_provider_started():
    collector = CURRENT_MEASUREMENTS.get()
    if collector is not None:
        collector["providerStarted"] = True

def validator_code(error):
    # Matching stays in memory; only fixed codes leave this function.
    text = str(error)
    rules = (
        ("tool markup", "markup"), ("quotation", "quotes"),
        ("unknown source", "unknown_source"), ("source reference", "missing_source"),
        ("omitted from changed facts", "missing_fact"), ("cancelled", "lifecycle_misstatement"),
        ("update omitted", "omitted_update"), ("earlier instruction or release holds", "contradictory_instruction"),
        ("drafts are not authorized", "unauthorized_drafts"), ("two affected recipient groups", "group_coverage"),
        ("delivery states", "delivery_state_claim"), ("exact earlier instruction", "missing_prior_instruction"),
        ("direct movement", "movement_directive"), ("contradicts", "contradictory_instruction"),
        ("safety or delivery instruction", "safety_directive"), ("Incomplete model response", "incomplete_output"),
        ("Refresh live evidence", "evidence_input_invalid"), ("recorded review", "evidence_input_invalid"),
        ("recorded review context", "evidence_input_invalid"), ("LIVE and REPLAY", "evidence_input_invalid"),
    )
    for needle, code in rules:
        if needle.lower() in text.lower():
            return code
    return "output_contract_violation"

def percentiles(values):
    values = sorted(v for v in values if number(v) is not None)
    def percentile(p):
        return values[max(0, math.ceil(p * len(values)) - 1)] if values else None
    return {"sampleCount": len(values), "p50": percentile(.50), "p95": percentile(.95), "method": "nearest-rank"}

def normalized_record(record):
    """Reapply the allowlist when loading JSONL; edited files cannot expose extra fields."""
    if not isinstance(record, dict) or record.get("outcome") not in OUTCOMES:
        return None
    metrics = record.get("metrics") or {}
    if not isinstance(metrics, dict):
        metrics = {}
    sanitized_metrics = {key: number(metrics.get(key), True) for key in ("cycles", "inputTokens", "outputTokens", "totalTokens", "toolCalls")}
    sanitized_metrics.update(cycleLatencyMs=numeric_latencies(metrics.get("cycleLatencyMs")),
                             avgCycleLatencyMs=number(metrics.get("avgCycleLatencyMs")),
                             totalCycleLatencyMs=number(metrics.get("totalCycleLatencyMs")))
    sanitized_metrics["tools"] = []
    for tool in metrics.get("tools", [])[:20] if isinstance(metrics.get("tools"), list) else []:
        if isinstance(tool, dict) and tool.get("name") == "evidence_reference":
            sanitized_metrics["tools"].append({"name": "evidence_reference", **{key: number(tool.get(key), key != "totalTimeMs") for key in ("calls", "successes", "errors", "totalTimeMs")}})
    def safe_time(value):
        try:
            parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
            return parsed.isoformat() if parsed.tzinfo else None
        except (AttributeError, TypeError, ValueError):
            return None
    reasons = record.get("validatorReasons") or []
    return {"id": safe_identifier(record.get("id")), "startedAt": safe_time(record.get("startedAt")),
            "finishedAt": safe_time(record.get("finishedAt")), "mode": record.get("mode") if record.get("mode") in {"LIVE", "REPLAY"} else "UNKNOWN",
            "facilityId": safe_identifier(record.get("facilityId")), "outcome": record["outcome"],
            "latencyMs": number(record.get("latencyMs")), "modelId": safe_identifier(record.get("modelId")),
            "providerStarted": record.get("providerStarted") is True,
            "validatorReasons": sorted({r for r in reasons if isinstance(r, str) and r in VALIDATOR_CODES}) if isinstance(reasons, list) else [],
            "metrics": sanitized_metrics}

class MetricsStore:
    def __init__(self, path=None, evaluation_directory=None, memory_only=False):
        self.path = None if memory_only else Path(path or os.getenv("WEATHER_DESK_METRICS_PATH", str(ROOT / ".cache/inference-metrics.jsonl")))
        self.evaluation_directory = Path(evaluation_directory or ROOT / "docs/evaluation-results")
        self.lock = threading.Lock()
        self.records = []
        self.persistence_state = "memory-only" if memory_only else "ready"
        self.skipped_lines = 0
        if self.path is not None:
            try:
                self.path.parent.mkdir(parents=True, exist_ok=True)
                if self.path.exists():
                    for line in self.path.read_text().splitlines():
                        try:
                            record = normalized_record(json.loads(line))
                            if record is None:
                                self.skipped_lines += 1
                            else:
                                self.records.append(record)
                        except (ValueError, TypeError):
                            self.skipped_lines += 1
            except OSError:
                self.persistence_state = "unavailable"

    def record(self, *, started_at, duration_ms, mode, facility_id, outcome, model_id=None, reasons=(), measurements=None):
        measurements = measurements or {}
        record = normalized_record({"id": str(uuid.uuid4()), "startedAt": started_at, "finishedAt": timestamp(),
                                    "mode": mode, "facilityId": facility_id, "outcome": outcome, "latencyMs": duration_ms,
                                    "modelId": model_id, "validatorReasons": list(reasons),
                                    "providerStarted": measurements.get("providerStarted", False), "metrics": measurements.get("metrics")})
        if record is None:
            return
        with self.lock:
            self.records.append(record)
            if self.path is not None:
                try:
                    with self.path.open("a", encoding="utf-8") as output:
                        output.write(json.dumps(record, separators=(",", ":"), allow_nan=False) + "\n")
                        output.flush()
                        os.fsync(output.fileno())
                    self.persistence_state = "ready"
                except OSError:
                    self.persistence_state = "unavailable"

    def snapshot(self, now=None):
        now = now or datetime.now(timezone.utc)
        if now.tzinfo is None:
            raise ValueError("Metrics window requires an explicit timezone")
        now = now.astimezone(timezone.utc)
        window_start = now - timedelta(hours=24)
        with self.lock:
            all_records = list(self.records)
            state, skipped = self.persistence_state, self.skipped_lines
        timed = [(record, parsed_time(record.get("finishedAt"))) for record in all_records]
        unknown_times = sum(finished is None for _, finished in timed)
        future_times = sum(finished is not None and finished > now for _, finished in timed)
        records = [record for record, finished in sorted((item for item in timed if item[1] is not None), key=lambda item: item[1]) if window_start <= finished <= now]
        # Exactly 24 rolling hours, not clock-aligned bins that could require 25.
        # Buckets include their start; only the final bucket includes its end.
        buckets = [{"start": (window_start+timedelta(hours=i)).isoformat(), "end": (window_start+timedelta(hours=i+1)).isoformat(),
                    "accepted": 0, "rejected": 0, "timeout": 0, "cancelled": 0, "providerError": 0} for i in range(24)]
        histogram = [{"minMs": lower, "maxMs": upper, "count": 0} for lower, upper in ((0,5000),(5000,15000),(15000,30000),(30000,60000),(60000,90000),(90000,None))]
        for record in records:
            finished = parsed_time(record["finishedAt"])
            index = min(23, int((finished-window_start).total_seconds() // 3600))
            outcome = "providerError" if record["outcome"] == "provider_error" else record["outcome"]
            buckets[index][outcome] += 1
            latency = number(record.get("latencyMs"))
            if latency is not None:
                for latency_bin in histogram:
                    if latency >= latency_bin["minMs"] and (latency_bin["maxMs"] is None or latency < latency_bin["maxMs"]):
                        latency_bin["count"] += 1
                        break
        counts = Counter(r["outcome"] for r in records)
        measured = [r for r in records if r["metrics"]["totalTokens"] is not None]
        totals = {"calls": len(records), **{key: counts[outcome] for key, outcome in (("accepted", "accepted"), ("rejected", "rejected"), ("timeout", "timeout"), ("cancelled", "cancelled"), ("providerError", "provider_error"))},
                  "guardrailCatches": sum(r["outcome"] == "rejected" and any(reason not in {"provider_unconfigured", "request_incomplete", "worker_busy", "evidence_input_invalid"} for reason in r["validatorReasons"]) for r in records),
                  "providerStarted": sum(r["providerStarted"] for r in records), "callsWithTokenUsage": len(measured)}
        for key in ("inputTokens", "outputTokens", "totalTokens"):
            values = [r["metrics"][key] for r in records if r["metrics"][key] is not None]
            totals[key] = sum(values) if values else None
        return {"schemaVersion": 1, "generatedAt": now.isoformat(),
                "application": {"label": "Application inference requests · last 24 hours", "persistenceState": state,
                                "windowStart": window_start.isoformat(), "windowEnd": now.isoformat(),
                                "excludedUnknownTimestampCount": unknown_times, "excludedFutureTimestampCount": future_times,
                                "unknownLatencyCount": sum(r["latencyMs"] is None for r in records),
                                "outcomeBuckets": buckets, "latencyHistogram": histogram,
                                "skippedJournalLines": skipped, "totals": totals,
                                "latencyMs": percentiles([r["latencyMs"] for r in records]),
                                "recentCalls": list(reversed(records[-20:]))},
                "evaluations": evaluation_snapshot(self.evaluation_directory),
                "notes": ["Application counters begin with this instrumentation; historical app calls are not reconstructed.",
                          "Accepted means the application validator returned an output; it is not a truth or safety certification.",
                          "Evaluation group coverage is not a model-quality pass. Evaluation runs are separate from application calls.",
                          "Absent token/cycle measurements remain unknown. Provider failures may consume unreported tokens.",
                          "Application totals, percentiles and charts use every finished call in the stated 24-hour window; recent calls show only the latest 20.",
                          "Hourly buckets are anchored to the rolling window start. Bucket starts are inclusive; only the final bucket includes its end.",
                          "Records with unknown or future completion times are excluded from the time window and counted separately.",
                          "Prompts, source bodies, generated messages, credentials and raw provider errors are not retained."]}

def evaluation_snapshot(directory):
    runs, latencies, examples = [], [], []
    for path in sorted(directory.glob("model*.json")):
        try:
            rows = json.loads(path.read_text())
        except (OSError, ValueError):
            continue
        if not isinstance(rows, list):
            continue
        rows = [row for row in rows if isinstance(row, dict) and isinstance(row.get("case"), str)]
        times = [round(number(row["elapsed_seconds"]) * 1000, 3) for row in rows if number(row.get("elapsed_seconds")) is not None]
        latencies.extend(times)
        returned = sum(isinstance(row.get("output"), dict) for row in rows)
        catches = sum(not isinstance(row.get("output"), dict) and bool(row.get("validation_reasons")) for row in rows)
        for row in rows:
            reasons = row.get("validation_reasons")
            if not isinstance(row.get("output"), dict) and isinstance(reasons, list) and reasons and len(examples) < 20:
                examples.append({"case": safe_identifier(row["case"]) or "unknown-case", "reasonCodes": sorted({validator_code(reason) for reason in reasons[:20] if isinstance(reason, str)}),
                                 "runFile": safe_identifier(path.name), "latencyMs": milliseconds(row.get("elapsed_seconds"))})
        runs.append({"file": path.name, "samples": len(rows), "returned": returned, "failed": len(rows) - returned,
                     "groupCoveragePasses": sum(row.get("group_coverage_pass") is True for row in rows),
                     "guardrailCatches": catches, "latencyMs": percentiles(times)})
    totals = {key: sum(run[key] for run in runs) for key in ("samples", "returned", "failed", "groupCoveragePasses", "guardrailCatches")}
    return {"label": "Recorded hosted evaluations (separate)", "totals": totals, "latencyMs": percentiles(latencies), "runs": runs,
            "guardrailExamplesLabel": "Recorded evaluation validation catches", "guardrailExamples": examples,
            "tokenUsage": None, "tokenUsageNote": "Token usage was not recorded in historical evaluation artifacts."}
