all:
	mkdir -p dist
	go build -o dist/bytemine-sip
	cp plugin.json dist
	cd dist && tar cvzf mattermost-plugin-bytemine-sip.tar.gz plugin.json bytemine-sip
	rm dist/bytemine-sip dist/plugin.json
