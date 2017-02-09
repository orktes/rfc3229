OUTPUT_DIR := ../server/example/fileserver/test_files

.DEFAULT_GOAL := build

build: ${OUTPUT_DIR}/index.html ${OUTPUT_DIR}/worker.js

${OUTPUT_DIR}/index.html: index.html
	cp $? $@

${OUTPUT_DIR}/worker.js: ServiceWorker.inc.js patch/patch.go main.go
	rm -rf build
	mkdir build
	gopherjs build -o build/worker.js main.go
	cat *.inc.js >> build/worker.js
	cp build/worker.js ${OUTPUT_DIR}/worker.js
