// Package alice contains the skill's dialogue logic and webhook protocol types.
package alice

import "encoding/json"

// Request models the subset of the Alice protocol used by this skill.
// Additional protocol fields are intentionally ignored when decoding.
type Request struct {
	Version string  `json:"version"`
	Session Session `json:"session"`
	Request Input   `json:"request"`
}

type Session struct {
	New       bool   `json:"new"`
	SessionID string `json:"session_id"`
	MessageID int    `json:"message_id"`
	SkillID   string `json:"skill_id"`
}

type Input struct {
	Type    string          `json:"type"`
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	Version  string `json:"version"`
	Response Reply  `json:"response"`
}

type Reply struct {
	Text       string `json:"text"`
	EndSession bool   `json:"end_session"`
}
