package stages

import (
	"testing"
	"time"

	"github.com/grafana/alloy/internal/featuregate"
	"github.com/grafana/alloy/internal/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

var testJSONDropAlloy = `
stage.json_drop {
	keys = ["extra"]
}
`

var testJSONDropLogLine = `{"app":"loki","component":["parser","type"],"duration":125,"extra":"{\"user\":\"marco\"}","level":"WARN","message":"this is a log line","nested":{"child":"value"},"time":"2012-11-01T22:08:41+00:00"}`
var testJSONDropLogLineExpected = `{"app":"loki","component":["parser","type"],"duration":125,"level":"WARN","message":"this is a log line","nested":{"child":"value"},"time":"2012-11-01T22:08:41+00:00"}`

func TestPipeline_JSONDrop(t *testing.T) {
	t.Parallel()
	logger := util.TestAlloyLogger(t)

	tests := map[string]struct {
		config         string
		entry          string
		expectedOutput string
	}{
		"succesfully run a pipeline with 1 json_drop stage removing keys": {
			testJSONDropAlloy,
			testJSONDropLogLine,
			testJSONDropLogLineExpected,
		},
	}
	for testName, testData := range tests {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			logger.Log(testData)

			pl, err := NewPipeline(logger, loadConfig(testData.config), nil, prometheus.DefaultRegisterer, featuregate.StabilityGenerallyAvailable)
			assert.NoError(t, err, "Expected pipeline creation to not result in error")
			out := processEntries(pl, newEntry(nil, nil, testData.entry, time.Now()))[0]
			assert.Equal(t, testData.expectedOutput, out.Entry.Line)
		})
	}
}
