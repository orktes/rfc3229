OUTPUT_DIR := ../server/example/fileserver/test_files

.DEFAULT_GOAL := build

build: ${OUTPUT_DIR}/index.html ${OUTPUT_DIR}/worker.js

${OUTPUT_DIR}/index.html: index.html
	cp $? $@

${OUTPUT_DIR}/worker.js: ServiceWorker.js
	./node_modules/.bin/webpack --devtool source-map $? $@
