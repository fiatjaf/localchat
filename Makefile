localchat: $(shell find . -name "*.go") bindata.go
	go build -ldflags="-s -w" -o ./localchat

bindata.go: static/bundle.js static/index.html static/global.css static/bundle.css static/icon.png
	go-bindata -o bindata.go static/...

static/bundle.js: $(shell find client)
	./node_modules/.bin/rollup -c

deploy: localchat
	ssh root@nusakan-58 'systemctl stop localchat'
	scp localchat nusakan-58:localchat/localchat
	ssh root@nusakan-58 'systemctl start localchat'
