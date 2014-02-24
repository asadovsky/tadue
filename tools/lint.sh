#!/bin/bash

# TODO(sadovsky): Lint CSS using RECESS.

set -e
set -u

SRC=$PROJPATH
cd $SRC

echo "Removing deps.js..."
rm -f $SRC/js/deps.js

echo "Running gofmt..."
gofmt -w $SRC

echo "Running gjslint..."
find . -name '*.js' -not -path '*/third_party/*' \
  -print0 | xargs -0 gjslint --nojsdoc --nobeep

echo "Running jshint..."
find . -name '*.js' -not -path '*/third_party/*' \
  -print0 | xargs -0 jshint

# Write new deps.js.
echo "Running depswriter..."
$SRC/third_party/closure-library/closure/bin/build/depswriter.py --root_with_prefix='js ../../../../js' > $SRC/js/deps.js
