#!/usr/bin/env python3
"""Update only the already provisioned personal Weather Desk host.
Private environment and SSH keys stay in ignored .cache; no secrets in arguments.
"""
import json
import subprocess
import tarfile
from pathlib import Path
root=Path(__file__).resolve().parent.parent
state=json.loads((root/'.cache/deploy.json').read_text())
if state['label']!='weather-desk-hack26':raise SystemExit('Unexpected deployment target')
server='root@'+state['ipv4'][0]
ssh=['-i',str(root/'.cache/deploy-ssh'),'-o','UserKnownHostsFile='+str(root/'.cache/deploy-known-hosts')]
archive=root/'.cache/application.tar.gz'
names=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','-z'],cwd=root).decode().split('\0')
with tarfile.open(archive,'w:gz') as tar:
 for name in names:
  if name and (root/name).is_file():tar.add(root/name,arcname=name,recursive=False)
subprocess.run(['scp',*ssh,str(archive),server+':/tmp/weather-desk-deploy.tar.gz'],check=True)
subprocess.run(['scp',*ssh,str(root/'.cache/deploy.env'),server+':/tmp/weather-desk-deploy.env'],check=True)
command='mkdir -p /opt/weather-desk; chmod 600 /tmp/weather-desk-deploy.env; tar -xzf /tmp/weather-desk-deploy.tar.gz -C /opt/weather-desk; mv /tmp/weather-desk-deploy.env /opt/weather-desk/deploy/.env; cd /opt/weather-desk/deploy; docker compose config --quiet && docker compose build agent && docker compose build api && docker compose build web && docker compose up -d'
subprocess.run(['ssh',*ssh,server,command],check=True)
print('Updated https://'+state['hostname'])
