all:
	mkdir -p dist
	rm -rf $$HOME/go/src/github.com
	go get github.com/bytemine/mattermost-plugin-sip/sip
	go get github.com/mattermost/mattermost-server/plugin
	go build -a -o dist/bytemine-sip
	cp plugin.json dist
	cd dist && tar cvzf mattermost-plugin-bytemine-sip.tar.gz plugin.json bytemine-sip
	rm dist/bytemine-sip dist/plugin.json
