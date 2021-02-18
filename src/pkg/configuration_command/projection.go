package configuration_command

import (
	log "log"

	axon_utils "github.com/dendrite2go/dendrite/src/pkg/axon_utils"
	grpc_config "github.com/dendrite2go/dendrite/src/pkg/grpc/dendrite_config"
)

// Redeclare event types, so that they can be extended with event handler methods.
type PropertyChangedEvent struct {
	grpc_config.PropertyChangedEvent
}

// Projection

type Projection struct {
	Configuration  map[string]string
	AggregateState axon_utils.AggregateState
}

func (projection *Projection) GetAggregateState() axon_utils.AggregateState {
	return projection.AggregateState
}

func RestoreProjection(aggregateIdentifier string, clientConnection *axon_utils.ClientConnection) *Projection {
	return axon_utils.RestoreProjection("Configuration", aggregateIdentifier, createInitialProjection, clientConnection, prepareUnmarshal).(*Projection)
}

func createInitialProjection() interface{} {
	return &Projection{
		Configuration:  make(map[string]string),
		AggregateState: axon_utils.NewAggregateState(),
	}
}

func (projection *Projection) Apply(event axon_utils.Event) {
	event.ApplyTo(projection)
}

// Map an event name as stored in AxonServer to an object that has two aspects:
// 1. It is a proto.Message, so it can be unmarshalled from the Axon event
// 2. It is an axon_utils.Event, so it can be applied to a projection
func prepareUnmarshal(payloadType string) (event axon_utils.Event) {
	log.Printf("Configuration Projection: Payload type: %v", payloadType)
	switch payloadType {
	case "PropertyChangedEvent":
		event = &PropertyChangedEvent{}
	default:
		event = nil
	}
	return event
}

// Event Handlers

func (event *PropertyChangedEvent) ApplyTo(projectionWrapper interface{}) {
	projection := projectionWrapper.(*Projection)
	projection.Configuration[event.Property.Key] = event.Property.Value
}
