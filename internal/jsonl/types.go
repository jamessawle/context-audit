package jsonl

// Attachment is one attachment-typed line from the JSONL.
//
// In the on-disk format, attachment lines have shape:
//
//	{"type": "attachment", "attachment": {"type": "<subtype>", ...subtype-specific fields}}
//
// We flatten that here: SubType holds attachment.type, and the
// remaining fields are populated based on SubType. Callers must inspect
// SubType before reading sub-type-specific fields; fields not relevant
// to a given subtype are left zero-valued.
type Attachment struct {
	SubType  string   // "skill_listing", "deferred_tools_delta", "hook_success", "hook_additional_context", ...
	HookName string   // populated for hook_success and hook_additional_context
	Content  string   // populated for skill_listing (markdown) and hook_additional_context
	Stdout   string   // populated for hook_success (raw stdout, often JSON-stringified)
	Added    []string // populated for deferred_tools_delta (tool names)
}

// Session is the subset of a startup JSONL we care about in v0.1:
// the first assistant turn's cache_creation_input_tokens (a proxy for
// total startup context size) and every attachment line in order.
//
// Warnings records non-fatal parse issues (malformed lines, unknown
// content shapes) so callers can surface schema drift without the
// parser aborting. A non-empty Warnings slice is the observable signal
// that the transcript contained something the parser did not fully
// understand.
type Session struct {
	CacheCreationInputTokens int
	Attachments              []Attachment
	Warnings                 []string
}
