// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package reporterimpl

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	observerdef "github.com/DataDog/datadog-agent/comp/anomalydetection/observer/def"
)

// TestBuildChangeEventPayload_WireShape asserts the JSON envelope produced for
// the event-management intake matches the v2 Events API ChangeEvent schema we
// used to obtain via datadog-api-client-go. The contract here is the wire
// format, not the helper functions: if the intake changes field names we want
// this test to fail loudly.
func TestBuildChangeEventPayload_WireShape(t *testing.T) {
	c := observerdef.ActiveCorrelation{
		Pattern: "kernel_bottleneck",
		Title:   "kernel bottleneck detected",
		Anomalies: []observerdef.Anomaly{
			{
				Type:   observerdef.AnomalyTypeMetric,
				Source: observerdef.SeriesDescriptor{Namespace: "dogstatsd", Tags: []string{"service:web", "env:prod"}},
			},
		},
	}

	payload := buildChangeEventPayload(c, "hello", "2024-01-01T00:00:00Z", "observer:kernel_bottleneck")

	// Round-trip through JSON so we exercise the same marshalling path as send().
	blob, err := json.Marshal(payload)
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(blob, &decoded))

	data, ok := decoded["data"].(map[string]any)
	assert.True(t, ok, "missing data envelope")
	assert.Equal(t, "event", data["type"])

	attrs, ok := data["attributes"].(map[string]any)
	assert.True(t, ok, "missing data.attributes")
	assert.Equal(t, "kernel bottleneck detected", attrs["title"])
	assert.Equal(t, "hello", attrs["message"])
	assert.Equal(t, "change", attrs["category"])
	assert.Equal(t, "2024-01-01T00:00:00Z", attrs["timestamp"])
	assert.Equal(t, "observer:kernel_bottleneck", attrs["aggregation_key"])

	tags, ok := attrs["tags"].([]any)
	assert.True(t, ok, "tags must be a JSON array")
	assert.NotEmpty(t, tags)

	inner, ok := attrs["attributes"].(map[string]any)
	assert.True(t, ok, "missing nested change-event attributes")

	changed, ok := inner["changed_resource"].(map[string]any)
	assert.True(t, ok, "missing changed_resource")
	assert.Equal(t, "kernel_bottleneck", changed["name"])
	assert.Equal(t, "configuration", changed["type"])

	author, ok := inner["author"].(map[string]any)
	assert.True(t, ok, "missing author")
	assert.Equal(t, "datadog-agent-observer", author["name"])
	assert.Equal(t, "automation", author["type"])

	impacted, ok := inner["impacted_resources"].([]any)
	assert.True(t, ok, "missing impacted_resources")
	assert.Len(t, impacted, 1)
	item := impacted[0].(map[string]any)
	assert.Equal(t, "web", item["name"])
	assert.Equal(t, "service", item["type"])

	assert.Contains(t, inner, "prev_value")
	assert.Contains(t, inner, "new_value")
	assert.Contains(t, inner, "change_metadata")
}

// TestBuildChangeEventPayload_TruncatesChangedResourceName ensures we don't
// emit a name longer than the v2 API accepts (128 chars).
func TestBuildChangeEventPayload_TruncatesChangedResourceName(t *testing.T) {
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'x'
	}
	c := observerdef.ActiveCorrelation{Pattern: string(long), Title: "t"}

	payload := buildChangeEventPayload(c, "m", "2024-01-01T00:00:00Z", "k")

	inner := payload["data"].(map[string]any)["attributes"].(map[string]any)["attributes"].(map[string]any)
	changed := inner["changed_resource"].(map[string]any)
	assert.Len(t, changed["name"].(string), changedResourceNameMaxLen)
}

// TestBuildChangeEventPayload_NoImpactedResourcesWhenEmpty makes sure we omit
// the impacted_resources key entirely (rather than emitting an empty array)
// when no service tags are present, matching the upstream omitempty behaviour.
func TestBuildChangeEventPayload_NoImpactedResourcesWhenEmpty(t *testing.T) {
	c := observerdef.ActiveCorrelation{Pattern: "p", Title: "t"}

	payload := buildChangeEventPayload(c, "m", "2024-01-01T00:00:00Z", "k")

	inner := payload["data"].(map[string]any)["attributes"].(map[string]any)["attributes"].(map[string]any)
	_, present := inner["impacted_resources"]
	assert.False(t, present, "impacted_resources should be omitted when no services are impacted")
}
