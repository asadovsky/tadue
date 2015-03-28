#!/bin/bash

# TODO(sadovsky):
#  - Use ADVANCED_OPTIMIZATIONS (with jQuery externs).
#  - Run tests on the compiled app.
#  - Write a proper Makefile.

set -e
set -u

usage="Usage: `basename $0` {local,prod}"

if [ $# -ne 1 ]; then
  echo $usage
  exit 1
fi

v=$1

SRC=$PROJPATH
cd $SRC

config_go="$SRC/app/config_$v.go"
if [ ! -e $config_go ]; then
  echo "Missing file $config_go"
  exit 1
fi

DST=`mktemp -d /tmp/tadue.$v.XXXX`
echo "Made $DST"

echo 'Copying files...'
cp *.yaml $DST/
cp -rf app code.google.com securecookie templates $DST/
mkdir $DST/public
cp -rf public/static $DST/public/
cp $config_go $DST/app/config.go

mkdir $DST/third_party
cp $SRC/third_party/jquery.min.js $DST/third_party/

echo 'Compiling JS files...'
mkdir $DST/public/js

inputs=`ls $SRC/public/js/*.js | grep -v deps.js | sed -e 's|^|--input=|' | tr '\n' ' '`
$SRC/third_party/closure-library/closure/bin/build/closurebuilder.py \
  --root=$SRC/third_party/closure-library/ --root=$SRC/public/js/ $inputs \
  --output_mode=compiled \
  --compiler_jar=$SRC/third_party/closure-compiler/compiler.jar \
  --compiler_flags='--compilation_level=SIMPLE_OPTIMIZATIONS' \
  --output_file=$DST/public/js/tadue.js 2>/dev/null

echo 'Compiling CSS files...'
mkdir $DST/public/css

for f in `ls $SRC/public/css/*\.less`; do
  echo "  $f"
  out=`echo $f | sed -e 's|.*/||' -e 's|.less$|.css|'`
  node_modules/.bin/lessc --yui-compress $f > $DST/public/css/$out
done

echo 'Updating tags in html files...'

# Replace LESS with CSS.
ls $DST/templates/*.html | xargs \
  sed -i '' -e 's|stylesheet/less|stylesheet|' -e 's|/\([A-Za-z_\-]*\)\.less|/\1.css|'

# Remove all internal, less, and closure JS imports.
ls $DST/templates/*.html | xargs \
  sed -i '' -e '/src="\/js\//d' -e '/src="\/third_party\/less/d' \
  -e '/src="\/third_party\/closure/d' $f

# Add tadue.js import to base.html.
sed -i '' -e 's|{{/\* TADUE_JS \*/}}|<script src="/js/tadue.js"></script>|' \
  $DST/templates/base.html

echo 'Updating application name in app.yaml...'
sed -i '' -e "s/tadue-prod/tadue-$v/" $DST/app.yaml

echo 'Success!'
echo $DST
