package logger

import (
	"encoding/json"
	"os"
	"time"
)

type Field struct {
	Key   string
	Value interface{}
}

func log(level, msg string, fields []Field, err error) {
	entry := map[string]interface{}{
		"ts":    time.Now().UTC().Format(time.RFC3339Nano),
		"level": level,
		"msg":   msg,
	}
	if err != nil {
		entry["error"] = err.Error()
	}
	for _, f := range fields {
		entry[f.Key] = f.Value
	}
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(entry)
}

func Info(msg string, fields ...Field) {
	log("info", msg, fields, nil)
}

func Error(msg string, err error, fields ...Field) {
	log("error", msg, fields, err)
}

func Debug(msg string, fields ...Field) {
	if os.Getenv("DEBUG") == "1" {
		log("debug", msg, fields, nil)
	}
}

func FieldKV(key string, value interface{}) Field { return Field{Key: key, Value: value} }
