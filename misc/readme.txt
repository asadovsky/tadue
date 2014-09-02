export PROJPATH=${HOME}/dev/tadue

Compile and deploy:
${PROJPATH}/tools/compile.sh prod
${PROJPATH}/tools/deploy.sh /tmp/tadue.prod.DIR

Add securecookie submodule:
git submodule add git://github.com/gorilla/securecookie.git securecookie
git submodule update --init

Add closure-library:
https://developers.google.com/closure/library/docs/gettingstarted

Add goauth2:
hg clone http://code.google.com/p/goauth2 code.google.com/p/goauth2
