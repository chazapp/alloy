// Package kafka provides an otelcol.receiver.kafka component.
package kafka

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/otelcol"
	otelcolCfg "github.com/grafana/alloy/internal/component/otelcol/config"
	"github.com/grafana/alloy/internal/component/otelcol/receiver"
	"github.com/grafana/alloy/internal/featuregate"
	"github.com/mitchellh/mapstructure"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"
	otelcomponent "go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/pipeline"
)

func init() {
	component.Register(component.Registration{
		Name:      "otelcol.receiver.kafka",
		Stability: featuregate.StabilityGenerallyAvailable,
		Args:      Arguments{},

		Build: func(opts component.Options, args component.Arguments) (component.Component, error) {
			fact := kafkareceiver.NewFactory()
			return receiver.New(opts, fact, args.(Arguments))
		},
	})
}

// Arguments configures the otelcol.receiver.kafka component.
type Arguments struct {
	Brokers           []string      `alloy:"brokers,attr"`
	ProtocolVersion   string        `alloy:"protocol_version,attr"`
	SessionTimeout    time.Duration `alloy:"session_timeout,attr,optional"`
	HeartbeatInterval time.Duration `alloy:"heartbeat_interval,attr,optional"`
	Topic             string        `alloy:"topic,attr,optional"`
	Encoding          string        `alloy:"encoding,attr,optional"`
	GroupID           string        `alloy:"group_id,attr,optional"`
	ClientID          string        `alloy:"client_id,attr,optional"`
	InitialOffset     string        `alloy:"initial_offset,attr,optional"`

	ResolveCanonicalBootstrapServersOnly bool `alloy:"resolve_canonical_bootstrap_servers_only,attr,optional"`

	Authentication   otelcol.KafkaAuthenticationArguments `alloy:"authentication,block,optional"`
	Metadata         otelcol.KafkaMetadataArguments       `alloy:"metadata,block,optional"`
	AutoCommit       AutoCommitArguments                  `alloy:"autocommit,block,optional"`
	MessageMarking   MessageMarkingArguments              `alloy:"message_marking,block,optional"`
	HeaderExtraction HeaderExtraction                     `alloy:"header_extraction,block,optional"`

	MinFetchSize     int32 `alloy:"min_fetch_size,attr,optional"`
	DefaultFetchSize int32 `alloy:"default_fetch_size,attr,optional"`
	MaxFetchSize     int32 `alloy:"max_fetch_size,attr,optional"`

	ErrorBackOff ErrorBackOffArguments `alloy:"error_backoff,block,optional"`

	// DebugMetrics configures component internal metrics. Optional.
	DebugMetrics otelcolCfg.DebugMetricsArguments `alloy:"debug_metrics,block,optional"`

	// Output configures where to send received data. Required.
	Output *otelcol.ConsumerArguments `alloy:"output,block"`
}

var _ receiver.Arguments = Arguments{}

// SetToDefault implements syntax.Defaulter.
func (args *Arguments) SetToDefault() {
	*args = Arguments{
		// We use the defaults from the upstream OpenTelemetry Collector component
		// for compatibility, even though that means using a client and group ID of
		// "otel-collector".

		Encoding:          "otlp_proto",
		Brokers:           []string{"localhost:9092"},
		ClientID:          "otel-collector",
		GroupID:           "otel-collector",
		InitialOffset:     "latest",
		SessionTimeout:    10 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		MinFetchSize:      1,
		DefaultFetchSize:  1048576,
		MaxFetchSize:      0,
	}
	args.Metadata.SetToDefault()
	args.AutoCommit.SetToDefault()
	args.MessageMarking.SetToDefault()
	args.HeaderExtraction.SetToDefault()
	args.DebugMetrics.SetToDefault()
}

// Validate implements syntax.Validator.
func (args *Arguments) Validate() error {
	var signals []string

	if len(args.Topic) > 0 {
		if len(args.Output.Logs) > 0 {
			signals = append(signals, "logs")
		}
		if len(args.Output.Metrics) > 0 {
			signals = append(signals, "metrics")
		}
		if len(args.Output.Traces) > 0 {
			signals = append(signals, "traces")
		}
		if len(signals) > 1 {
			return fmt.Errorf("only one signal can be set in the output block when a Kafka topic is explicitly set; currently set signals: %s", strings.Join(signals, ", "))
		}
	}

	if args.ErrorBackOff.Enabled {
		if args.ErrorBackOff.Multiplier <= 1 {
			return fmt.Errorf("multiplier must be greater than 1.0")
		}

		if args.ErrorBackOff.RandomizationFactor < 0 {
			return fmt.Errorf("randomization_factor must be greater or equal to 0")
		}
	}

	return nil
}

