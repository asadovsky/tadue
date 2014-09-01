#!/bin/bash

# TODO(sadovsky): Lint CSS using RECESS.

set -e
set -u

SRC=$PROJPATH
cd $SRC

echo "Removing deps.js..."
rm -f $SRC/public/js/deps.js

echo "Running gofmt..."
gofmt -w $SRC

FILES=$(find . -name '*.js' \
  -not -name 'deps.js' \
  -not -path '*/third_party/*' -not -path '*/node_modules/*')

echo "Running gjslint..."
echo $FILES | xargs gjslint --nojsdoc --nobeep

echo "Running jshint..."
echo $FILES | xargs jshint

# Write new deps.js.
echo "Running depswriter..."
$SRC/third_party/closure-library/closure/bin/build/depswriter.py --root_with_prefix="public/js ../../../../js" > $SRC/public/js/deps.js
