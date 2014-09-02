export PROJPATH := ${HOME}/dev/tadue

start:
	/usr/local/go_appengine/dev_appserver.py --clear_datastore=1 --smtp_host=127.0.0.1 --smtp_port=1025 ${PROJPATH}

smtp:
	python -m smtpd -n -c DebuggingServer localhost:1025

lint:
	${PROJPATH}/tools/lint.sh

.PHONY: start smtp lint
