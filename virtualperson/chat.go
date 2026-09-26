package virtualperson

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// PersonaCard is the adult character prompt + display metadata.
// Safety: characters are adults only; no minor roleplay.
type PersonaCard struct {
	CharacterID  CharacterID
	Name         string
	SystemPrompt string
	AdultOnly    bool // must be true to enable chat
}

// DefaultPersona01 is the first Virtual Person chat card (stub text).
var DefaultPersona01 = PersonaCard{
	CharacterID: "character-01",
	Name:        "Character 01",
	AdultOnly:   true,
	SystemPrompt: strings.TrimSpace(`
You are Character 01, an adult Virtual Person in SamNPlayer.
All participants are consenting adults. Refuse any minor or underage content.
Virtual toys are props in the scene (not real hardware commands).
When giving a prop or starting an activity, end with tags on their own lines:
[prop: give=dildo]
[activity: id=titjob_dildo intensity=0.5 duration=30s]
Allowed props: dildo
Allowed activities: idle, titjob_dildo, kiss, stroke_slow, stroke_fast, oral_suction, tease, climax_window, stop
Do not invent other ids. Do not emit raw device commands.
`),
}

// ChatMessage is one turn in the dialogue.
type ChatMessage struct {
	Role    string // "user" | "assistant" | "system"
	Content string
}

// ChatReply is assistant text plus optional prop/activity directives.
type ChatReply struct {
	Text     string
	GiveProp PropID
	Activity *ActivityIntent
}

// ChatBackend talks to a local or remote LLM.
// Prefer local OpenAI-compatible servers (Colibri, LM Studio, Ollama).
type ChatBackend interface {
	Complete(ctx context.Context, persona PersonaCard, history []ChatMessage) (ChatReply, error)
}

// EchoBackend is an MVP stub: echoes the user and emits demo tags.
type EchoBackend struct{}

// Complete implements ChatBackend.
func (EchoBackend) Complete(_ context.Context, _ PersonaCard, history []ChatMessage) (ChatReply, error) {
	var last string
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			last = history[i].Content
			break
		}
	}
	text := "I hear you. (chat stub — wire Colibri/LM Studio next)"
	if last != "" {
		text = "Got it: " + last + "\n[prop: give=dildo]\n[activity: id=titjob_dildo intensity=0.45 duration=20s]"
	}
	return ChatReply{
		Text:     text,
		GiveProp: ParsePropGiveTag(text),
		Activity: ParseActivityTag(text),
	}, nil
}

var (
	activityTagRE = regexp.MustCompile(`(?i)\[activity:\s*id=([a-z0-9_]+)\s+intensity=([0-9]*\.?[0-9]+)\s+duration=(\d+)s?\s*\]`)
	propGiveRE    = regexp.MustCompile(`(?i)\[prop:\s*give=([a-z0-9_]+)\s*\]`)
)

// ParseActivityTag extracts the last [activity: …] tag from text.
func ParseActivityTag(text string) *ActivityIntent {
	matches := activityTagRE.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	m := matches[len(matches)-1]
	intensity, _ := strconv.ParseFloat(m[2], 64)
	dur, _ := strconv.Atoi(m[3])
	return &ActivityIntent{
		ID:        ActivityID(strings.ToLower(m[1])),
		Intensity: clamp01(intensity),
		DurationS: dur,
		Source:    SourceChatPulse,
	}
}

// ParsePropGiveTag extracts [prop: give=…] if present.
func ParsePropGiveTag(text string) PropID {
	m := propGiveRE.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return PropID(strings.ToLower(m[1]))
}

// ChatSession holds history and a backend for one character.
type ChatSession struct {
	mu      sync.Mutex
	persona PersonaCard
	backend ChatBackend
	history []ChatMessage
	bus     *MotionBus
	catalog *ActivityCatalog
}

// NewChatSession builds a session. Backend defaults to EchoBackend.
func NewChatSession(persona PersonaCard, bus *MotionBus, catalog *ActivityCatalog, backend ChatBackend) *ChatSession {
	persona.AdultOnly = true
	if backend == nil {
		backend = EchoBackend{}
	}
	return &ChatSession{
		persona: persona,
		backend: backend,
		bus:     bus,
		catalog: catalog,
		history: []ChatMessage{{Role: "system", Content: persona.SystemPrompt}},
	}
}

// SendUser appends a user message, calls the backend, applies prop/activity tags.
func (s *ChatSession) SendUser(ctx context.Context, text string) (ChatReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, ChatMessage{Role: "user", Content: text})
	reply, err := s.backend.Complete(ctx, s.persona, s.history)
	if err != nil {
		return ChatReply{}, err
	}
	s.history = append(s.history, ChatMessage{Role: "assistant", Content: reply.Text})
	if reply.GiveProp == "" {
		reply.GiveProp = ParsePropGiveTag(reply.Text)
	}
	if reply.Activity == nil {
		reply.Activity = ParseActivityTag(reply.Text)
	}
	if s.catalog != nil {
		if reply.GiveProp != "" {
			_ = s.catalog.GiveProp(reply.GiveProp)
		}
		if reply.Activity != nil {
			_ = s.catalog.Start(*reply.Activity)
		}
	}
	return reply, nil
}

// LocalOpenAIBackend calls an OpenAI-compatible /v1/chat/completions endpoint
// (Colibri, LM Studio, Ollama with OpenAI shim). Default baseURL is
// http://127.0.0.1:11434/v1 for Ollama; override for other local servers.
// No cloud calls — local-first as recommended in the Virtual Person plan.
type LocalOpenAIBackend struct {
	BaseURL string // e.g. "http://127.0.0.1:1234/v1"
	Model   string // e.g. "local-model"
	// HTTPClient optional; if nil, uses http.DefaultClient with a short timeout.
}

// Complete implements ChatBackend. On network error it falls back to EchoBackend
// so offline demos still emit demo prop/activity tags.
func (b LocalOpenAIBackend) Complete(ctx context.Context, persona PersonaCard, history []ChatMessage) (ChatReply, error) {
	base := strings.TrimRight(b.BaseURL, "/")
	if base == "" {
		base = "http://127.0.0.1:11434/v1"
	}
	model := b.Model
	if model == "" {
		model = "local-model"
	}

	type oaMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	msgs := make([]oaMsg, 0, len(history))
	for _, m := range history {
		msgs = append(msgs, oaMsg{Role: m.Role, Content: m.Content})
	}
	body, err := json.Marshal(map[string]any{
		"model":       model,
		"messages":    msgs,
		"temperature": 0.7,
		"max_tokens":  400,
	})
	if err != nil {
		return EchoBackend{}.Complete(ctx, persona, history)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return EchoBackend{}.Complete(ctx, persona, history)
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return EchoBackend{}.Complete(ctx, persona, history)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return EchoBackend{}.Complete(ctx, persona, history)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil || len(parsed.Choices) == 0 {
		return EchoBackend{}.Complete(ctx, persona, history)
	}
	text := parsed.Choices[0].Message.Content
	return ChatReply{
		Text:     text,
		GiveProp: ParsePropGiveTag(text),
		Activity: ParseActivityTag(text),
	}, nil
}
