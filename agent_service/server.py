"""Local gRPC boundary for the single read-only Strands evidence agent."""
import signal
import asyncio
import threading
from concurrent import futures
from pathlib import Path
import sys

import grpc
from google.protobuf.json_format import MessageToDict, ParseDict

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
sys.path.insert(0, str(Path(__file__).resolve().parent / "generated"))
from briefing import configured, generate_briefing_async
from dotenv import load_dotenv

def reload_configuration():
    load_dotenv(Path(__file__).resolve().parent.parent / ".env", override=True)
import briefing_pb2 as pb
import briefing_pb2_grpc as rpc

class EvidenceService(rpc.EvidenceBriefingServiceServicer):
    def __init__(self, generator=generate_briefing_async):
        self.generator = generator
        self.lock = threading.Lock()

    def GetStatus(self, request, context):
        reload_configuration()
        return pb.StatusResponse(installed=True, configured=configured(), provider="OpenAI-compatible model provider")

    def Explain(self, request, context):
        reload_configuration()
        if not configured():
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "Inference provider is not configured")
        if not request.request_id or request.mode != "LIVE" or not request.facility.id:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "A live facility evidence snapshot is required")
        if not self.lock.acquire(blocking=False):
            context.abort(grpc.StatusCode.RESOURCE_EXHAUSTED, "A briefing is already running")
        try:
            facility = MessageToDict(request.facility, preserving_proto_field_name=True)
            alerts = [MessageToDict(a, preserving_proto_field_name=True) for a in request.alerts]
            for alert in alerts:
                alert["messageType"] = alert.pop("message_type", "")
                alert["matchMethod"] = alert.pop("match_method", "")
            evidence = {"mode":request.mode, "facility":facility, "lastSuccessfulRefresh":request.evidence_timestamp, "alerts":alerts, "gaps":list(request.evidence_gaps)}
            async def invoke():
                loop=asyncio.get_running_loop()
                task=asyncio.create_task(self.generator(evidence))
                def cancel():
                    try: loop.call_soon_threadsafe(task.cancel)
                    except RuntimeError: pass  # Loop already closed after completion.
                if not context.add_callback(cancel):
                    task.cancel()
                remaining=context.time_remaining()
                return await asyncio.wait_for(task, timeout=min(remaining if remaining is not None else 90, 90))
            result = asyncio.run(invoke())
            briefing = ParseDict(result, pb.Briefing(), ignore_unknown_fields=False)
            return pb.ExplainResponse(request_id=request.request_id, evidence_timestamp=request.evidence_timestamp, facility_id=request.facility.id, briefing=briefing)
        except asyncio.CancelledError:
            context.abort(grpc.StatusCode.CANCELLED, "Request cancelled")
        except asyncio.TimeoutError:
            context.abort(grpc.StatusCode.DEADLINE_EXCEEDED, "Inference deadline exceeded")
        except ValueError:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, "Evidence or model output did not satisfy the briefing contract")
        except Exception:
            context.abort(grpc.StatusCode.INTERNAL, "Inference failed; check the provider configuration")
        finally:
            self.lock.release()


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2), options=[("grpc.max_receive_message_length", 2 * 1024 * 1024)])
    rpc.add_EvidenceBriefingServiceServicer_to_server(EvidenceService(), server)
    if not server.add_insecure_port("127.0.0.1:8089"):
        raise RuntimeError("Agent gRPC port is unavailable")
    signal.signal(signal.SIGTERM, lambda *_: server.stop(5))
    signal.signal(signal.SIGINT, lambda *_: server.stop(5))
    server.start()
    print("Strands gRPC worker: 127.0.0.1:8089", flush=True)
    server.wait_for_termination()

if __name__ == "__main__":
    serve()
