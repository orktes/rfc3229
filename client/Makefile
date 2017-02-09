OUTPUT_DIR := ../server/example/fileserver/test_files

.DEFAULT_GOAL := all

all: ${OUTPUT_DIR}/index.html ${OUTPUT_DIR}/worker.js
.PHONY: all

${OUTPUT_DIR}/index.html: index.html
	cp $? $@

${OUTPUT_DIR}/worker.js: service_worker/ServiceWorker.js
	./node_modules/.bin/webpack $? $@
