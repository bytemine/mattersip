# This plugin is no longer maintained and might or might not work with current and/or future versions of Mattermost

# mattersip

Mattermost plugin for integration of our SIP PBX.

## Compiling

- install go: https://golang.org/doc/install
- because this uses go modules, clone this repository to somewhere **not** in $GOPATH.
  See the respective [go command documentation](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more) for details.
- cd to the repository and run `make` which builds the plugin binary.
  afterwards it creates a tarball in `dist` which can be uploaded via the mattermost ui.

## Usage

### Phone setup

This plugin listens for HTTP requests from the phones, signaling their status.
These requests have the following structure:

#### Phone status

    https://mattermost.example.com/plugins/net.bytemine.sip/sip/<action>/<user>[?secret=<secret>]

where `action` can be one of

	dnd-on
	dnd-off
	offhook
	onhook
	paused-on
	paused-off
	login
	logout
	agent-login
	agent-logout
	answering-call

and `user` being the user name.

##### Example

    https://mattermost.example.com/plugins/net.bytemine.sip/sip/dnd-on/bob[?secret=<secret>]

Would signal that user bob went DND.

#### Call status

    https://mattermost.example.com/plugins/net.bytemine.sip/sip/<action>/<user>/<number>[?secret=<secret>]

where `action` can be one of

    incoming-call
    incoming-conf
    unknown-exten

`user` is again the name of the affected user, `number` being a phone number.

##### Example

    https://mattermost.example.com/plugins/net.bytemine.sip/sip/incoming-call/bob/1234567[?secret=<secret>]

Would signal that user bob has an incoming call from 1234567.

### Plugin settings

The settings of the plugin can be modified in mattermosts system console:

- Team Name: name of the team to join
- Channel Name: name of channel to post status messages to
- User E-mail: email address of user to post the status messages as
- Dashboard Secret: secret to use for protecting the dashboard
- Hide Connection Messages: if true don't display off-hook and on-hook messages in status channel
- Number-User Mappings: mapping from numbers to names in the format
  `<number>:<name>[,<number>:<name>]`, for example
  `123:bob,124:alice`

The plugin enables two slash-commands:

- `/sip-dashboard`: Shows a link to a HTML page which shows the current status of known clients. This page auto-refreshes.
- `/sip-status`: Shows the status of known clients in Mattermost.

## Plugin status

Health of the plugin can be checked using the URL `https://mattermost.example.com/plugins/net.bytemine.sip/status[?secret=<secret>]`

