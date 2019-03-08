# mattermost-plugin-sip

mattermost plugin for integration of our SIP PBX.

## compiling

- install go: https://golang.org/doc/install
- clone this repository.
- cd to the repository and run `make` which builds the plugin binary.
  afterwards it creates a tarball in `dist` which can be uploaded via the mattermost ui.

## usage

### phone setup

this plugin listens for HTTP requests from the phones, signaling their status.
these requests have the following structure:

#### phone status

    https://mattermost.example.com/plugins/net.bytemine.sip/sip/<action>/<user>

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

    https://mattermost.example.com/plugins/net.bytemine.sip/sip/dnd-on/bob

Would signal that user bob went DND.

#### call status

    https://mattermost.example.com/plugins/net.bytemine.sip/sip/<action>/<user>/<number>

where `action` can be one of

    incoming-call
    incoming-conf
    unknown-exten

`user` is again the name of the affected user, `number` being a phone number.

##### Example

    https://mattermost.example.com/plugins/net.bytemine.sip/sip/incoming-call/bob/1234567

Would signal that user bob has an incoming call from 1234567.

### plugin settings

the settings of the plugin can be modified in mattermosts system console:

- Team Name: name of the team to join
- Channel Name: name of channel to post status messages to
- User E-Mail: email address of user to post the status messages as
- Dashboard Secret: secret to use for protecting the dashboard
- Hide Connection Messages: if true don't display off-hook and on-hook messages in status channel
- Number-User Mappings: mapping from numbers to names in the format
  `<number>:<name>[,<number>:<name>]`, for example
  `123:bob,124:alice`

the plugin enables two slash-commands:

- `/sip-dashboard`: shows a link to a HTML page which shows the current status of known clients. this page auto-refreshes.
- `/sip-status`: shows the status of know clients in mattermost

## plugin status

the health of the plugin can be checked using the URL https://mattermost.example.com/plugins/net.bytemine.sip/status

