from __future__ import print_function

import json
import os
import subprocess
import sys
from StringIO import StringIO

env = json.loads(open("config.env", "rb").read())

def lambda_handler(event, context):
    p = subprocess.Popen(["./lambda-slack-errors"], stdin=subprocess.PIPE, stdout=sys.stdout, stderr=sys.stderr, env=env)
    p.communicate(json.dumps(event))

if __name__ == "__main__":
    lambda_handler({"Records": [{}]}, None)
