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

func (v *MessageValidator) load() {
	data, err := os.ReadFile(v.path)
	if err != nil {
		v.err = fmt.Errorf("read schema: %w", err)
		return
	}
	tmp := filepath.ToSlash(v.path)
	loader := gojsonschema.NewReferenceLoader("file://" + tmp)
	v.schema, v.err = gojsonschema.NewSchema(loader)
	if v.err != nil {
		v.err = fmt.Errorf("compile schema: %w", v.err)
	}
	_ = data // keep for potential future hashing/versioning
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
