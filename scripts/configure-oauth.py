#!/usr/bin/env python3
"""Read OAuth credentials from a private file and update the existing cloud app."""
import json
import os
import subprocess
import sys
from pathlib import Path
from dotenv import dotenv_values,set_key
root=Path(__file__).resolve().parent.parent
if len(sys.argv)!=2:raise SystemExit('Usage: ../.venv/bin/python scripts/configure-oauth.py /absolute/path/to/private-oauth.env')
source=dotenv_values(Path(sys.argv[1]).expanduser())
private=root/'.cache/deploy.env'
if not private.exists():raise SystemExit('Existing deployment configuration is required')
for key in ('AUTH_GITHUB_ID','AUTH_GITHUB_SECRET'):
 value=source.get(key)
 if not value or '\n' in value:raise SystemExit(key+' is missing or invalid in the credential file')
for key in ('AUTH_GITHUB_ID','AUTH_GITHUB_SECRET'):set_key(str(private),key,source[key])
os.chmod(private,0o600)
state=json.loads((root/'.cache/deploy.json').read_text())
if state['label']!='weather-desk-hack26':raise SystemExit('Unexpected cloud target')
server='root@'+state['ipv4'][0]
ssh=['-i',str(root/'.cache/deploy-ssh'),'-o','UserKnownHostsFile='+str(root/'.cache/deploy-known-hosts')]
subprocess.run(['scp',*ssh,str(private),server+':/tmp/weather-desk-oauth.env'],check=True)
subprocess.run(['ssh',*ssh,server,'chmod 600 /tmp/weather-desk-oauth.env; mv /tmp/weather-desk-oauth.env /opt/weather-desk/deploy/.env; cd /opt/weather-desk/deploy; docker compose config --quiet && docker compose up -d --no-deps web'],check=True)
print('OAuth credentials configured without logging values. Verify GitHub sign-in at https://'+state['hostname'])
