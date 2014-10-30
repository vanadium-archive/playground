#!/usr/bin/python2.7

"""Playground GCE monitoring script.

This needs to run on a GCE VM with the replica pool service account scope
(https://www.googleapis.com/auth/ndev.cloudman).

You also need to enable preview in gcloud:
$ gcloud components update preview

Then add it to your crontab, e.g.
*/10 * * * * gcloud preview replica-pools --zone us-central1-a replicas --pool playground-pool list|monitor.py
"""

import datetime
import subprocess
import sys
import yaml

DESIRED = 2
MAX_ALIVE_MIN = 60
POOL = 'playground-pool'


def RunCommand(*args):
  cmd = ['gcloud', 'preview', 'replica-pools', '--zone', 'us-central1-a']
  cmd.extend(args)
  subprocess.check_call(cmd)


def ResizePool(size):
  RunCommand('resize', '--new-size', str(size), POOL)


def ShouldRestart(replica):
  if replica['status']['state'] == 'PERMANENTLY_FAILING':
    print 'Replica %s failed: %s' % (
        replica['name'], replica['status']['details'])
    return True
  return IsTooOld(replica)


def IsTooOld(replica):
  start_text = replica['status']['vmStartTime']
  if start_text:
    start = yaml.load(start_text)
    uptime = datetime.datetime.now() - start
    return uptime.seconds > MAX_ALIVE_MIN * 60


def RestartReplica(replica):
  print 'Restarting replica ' + replica['name']
  ResizePool(DESIRED + 1)
  RunCommand('replicas', '--pool', POOL, 'delete', replica['name'])


def MaybeRestartReplica(replica):
  if ShouldRestart(replica):
    RestartReplica(replica)


def main():
  replicas = yaml.load_all(sys.stdin.read())
  for replica in replicas:
    MaybeRestartReplica(replica)


if __name__ == '__main__':
  main()