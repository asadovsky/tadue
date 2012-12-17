#!/bin/bash

# TODO(sadovsky): Run lint on CSS somehow.

set -e
set -u

SRC="/Users/asadovsky/active/dev/tadue"

# Update deps.
$SRC/third_party/closure-library/closure/bin/build/depswriter.py --root_with_prefix='js ../../../../js' > $SRC/js/deps.js

# Run gofmt.
ls $SRC/tadue/*\.go $SRC/misc/*\.go\.txt|xargs gofmt -d
#ls $SRC/tadue/*\.go $SRC/misc/*\.go\.txt|xargs gofmt -w

# Run gjslint.
ls $SRC/js/*\.js $SRC/misc/html/*\.js|xargs gjslint --nojsdoc --nobeep
#ls $SRC/js/*\.js $SRC/misc/html/*\.js|xargs fixjsstyle

# Run jslint.
# http://www.jslint.com/lint.html
ls $SRC/js/*\.js $SRC/misc/html/*\.js|xargs jslint --browser --devel --predef $ --predef goog --predef tadue --vars --nomen --sub --color --indent 2 --plusplus --white --terse
