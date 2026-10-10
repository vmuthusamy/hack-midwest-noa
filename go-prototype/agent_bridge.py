"""Private stdin/stdout bridge from Go to the existing Strands SDK adapter."""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
from briefing import configured, generate_briefing

if "--status" in sys.argv:
    from importlib.util import find_spec
    print(json.dumps({"configured": configured(), "installed": find_spec("strands") is not None}))
else:
    try:
        evidence = json.load(sys.stdin)
        print(json.dumps(generate_briefing(evidence)))
    except Exception:
        # Provider error messages can contain request data; do not expose them.
        sys.exit(1)
