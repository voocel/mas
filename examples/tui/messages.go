package main

import "github.com/voocel/agentcore"

// agentEventMsg bridges agentcore agent events into the bubbletea Elm loop.
// Subscribe callback sends these via p.Send(agentEventMsg{ev}).
type agentEventMsg struct {
	event agentcore.Event
}
