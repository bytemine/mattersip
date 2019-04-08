GO111MODULE := on

all:
	mkdir -p dist
	go build -a -o dist/mattersip
	cp plugin.json dist
	cd dist && tar cvzf mattersip.tar.gz plugin.json mattersip
	rm dist/mattersip dist/plugin.json
