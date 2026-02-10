package main

import "github.com/voocel/mas"

// agentEventMsg bridges mas agent events into the bubbletea Elm loop.
// Subscribe callback sends these via p.Send(agentEventMsg{ev}).
type agentEventMsg struct {
	event mas.Event
}
