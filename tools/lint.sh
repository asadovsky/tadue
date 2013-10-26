#!/bin/bash

# TODO(sadovsky): Run lint on CSS somehow.

set -e
set -u

SRC="${HOME}/dev/tadue"
cd $SRC

# Update deps.
echo "Running depswriter..."
$SRC/third_party/closure-library/closure/bin/build/depswriter.py --root_with_prefix='js ../../../../js' > $SRC/js/deps.js

# Run gofmt.
echo "Running gofmt..."
#ls $SRC/tadue/*\.go $SRC/misc/*\.go\.txt|xargs gofmt -d
ls $SRC/tadue/*\.go $SRC/misc/*\.go\.txt|xargs gofmt -w

# Run gjslint.
echo "Running gjslint..."
ls $SRC/js/*\.js $SRC/misc/html/*\.js|xargs gjslint --nojsdoc --nobeep
#ls $SRC/js/*\.js $SRC/misc/html/*\.js|xargs fixjsstyle

# Run jslint.
# http://www.jslint.com/lint.html
echo "Running jslint..."
ls $SRC/js/*\.js $SRC/misc/html/*\.js|xargs jslint --browser --devel --predef $ --predef goog --predef tadue --vars --nomen --sub --color --indent 2 --plusplus --white --terse
