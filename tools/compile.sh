#!/bin/bash

# TODO(sadovsky):
#  - Use ADVANCED_OPTIMIZATIONS (with jQuery externs).
#  - Run tests on the compiled app.
#  - Write a proper Makefile.
#  - Fix Google Analytics script tag.

set -e
set -u

usage="Usage: `basename $0` {dev,local,prod}"

if [ $# -ne 1 ]; then
  echo $usage
  exit 1
fi

v=$1

SRC='/Users/asadovsky/active/dev/tadue'
cd $SRC

config_go="$SRC/misc/config_$v.go.txt"
if [ ! -e $config_go ]; then
  echo $usage
  exit 1
fi

DST=`mktemp -d /tmp/tadue.$v.XXXX`
echo "Made $DST"

echo 'Copying files...'
cp *.yaml $DST/
cp -rf code.google.com misc securecookie static tadue templates $DST/
cp $config_go $DST/tadue/config.go

mkdir $DST/third_party
cp $SRC/third_party/jquery-*.min.js $DST/third_party/

echo 'Compiling JS files...'
mkdir $DST/js

inputs=`ls $SRC/js/*\.js | grep -v deps\.js | sed -e 's|^|--input=|' | tr '\n' ' '`
$SRC/third_party/closure-library/closure/bin/build/closurebuilder.py \
  --root=$SRC/third_party/closure-library/ --root=$SRC/js/ $inputs \
  --output_mode=compiled \
  --compiler_jar=/Users/asadovsky/active/dev/third_party/closure-compiler/compiler.jar \
  --compiler_flags='--compilation_level=SIMPLE_OPTIMIZATIONS' \
  --output_file=$DST/js/tadue.js 2>/dev/null

echo 'Compiling CSS files...'
mkdir $DST/css

for f in `ls $SRC/less/*\.less`; do
  echo "  $f"
  out=`echo $f | sed -e 's|.*/||' -e 's|.less$|.css|'`
  lessc --yui-compress $f > $DST/css/$out
done

echo 'Updating tags in html files...'

# Replace LESS with CSS.
ls $DST/templates/*.html | xargs \
  sed -i '' -e 's|stylesheet/less|stylesheet|' -e 's|/less/\([A-Za-z_\-]*\)\.less|/css/\1.css|'

# Remove all internal, less, and closure JS imports.
ls $DST/templates/*.html | xargs \
  sed -i '' -e '/src="\/js\//d' -e '/src="\/third_party\/less/d' \
  -e '/src="\/third_party\/closure/d' $f

# Create tadue.js and (if prod) ga.js imports in base.html.
sed -i '' -e 's|{{/\* TADUE_JS \*/}}|<script src="/js/tadue.js">|' $DST/templates/base.html
if [ $v = 'prod' ]; then
  sed -i '' -e 's|{{/\* GA_JS \*/}}|<script src="/js/ga.js">|' $DST/templates/base.html
fi

echo 'Updating application name in app.yaml...'
sed -i '' -e "s/tadue-prod/tadue-$v/" $DST/app.yaml

echo 'Success!'
echo $DST
