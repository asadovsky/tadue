export SHELL := /bin/bash -euo pipefail
export GOPATH := $(shell pwd)
export PATH := /usr/local/go_appengine:node_modules/.bin:$(PATH)
export PROJPATH := $(shell pwd)

smtpd:
	python -m smtpd -n -c DebuggingServer localhost:1025

serve:
	dev_appserver.py --skip_sdk_update_check=1 --clear_datastore=1 --smtp_host=127.0.0.1 --smtp_port=1025 .

lint:
	tools/lint.sh

.PHONY: smtpd serve lint
