#!/bin/bash

set -e
set -u

usage="Usage: `basename $0` [dir]"

if [ $# -ne 1 ]; then
  echo $usage
  exit 1
fi

dir=$1

if [ ! -d $dir ]; then
  echo $usage
  exit 1
fi

/usr/local/go_appengine/appcfg.py --noauth_local_webserver --oauth2 update $dir