type ErrorBackOffArguments struct {
	Enabled             bool          `alloy:"enabled,attr,optional"`
	InitialInterval     time.Duration `alloy:"initial_interval,attr,optional"`
	RandomizationFactor float64       `alloy:"randomization_factor,attr,optional"`
	Multiplier          float64       `alloy:"multiplier,attr,optional"`
	MaxInterval         time.Duration `alloy:"max_interval,attr,optional"`
	MaxElapsedTime      time.Duration `alloy:"max_elapsed_time,attr,optional"`
}

// Convert converts args into the upstream type.
func (args *ErrorBackOffArguments) Convert() *configretry.BackOffConfig {
	if args == nil {
		return nil
	}

	return &configretry.BackOffConfig{
		Enabled:             args.Enabled,
		InitialInterval:     args.InitialInterval,
		RandomizationFactor: args.RandomizationFactor,
		Multiplier:          args.Multiplier,
		MaxInterval:         args.MaxInterval,
		MaxElapsedTime:      args.MaxElapsedTime,
	}
}

// Convert implements receiver.Arguments.
func (args Arguments) Convert() (otelcomponent.Config, error) {
	input := make(map[string]interface{})
	input["auth"] = args.Authentication.Convert()

	var result kafkareceiver.Config
	err := mapstructure.Decode(input, &result)
	if err != nil {
		return nil, err
	}

	result.Brokers = args.Brokers
	result.ProtocolVersion = args.ProtocolVersion
	result.SessionTimeout = args.SessionTimeout
	result.HeartbeatInterval = args.HeartbeatInterval
	result.Topic = args.Topic
	result.Encoding = args.Encoding
	result.GroupID = args.GroupID
	result.ClientID = args.ClientID
	result.InitialOffset = args.InitialOffset
	result.ResolveCanonicalBootstrapServersOnly = args.ResolveCanonicalBootstrapServersOnly
	result.Metadata = args.Metadata.Convert()
	result.AutoCommit = args.AutoCommit.Convert()
	result.MessageMarking = args.MessageMarking.Convert()
	result.HeaderExtraction = args.HeaderExtraction.Convert()
	result.MinFetchSize = args.MinFetchSize
	result.DefaultFetchSize = args.DefaultFetchSize
	result.MaxFetchSize = args.MaxFetchSize
	result.ErrorBackOff = *args.ErrorBackOff.Convert()

	return &result, nil
}

// Extensions implements receiver.Arguments.
func (args Arguments) Extensions() map[otelcomponent.ID]otelcomponent.Component {
	return nil
}

// Exporters implements receiver.Arguments.
func (args Arguments) Exporters() map[pipeline.Signal]map[otelcomponent.ID]otelcomponent.Component {
	return nil
}

// NextConsumers implements receiver.Arguments.
func (args Arguments) NextConsumers() *otelcol.ConsumerArguments {
	return args.Output
}

// AutoCommitArguments configures how to automatically commit updated topic
// offsets back to the Kafka broker.
type AutoCommitArguments struct {
	Enable   bool          `alloy:"enable,attr,optional"`
	Interval time.Duration `alloy:"interval,attr,optional"`
}

func (args *AutoCommitArguments) SetToDefault() {
	*args = AutoCommitArguments{
		Enable:   true,
		Interval: time.Second,
	}
}

// Convert converts args into the upstream type.
func (args AutoCommitArguments) Convert() kafkareceiver.AutoCommit {
	return kafkareceiver.AutoCommit{
		Enable:   args.Enable,
		Interval: args.Interval,
	}
}

// MessageMarkingArguments configures when Kafka messages are marked as read.
type MessageMarkingArguments struct {
	AfterExecution      bool `alloy:"after_execution,attr,optional"`
	IncludeUnsuccessful bool `alloy:"include_unsuccessful,attr,optional"`
}

func (args *MessageMarkingArguments) SetToDefault() {
	*args = MessageMarkingArguments{
		AfterExecution:      false,
		IncludeUnsuccessful: false,
	}
}

// Convert converts args into the upstream type.
func (args MessageMarkingArguments) Convert() kafkareceiver.MessageMarking {
	return kafkareceiver.MessageMarking{
		After:   args.AfterExecution,
		OnError: args.IncludeUnsuccessful,
	}
}

type HeaderExtraction struct {
	ExtractHeaders bool     `alloy:"extract_headers,attr,optional"`
	Headers        []string `alloy:"headers,attr,optional"`
}

func (h *HeaderExtraction) SetToDefault() {
	*h = HeaderExtraction{
		ExtractHeaders: false,
		Headers:        []string{},
	}
}

// Convert converts HeaderExtraction into the upstream type.
func (h HeaderExtraction) Convert() kafkareceiver.HeaderExtraction {
	return kafkareceiver.HeaderExtraction{
		ExtractHeaders: h.ExtractHeaders,
		Headers:        h.Headers,
	}
}

// DebugMetricsConfig implements receiver.Arguments.
func (args Arguments) DebugMetricsConfig() otelcolCfg.DebugMetricsArguments {
	return args.DebugMetrics
}
