#!/bin/bash

# TODO(sadovsky): Lint CSS using RECESS.

set -e
set -u

SRC="${HOME}/dev/tadue"
cd $SRC

echo "Removing deps.js..."
rm -f $SRC/js/deps.js

echo "Running gofmt..."
gofmt -w $SRC

echo "Running gjslint..."
ls $SRC/js/*\.js $SRC/misc/html/*\.js|xargs gjslint --nojsdoc --nobeep

echo "Running jshint..."
ls $SRC/js/*\.js $SRC/misc/html/*\.js|xargs jshint

# Write new deps.js.
echo "Running depswriter..."
$SRC/third_party/closure-library/closure/bin/build/depswriter.py --root_with_prefix='js ../../../../js' > $SRC/js/deps.js
