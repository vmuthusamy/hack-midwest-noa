"""Run this local experiment. No API route sends employee messages or creates cloud resources."""
import json
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from threading import Lock
from urllib.parse import urlparse
from evidence import fetch_evidence
from briefing import configured, generate_briefing

ROOT=Path(__file__).parent
CACHE=None
BRIEFING_LOCK=Lock()

class Handler(SimpleHTTPRequestHandler):
    def json_response(self, code, payload):
        body=json.dumps(payload).encode()
        self.send_response(code); self.send_header("Content-Type","application/json"); self.send_header("Cache-Control","no-store"); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)

    def do_GET(self):
        global CACHE
        path=urlparse(self.path).path
        if path=="/api/evidence":
            try:
                CACHE=fetch_evidence()
                self.json_response(200,{**CACHE,"agentConfigured":configured()})
            except Exception:
                CACHE=None
                self.json_response(502,{"mode":"LIVE","error":"NOAA refresh failed. Current conditions are unknown; the last successful evidence is not an all-clear."})
            return
        if path=="/":
            self.send_response(302); self.send_header("Location","/prototype/index.html"); self.end_headers(); return
        # Only experimental UI and public fixtures can be served. Never expose .env.
        if not self.path.startswith(("/prototype/","/fixtures/")) or ".." in self.path or "%" in self.path:
            self.send_error(404); return
        super().do_GET()

    def do_POST(self):
        # Same-origin only. A local experiment is not production authentication.
        origin=self.headers.get("Origin")
        if origin not in (None,"http://127.0.0.1:8086","http://localhost:8086"):
            self.json_response(403,{"error":"Origin not allowed"}); return
        if self.path!="/api/briefing": self.send_error(404); return
        if CACHE is None: self.json_response(409,{"error":"Refresh NOAA evidence first"}); return
        if not configured(): self.json_response(409,{"error":"Akamai/OpenAI-compatible inference is not configured"}); return
        if not BRIEFING_LOCK.acquire(blocking=False): self.json_response(409,{"error":"A briefing is already running"}); return
        try:
            snapshot=CACHE
            briefing=generate_briefing(snapshot)
            self.json_response(200,{"briefing":briefing,"evidenceRefresh":snapshot["lastSuccessfulRefresh"],"mode":"LIVE","provider":"Strands · configured endpoint"})
        except Exception:
            self.json_response(502,{"error":"Briefing failed validation or inference. Official evidence remains available; no operational action was taken."})
        finally:
            BRIEFING_LOCK.release()

if __name__=="__main__":
    print("Weather Desk experiment: http://127.0.0.1:8086",flush=True)
    ThreadingHTTPServer(("127.0.0.1",8086),partial(Handler,directory=str(ROOT))).serve_forever()
