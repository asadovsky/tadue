export SHELL := /bin/bash -euo pipefail
export GOPATH := $(shell pwd)
export PATH := /usr/local/go_appengine:node_modules/.bin:$(PATH)
export PROJPATH := $(HOME)/dev/tadue

start:
	dev_appserver.py --clear_datastore=1 --smtp_host=127.0.0.1 --smtp_port=1025 $(PROJPATH)

smtp:
	python -m smtpd -n -c DebuggingServer localhost:1025

lint:
	$(PROJPATH)/tools/lint.sh

.PHONY: start smtp lint
