package jsonl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ParseFile reads a Claude Code startup JSONL transcript and returns the
// subset of fields needed for v0.1 context auditing.
//
// Line types other than "assistant" and "attachment" are silently
// skipped. Unknown attachment subtypes are recorded with SubType set but
// all sub-type-specific fields left zero — that's intentional, so new
// subtypes don't break parsing.
//
// Blank lines are ignored. Lines that fail to decode are recorded as
// warnings on the returned Session and parsing continues, so a single
// truncated or malformed line does not discard everything parsed so
// far.
//
// The three usage fields — InputTokens, CacheCreationInputTokens, and
// CacheReadInputTokens — are taken from the very first "assistant" line,
// even if any of them are 0. In a --startup probe transcript that is the
// initial turn; their sum represents the harness's reported input-token
// cost of the loaded startup context (warm-cache runs are dominated by
// cache_read_input_tokens).
func ParseFile(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	session := &Session{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)

	var seenAssistant bool
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		var head struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &head); err != nil {
			session.Warnings = append(session.Warnings,
				fmt.Sprintf("line %d: decode head: %v", lineNum, err))
			continue
		}

		switch head.Type {
		case "assistant":
			var line struct {
				Message struct {
					Usage struct {
						InputTokens              int `json:"input_tokens"`
						CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
						CacheReadInputTokens     int `json:"cache_read_input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
				session.Warnings = append(session.Warnings,
					fmt.Sprintf("line %d: decode assistant: %v", lineNum, err))
				continue
			}
			if !seenAssistant {
				session.InputTokens = line.Message.Usage.InputTokens
				session.CacheCreationInputTokens = line.Message.Usage.CacheCreationInputTokens
				session.CacheReadInputTokens = line.Message.Usage.CacheReadInputTokens
				seenAssistant = true
			}
		case "attachment":
			a, warning, err := parseAttachment(scanner.Bytes(), lineNum)
			if err != nil {
				session.Warnings = append(session.Warnings,
					fmt.Sprintf("line %d: %v", lineNum, err))
				continue
			}
			if warning != "" {
				session.Warnings = append(session.Warnings, warning)
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
//
// If decodeContent encounters an unknown content shape, the returned
// warning is non-empty (and content holds the raw JSON for inspection).
func parseAttachment(raw []byte, lineNum int) (Attachment, string, error) {
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
		return Attachment{}, "", fmt.Errorf("decode attachment: %w", err)
	}
	content, unknownShape := decodeContent(line.Attachment.Content)
	var warning string
	if unknownShape {
		warning = fmt.Sprintf("line %d: attachment %q has unknown content shape: %s",
			lineNum, line.Attachment.Type, string(line.Attachment.Content))
	}
	return Attachment{
		SubType:  line.Attachment.Type,
		HookName: line.Attachment.HookName,
		Content:  content,
		Stdout:   line.Attachment.Stdout,
		Added:    line.Attachment.AddedNames,
	}, warning, nil
}

// decodeContent normalises the attachment.content field, which appears
// in two shapes in real transcripts:
//   - a plain string (skill_listing, hook_success)
//   - an array of strings concatenated as a single payload
//     (hook_additional_context)
//
// An empty / missing field returns "" with unknownShape=false. Anything
// else is preserved as raw JSON text and unknownShape=true so callers
// can record a warning about schema drift.
func decodeContent(raw json.RawMessage) (content string, unknownShape bool) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, false
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return strings.Join(arr, "\n"), false
	}
	return string(raw), true
}
