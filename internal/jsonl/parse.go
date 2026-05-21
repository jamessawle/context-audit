package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// ParseFile reads a Claude Code startup JSONL transcript and returns the
// subset of fields needed for v0.1 context auditing.
//
// Line types other than "assistant" and "attachment" are silently
// skipped. Unknown attachment subtypes are recorded with SubType set but
// all sub-type-specific fields left zero — that's intentional, so new
// subtypes don't break parsing.
//
// CacheCreationInputTokens is taken from the first "assistant" line
// whose message.usage.cache_creation_input_tokens is non-zero. In a
// --startup probe transcript that is the initial turn and represents
// the cost of the loaded startup context.
func ParseFile(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	session := &Session{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		var head struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &head); err != nil {
			return nil, fmt.Errorf("line %d: decode head: %w", lineNum, err)
		}

		switch head.Type {
		case "assistant":
			var line struct {
				Message struct {
					Usage struct {
						CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
				return nil, fmt.Errorf("line %d: decode assistant: %w", lineNum, err)
			}
			if session.CacheCreationInputTokens == 0 {
				session.CacheCreationInputTokens = line.Message.Usage.CacheCreationInputTokens
			}
		case "attachment":
			a, err := parseAttachment(scanner.Bytes())
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			session.Attachments = append(session.Attachments, a)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return session, nil
}

// parseAttachment unmarshals an attachment line. The on-disk shape is
//
//	{"type": "attachment", "attachment": {"type": "<subtype>", ...}}
//
// so we pull SubType from attachment.type and the rest of the fields
// from siblings inside the attachment object. Unknown subtypes are
// returned with empty sub-type-specific fields rather than producing an
// error — see the package doc for Attachment.
func parseAttachment(raw []byte) (Attachment, error) {
	var line struct {
		Attachment struct {
			Type       string          `json:"type"`
			HookName   string          `json:"hookName"`
			Content    json.RawMessage `json:"content"`
			Stdout     string          `json:"stdout"`
			AddedNames []string        `json:"addedNames"`
		} `json:"attachment"`
	}
	if err := json.Unmarshal(raw, &line); err != nil {
		return Attachment{}, fmt.Errorf("decode attachment: %w", err)
	}
	content, err := decodeContent(line.Attachment.Content)
	if err != nil {
		return Attachment{}, fmt.Errorf("decode attachment %q content: %w", line.Attachment.Type, err)
	}
	return Attachment{
		SubType:  line.Attachment.Type,
		HookName: line.Attachment.HookName,
		Content:  content,
		Stdout:   line.Attachment.Stdout,
		Added:    line.Attachment.AddedNames,
	}, nil
}

// decodeContent normalises the attachment.content field, which appears
// in two shapes in real transcripts:
//   - a plain string (skill_listing, hook_success)
//   - an array of strings concatenated as a single payload
//     (hook_additional_context)
//
// An empty / missing field returns "". Anything else is preserved as
// raw JSON text so callers can still inspect unknown shapes.
func decodeContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		out := ""
		for i, p := range arr {
			if i > 0 {
				out += "\n"
			}
			out += p
		}
		return out, nil
	}
	return string(raw), nil
}
