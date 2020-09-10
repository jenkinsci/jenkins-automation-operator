#! /usr/bin/env bash
TOKEN=$1
JOBID=$2
if [[ $# -ne 2 ]]; then
  echo Usage: $0 token jobid
  exit 1
fi

curl -s -X POST  -H "Content-Type: application/json"  \
                 -H "Accept: application/json" \
                 -H "Travis-API-Version: 3"   \
                 -H "Authorization: token $TOKEN"    \
                 -d '{ "quiet": true }' https://api.travis-ci.org/job/$JOBID/debug  

