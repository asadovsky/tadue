export PROJPATH=${HOME}/dev/tadue

Tools:
${PROJPATH}/tools/lint.sh
${PROJPATH}/tools/compile.sh prod
${PROJPATH}/tools/deploy.sh /tmp/tadue.prod.DIR

Diff:
git difftool -t tkdiff -y HEAD
git difftool -t opendiff -y HEAD

Run Python SMTP debug server:
python -m smtpd -n -c DebuggingServer localhost:1025

Run dev_appserver:
/usr/local/go_appengine/dev_appserver.py --clear_datastore --smtp_host=127.0.0.1 --smtp_port=1025 ${PROJPATH}

Add securecookie submodule:
git submodule add git://github.com/gorilla/securecookie.git securecookie

Add closure-library:
git clone http://code.google.com/p/closure-library third_party/closure-library
chmod 755 third_party/closure-library/closure/bin/build/*.py
rm -rf third_party/closure-library/.git

Add goauth2:
hg clone http://code.google.com/p/goauth2 code.google.com/p/goauth2
