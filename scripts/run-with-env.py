#!/usr/bin/env python3
"""Load private local configuration as data, then replace this process."""
import os
import sys
from pathlib import Path
from dotenv import dotenv_values
root = Path(__file__).resolve().parent.parent
values = dotenv_values(root / '.env')
environment = {**os.environ, **{k: v for k, v in values.items() if v is not None}}
os.execvpe(sys.argv[1], sys.argv[1:], environment)
