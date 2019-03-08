package main

import (
	"github.com/bytemine/mattermost-plugin-sip/sip"
	"github.com/mattermost/mattermost-server/plugin"
)

func main() {
	plugin.ClientMain(&sip.Sip{})
}
