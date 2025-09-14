package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

// MessageValidator validates messages against a pre-loaded JSON schema.
type MessageValidator struct {
	once   sync.Once
	schema *gojsonschema.Schema
	err    error
	path   string
}

func NewMessageValidator(schemaPath string) *MessageValidator {
	return &MessageValidator{path: schemaPath}
}

const defaultSchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title": "ChatMessage",
	"type": "object",
	"properties": {
		"message_id": { "type": "string" },
		"user_id": { "type": "string" },
		"content": { "type": "string" },
		"chat_id": { "type": "integer" },
		"timestamp": { "type": "string", "format": "date-time" }
	},
	"required": ["message_id", "user_id", "content", "timestamp"]
}`

func (v *MessageValidator) load() {
	var loader gojsonschema.JSONLoader
	if v.path != "" {
		if _, err := os.Stat(v.path); err == nil {
			tmp := filepath.ToSlash(v.path)
			loader = gojsonschema.NewReferenceLoader("file://" + tmp)
		}
	}
	if loader == nil {
		// fallback to embedded schema
		loader = gojsonschema.NewStringLoader(defaultSchema)
	}
	v.schema, v.err = gojsonschema.NewSchema(loader)
	if v.err != nil {
		v.err = fmt.Errorf("compile schema: %w", v.err)
	}
}

func (v *MessageValidator) Validate(doc interface{}) error {
	v.once.Do(v.load)
	if v.err != nil {
		return v.err
	}
	b, _ := json.Marshal(doc)
	res, err := v.schema.Validate(gojsonschema.NewBytesLoader(b))
	if err != nil {
		return err
	}
	if !res.Valid() {
		return fmt.Errorf("message invalid: %v", res.Errors())
	}
	return nil
}
