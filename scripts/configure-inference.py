#!/usr/bin/env python3
"""Save the organizer team key locally; never print it or pass it as a command argument."""
import getpass
import os
from pathlib import Path
root=Path(__file__).resolve().parent.parent
key=getpass.getpass("Akamai workshop team key (hidden): ").strip()
if not key or key=="paste_your_team_key_here":
    raise SystemExit("A team key is required. No changes saved.")
path=root/".env"
values={}
if path.exists():
    for line in path.read_text().splitlines():
        if "=" in line and not line.lstrip().startswith("#"):
            name,value=line.split("=",1);values[name]=value
values.update(AI_BASE_URL="https://ws.akamai-workshops.com/v1",AI_MODEL_ID="hackathon-chat",AI_API_KEY=key)
fd=os.open(path,os.O_WRONLY|os.O_CREAT|os.O_TRUNC,0o600)
os.chmod(path,0o600)
with os.fdopen(fd,"w") as file:
    file.write("# Local organizer inference configuration. Never commit.\n")
    for name,value in values.items(): file.write(name+"="+value+"\n")
print("Saved locally. Refresh the prototype; the agent worker reloads configuration for each request.")
