package main

import (
	"github.com/bytemine/mattersip/sip"
	"github.com/mattermost/mattermost-server/plugin"
)

func main() {
	plugin.ClientMain(&sip.Sip{})
}
