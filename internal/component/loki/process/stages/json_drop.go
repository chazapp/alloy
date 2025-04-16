package stages

import (
	"encoding/json"

	"github.com/go-kit/log"
)

// JSONConfig represents a JSON Stage configuration
type JSONDropConfig struct {
	Keys          []string `alloy:"keys,attr"`
	DropMalformed bool     `alloy:"drop_malformed,attr,optional"`
}

// jsonStage sets extracted data using JMESPath expressions
type jsonDropStage struct {
	cfg    *JSONDropConfig
	logger log.Logger
}

func validateJSONDropConfig(cfg *JSONDropConfig) error {
	return nil
}

// newJSONStage creates a new json_drop pipeline stage from a config.
func newJSONDropStage(logger log.Logger, cfg JSONDropConfig) (Stage, error) {
	err := validateJSONDropConfig(&cfg)
	if err != nil {
		return nil, err
	}
	return &jsonDropStage{
		cfg:    &cfg,
		logger: log.With(logger, "component", "stage", "type", "json_drop"),
	}, nil
}

func (j *jsonDropStage) Run(in chan Entry) chan Entry {
	out := make(chan Entry)
	go func() {
		defer close(out)
		for e := range in {
			if err := j.processEntry(&e.Line); err != nil && j.cfg.DropMalformed {
				continue
			}
			out <- e
		}
	}()
	return out
}

func (j *jsonDropStage) Name() string {
	return "json_drop"
}

func (j *jsonDropStage) Cleanup() {
}

func (j *jsonDropStage) processEntry(line *string) error {
	var jsonLine map[string]any
	if err := json.Unmarshal([]byte(*line), &jsonLine); err != nil {
		return err
	}
	for _, key := range j.cfg.Keys {
		delete(jsonLine, key)
	}
	res, err := json.Marshal(jsonLine)
	if err != nil {
		return err
	}
	*line = string(res)
	return nil
}
