package message

import (
	"bytes"
	"encoding/json"
	"strings"
)

type Normalizer interface {
	Normalize(raw any) ([]Message, error)
}

type StdinNormalizer struct{}

func (n *StdinNormalizer) Normalize(raw any) ([]Message, error) {
	data, ok := raw.([]byte)
	if !ok {
		return nil, nil
	}
	if len(data) == 0 {
		return nil, nil
	}
	return []Message{NewInput(data)}, nil
}

type JSONNormalizer struct{}

func (n *JSONNormalizer) Normalize(raw any) ([]Message, error) {
	data, ok := raw.([]byte)
	if !ok {
		return nil, nil
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return nil, nil
	}
	var env struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return []Message{{
		Type:    Type(env.Type),
		Payload: env.Payload,
		Source:  "json",
	}}, nil
}

type LineNormalizer struct {
	buf strings.Builder
}

func (n *LineNormalizer) Normalize(raw any) ([]Message, error) {
	data, ok := raw.([]byte)
	if !ok {
		return nil, nil
	}
	n.buf.WriteString(string(data))
	var msgs []Message
	for {
		idx := strings.IndexByte(n.buf.String(), '\n')
		if idx < 0 {
			break
		}
		line := n.buf.String()[:idx]
		n.buf.Reset()
		n.buf.WriteString(n.buf.String()[idx+1:])
		if len(line) > 0 {
			msgs = append(msgs, NewMessage(TypeStream, line))
		}
	}
	return msgs, nil
}

type ChainNormalizer struct {
	normalizers []Normalizer
}

func NewChain(normalizers ...Normalizer) *ChainNormalizer {
	return &ChainNormalizer{normalizers: normalizers}
}

func (c *ChainNormalizer) Normalize(raw any) ([]Message, error) {
	for _, n := range c.normalizers {
		msgs, err := n.Normalize(raw)
		if err != nil || len(msgs) > 0 {
			return msgs, err
		}
	}
	return nil, nil
}
