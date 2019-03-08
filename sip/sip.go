package sip

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const dashboardHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta http-equiv="refresh" content="5">
  <title>SIP dashboard</title>
</head>
<body>
{{ template "userStati" . }}
</body>
</html>
`

const userStatiHTML = `<table>
<tr>{{range $name, $status := .}}<td>{{$name}}</td>{{end}}</tr>
<tr>{{range $name, $status := .}}<td>{{$status.HTML}}</td>{{end}}</tr>
</table>
`

const userStatiMarkdown = `|{{ range $name, $status := . }}{{ $name }}|{{ end }}
|{{ range $name, $status := . }}---|{{ end }}
|{{ range $name, $status := . }}{{ $status.Icon }}|{{end}}
`

var dashboardHTMLTemp = template.Must(template.New("dashboard").Parse(dashboardHTML))
var userStatiHTMLTemp = template.Must(dashboardHTMLTemp.New("userStati").Parse(userStatiHTML))
var userStatiMarkdownTemp = template.Must(template.New("userStati").Parse(userStatiMarkdown))

var dashboardCommand = &model.Command{
	Trigger:          "sip-dashboard",
	AutoComplete:     true,
	AutoCompleteDesc: "Get link to SIP dashboard",
	Description:      "Request a link to the SIP dashboard",
}

var statusCommand = &model.Command{
	Trigger:          "sip-status",
	AutoComplete:     true,
	AutoCompleteDesc: "Show status of SIP clients",
	Description:      "Show status of SIP clients",
}

//phoneStatus describes the status of a phone/user
type phoneStatus string

const (
	statusUnknown       phoneStatus = ""
	statusDndOn         phoneStatus = "dnd-on"
	statusDndOff        phoneStatus = "dnd-off"
	statusOffhook       phoneStatus = "offhook"
	statusOnhook        phoneStatus = "onhook"
	statusPausedOn      phoneStatus = "paused-on"
	statusPausedOff     phoneStatus = "paused-off"
	statusLogin         phoneStatus = "login"
	statusLogout        phoneStatus = "logout"
	statusAgentLogin    phoneStatus = "agent-login"
	statusAgentLogout   phoneStatus = "agent-logout"
	statusAnsweringCall phoneStatus = "answering-call"
)

func (p phoneStatus) HTML() template.HTML {
	switch p {
	case statusDndOn:
		return template.HTML(fmt.Sprintf("<span style=\"color: red\">%v</span>", p.Icon()))
	case statusOffhook:
		return template.HTML(fmt.Sprintf("<span style=\"color: red\">%v</span>", p.Icon()))
	case statusDndOff, statusOnhook:
		return template.HTML(fmt.Sprintf("<span style=\"color: green\">%v</span>", p.Icon()))
	default:
		return template.HTML(p.Icon())
	}
}

func (p phoneStatus) message(user string) string {
	switch p {
	case statusDndOn:
		return fmt.Sprintf("%v DND on", user)
	case statusDndOff:
		return fmt.Sprintf("%v DND off", user)
	case statusOffhook:
		return fmt.Sprintf("%v connected", user)
	case statusOnhook:
		return fmt.Sprintf("%v disconnected", user)
	case statusPausedOn:
		return fmt.Sprintf("%v paused", user)
	case statusPausedOff:
		return fmt.Sprintf("%v unpaused", user)
	case statusLogin:
		return fmt.Sprintf("%v logged in", user)
	case statusLogout:
		return fmt.Sprintf("%v logged out", user)
	case statusAgentLogin:
		return fmt.Sprintf("%v logged in as agent", user)
	case statusAgentLogout:
		return fmt.Sprintf("%v logged out as agent", user)
	case statusAnsweringCall:
		return fmt.Sprintf("%v answers call", user)
	case statusUnknown:
		fallthrough
	default:
		return fmt.Sprintf("unknown phone status for user: %v", user)
	}
}

func (p phoneStatus) Icon() string {
	switch p {
	case statusDndOn:
		return "🚫"
	case statusDndOff:
		return "✔"
	case statusOffhook:
		return "🕻"
	case statusOnhook:
		return "✔"
	case statusPausedOn:
		return "⏸"
	case statusPausedOff:
		return "✔"
	case statusLogin:
		return "☎←"
	case statusLogout:
		return "☎→"
	case statusAgentLogin:
		return "⛟←"
	case statusAgentLogout:
		return "⛟→"
	case statusAnsweringCall:
		return "⚡"
	case statusUnknown:
		fallthrough
	default:
		return "⁇"
	}
}

//callStatus describes the status of a call
type callStatus string

const (
	statusIncomingCall       callStatus = "incoming-call"
	statusIncomingConference callStatus = "incoming-conf"
	statusUnknownExtension   callStatus = "unknown-exten"
)

func (c callStatus) message(user, number string) string {
	switch c {
	case statusIncomingCall:
		return fmt.Sprintf("incoming call for %v from %v", user, number)
	case statusIncomingConference:
		return fmt.Sprintf("incoming call for %v from %v", user, number)
	case statusUnknownExtension:
		return fmt.Sprintf("incoming call for %v from unknown extension %v", user, number)
	default:
		return fmt.Sprintf("unknown call status for user: %v number: %v", user, number)
	}
}

func (c callStatus) Icon() string {
	switch c {
	case statusIncomingCall:
		return "←⍾"
	case statusIncomingConference:
		return "←⌘"
	case statusUnknownExtension:
		return "←?"
	default:
		return "⁇"
	}
}

type Sip struct {
	plugin.MattermostPlugin

	sipConfig

	status map[string]phoneStatus

	mux *http.ServeMux
}

// sipConfig is used to get the config from the mattermost server
type sipConfig struct {
	TeamName               string
	ChannelName            string
	UserMail               string
	Secret                 string
	HideConnectionMessages bool
	NumbersUsers           string

	// set by OnConfigurationChange
	teamId       string
	channelId    string
	userId       string
	numbersUsers map[string]string
}

func (s *Sip) OnConfigurationChange() error {
	err := s.API.LoadPluginConfiguration(&s.sipConfig)
	if err != nil {
		return err
	}

	// get the channel id
	team, res := s.API.GetTeamByName(s.TeamName)
	if res != nil {
		s.API.LogError("failed to find team", "Error", res.Error, "DetailedError", res.DetailedError)
		return res
	}
	s.teamId = team.Id

	channel, res := s.API.GetChannelByName(s.teamId, s.ChannelName, false)
	if res != nil {
		s.API.LogError("failed to find channel", "channel_name", s.ChannelName)
		return res
	}
	s.channelId = channel.Id

	// get the user id
	user, res := s.API.GetUserByEmail(s.UserMail)
	if res != res {
		s.API.LogError("failed to find user", "user_mail", s.UserMail)
		return res
	}
	s.userId = user.Id

	s.numbersUsers = make(map[string]string)
	for _, v := range strings.Split(s.NumbersUsers, ",") {
		fs := strings.Split(v, ":")
		if len(fs) != 2 {
			s.API.LogError("user number mapping not a pair")
			continue
		}
		s.numbersUsers[fs[0]] = fs[1]
	}

	s.mux = http.NewServeMux()
	s.mux.Handle("/sip/", http.StripPrefix("/sip", http.HandlerFunc(s.handleSip)))
	s.mux.HandleFunc("/status", s.handleStatus)
	s.mux.HandleFunc("/dashboard", s.handleDashboard)

	s.status = make(map[string]phoneStatus)

	s.API.RegisterCommand(dashboardCommand)
	s.API.RegisterCommand(statusCommand)

	return nil
}

func (s *Sip) OnDeactivate() {
	s.API.UnregisterCommand(s.teamId, dashboardCommand.Trigger)
	s.API.UnregisterCommand(s.teamId, statusCommand.Trigger)
}

func (s *Sip) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	switch {
	case strings.Contains(args.Command, dashboardCommand.Trigger):
		return s.executeDashboardCommand(args)
	case strings.Contains(args.Command, statusCommand.Trigger):
		return s.executeStatusCommand(args)
	}

	return nil, &model.AppError{Message: "unknown command"}
}

func (s *Sip) executeDashboardCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	dashboardURL, err := url.Parse(args.SiteURL)
	if err != nil {
		return nil, &model.AppError{Message: err.Error()}
	}
	dashboardURL.Scheme = "https"
	dashboardURL.Path = path.Join(dashboardURL.Path, "plugins", "net.bytemine.sip", "dashboard")
	q := dashboardURL.Query()
	q.Set("secret", s.Secret)
	dashboardURL.RawQuery = q.Encode()

	x := &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf("[SIP Dashboard](%v)", dashboardURL.String()),
	}
	return x, nil
}

func (s *Sip) executeStatusCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	x := &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         s.markdownStatus(),
	}

	return x, nil
}

func (s *Sip) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Sip) handleSip(w http.ResponseWriter, r *http.Request) {
	x := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")

	s.API.LogError(fmt.Sprintf("request url parts: %#v", x))

	switch len(x) {
	case 2: // /action/user
		s.handlePhoneStatus(w, r, x[0], x[1])
	case 3: // /action/user/number
		s.handleCallStatus(w, r, x[0], x[1], x[2])
	default:
		http.Error(w, "malformed request", http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (s *Sip) handlePhoneStatus(w http.ResponseWriter, r *http.Request, action, user string) {
	// use username if number is known
	if username, ok := s.numbersUsers[user]; ok {
		user = username
	}

	status := phoneStatus(action)
	message := status.message(user)
	s.status[user] = status

	if s.HideConnectionMessages && (status == statusOffhook || status == statusOnhook) {
		w.WriteHeader(http.StatusOK)
		return
	}

	outPost := &model.Post{
		UserId:    s.userId,
		ChannelId: s.channelId,
		Message:   message,
		Props:     map[string]interface{}{"sent_by_plugin": true},
	}

	_, err := s.API.CreatePost(outPost)
	if err != nil {
		s.API.LogError(fmt.Sprintf("can't send message to channel: %v", err.Error()))
		http.Error(w, fmt.Sprintf("can't send message to channel: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	s.API.LogInfo("send message to", "channel", s.channelId)

	w.WriteHeader(http.StatusOK)
}

func (s *Sip) handleCallStatus(w http.ResponseWriter, r *http.Request, action, user, number string) {
	// use username if number is known
	if username, ok := s.numbersUsers[user]; ok {
		user = username
	}

	if numbername, ok := s.numbersUsers[number]; ok {
		number = numbername
	}

	status := callStatus(action)
	message := status.message(user, number)
	//s.status[user] = status

	outPost := &model.Post{
		UserId:    s.userId,
		ChannelId: s.channelId,
		Message:   message,
		Props:     map[string]interface{}{"sent_by_plugin": true},
	}

	_, err := s.API.CreatePost(outPost)
	if err != nil {
		s.API.LogError(fmt.Sprintf("can't send message to channel: %v", err.Error()))
		http.Error(w, fmt.Sprintf("can't send message to channel: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	s.API.LogInfo("send message to", "channel", s.channelId)

	w.WriteHeader(http.StatusOK)
}

func (s *Sip) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

func (s *Sip) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("secret") != s.Secret {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	dashboardHTMLTemp.Execute(w, s.status)
}

func (s *Sip) markdownStatus() string {
	var buf bytes.Buffer

	userStatiMarkdownTemp.Execute(&buf, s.status)

	return buf.String()
}
