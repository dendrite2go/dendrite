package axon_utils

import (
	context "context"
	fmt "fmt"
	log "log"
	reflect "reflect"
	time "time"

	proto "github.com/golang/protobuf/proto"
	uuid "github.com/google/uuid"
	grpc_codes "google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"

	axon_server "github.com/dendrite2go/dendrite/src/pkg/grpc/axon_server"
)

func SubscribeCommand(commandName string, stream axon_server.CommandService_OpenStreamClient, clientInfo *axon_server.ClientIdentification) {
	id := uuid.New()
	subscription := axon_server.CommandSubscription{
		MessageId:     id.String(),
		Command:       commandName,
		ClientId:      clientInfo.ClientId,
		ComponentName: clientInfo.ComponentName,
	}
	log.Printf("Subscribe command: Subscription: %v", subscription)
	subscriptionRequest := axon_server.CommandProviderOutbound_Subscribe{
		Subscribe: &subscription,
	}

	outbound := axon_server.CommandProviderOutbound{
		Request: &subscriptionRequest,
	}

	e := stream.Send(&outbound)
	if e != nil {
		panic(fmt.Sprintf("Subscribe command: Error sending subscription: %v", e))
	}
}

func AppendEvent(event Event, aggregateId string, projection interface{}, clientConnection *ClientConnection) (*Error, error) {
	log.Printf("Append event: event type kind: %v", reflect.TypeOf(event).Kind())
	eventType := reflect.TypeOf(event).Elem().Name()
	log.Printf("Append event: %v: %v", aggregateId, eventType)
	data, e := proto.Marshal(event)
	if e != nil {
		panic(fmt.Sprintf("Append event: could not marshal event: %v: %v", aggregateId, eventType))
	}
	message := &axon_server.SerializedObject{
		Type: eventType,
		Data: data,
	}

	conn := clientConnection.Connection
	client := axon_server.NewEventStoreClient(conn)

	var next int64
	switch p := projection.(type) {
	case CachedProjection:
		next = p.GetAggregateState().GetSequenceNumber() + 1
		log.Printf("Append event: Next sequence-nr after cached projection: %v", next)
	default:
		readRequest := axon_server.ReadHighestSequenceNrRequest{
			AggregateId:    aggregateId,
			FromSequenceNr: 0,
		}
		log.Printf("Append event: Read highest sequence-nr request: %v", readRequest)

		response, e := client.ReadHighestSequenceNr(context.Background(), &readRequest)
		if e != nil {
			log.Fatalf("Append event: Error while reading highest sequence-nr: %v", e)
			return nil, e
		}

		log.Printf("Append event: Response: %v", response)
		next = response.ToSequenceNr + 1
	}
	log.Printf("Append event: Next sequence number: %v", next)

	timestamp := time.Now().UnixNano() / 1000000

	id := uuid.New()
	eventMessage := axon_server.Event{
		MessageIdentifier:       id.String(),
		AggregateIdentifier:     aggregateId,
		AggregateSequenceNumber: next,
		AggregateType:           "ExampleAggregate",
		Timestamp:               timestamp,
		Snapshot:                false,
		Payload:                 message,
	}
	log.Printf("Append event: Event: %v", eventMessage)

	stream, e := client.AppendEvent(context.Background())
	if e != nil {
		log.Fatalf("Append event: Error while preparing to append event: %v", e)
		return nil, e
	}

	e = stream.Send(&eventMessage)
	if e != nil {
		log.Fatalf("Append event: Error while sending event: %v", e)
		return nil, e
	}

	confirmation, e := stream.CloseAndRecv()
	if e != nil {
		code := grpc_status.Code(e)
		if code == grpc_codes.OutOfRange {
			log.Printf("Append event: Out of range: %v", e)
			return &Error{
				Code:                "",
				Message:             "Out of range",
				AggregateIdentifier: aggregateId,
			}, nil
		}
		log.Fatalf("Append event: Error while closing event stream: %v: %v", e, code)
		return nil, e
	}

	log.Printf("Append event: Confirmation: %v", confirmation)
	event.ApplyTo(projection)
	return nil, nil
}

func CommandRespond(stream axon_server.CommandService_OpenStreamClient, requestId string) {
	id := uuid.New()
	commandResponse := axon_server.CommandResponse{
		MessageIdentifier: id.String(),
		RequestIdentifier: requestId,
	}
	log.Printf("Command respond: Command response: %v", commandResponse)
	commandResponseRequest := axon_server.CommandProviderOutbound_CommandResponse{
		CommandResponse: &commandResponse,
	}

	outbound := axon_server.CommandProviderOutbound{
		Request: &commandResponseRequest,
	}

	e := stream.Send(&outbound)
	if e != nil {
		panic(fmt.Sprintf("Command respond: Error sending command response: %v", e))
	}
}

func ReportError(stream axon_server.CommandService_OpenStreamClient, requestId string, errorCode string, errorMessageText string) {
	errorMessage := axon_server.ErrorMessage{
		Message:  errorMessageText,
		Location: "",
		Details:  nil,
	}

	id := uuid.New()
	commandResponse := axon_server.CommandResponse{
		MessageIdentifier: id.String(),
		RequestIdentifier: requestId,
		ErrorCode:         errorCode,
		ErrorMessage:      &errorMessage,
	}
	log.Printf("Command handler: Command error: %v", commandResponse)
	commandResponseRequest := axon_server.CommandProviderOutbound_CommandResponse{
		CommandResponse: &commandResponse,
	}

	outbound := axon_server.CommandProviderOutbound{
		Request: &commandResponseRequest,
	}

	e := stream.Send(&outbound)
	if e != nil {
		panic(fmt.Sprintf("Command handler: Error sending command error: %v", e))
	}
}

func CommandAddPermits(amount int64, stream axon_server.CommandService_OpenStreamClient, clientId string) {
	flowControl := axon_server.FlowControl{
		ClientId: clientId,
		Permits:  amount,
	}
	log.Printf("Command handler: Flow control: %v", flowControl)
	flowControlRequest := axon_server.CommandProviderOutbound_FlowControl{
		FlowControl: &flowControl,
	}

	outbound := axon_server.CommandProviderOutbound{
		Request: &flowControlRequest,
	}

	e := stream.Send(&outbound)
	if e != nil {
		panic(fmt.Sprintf("Command handler: Error sending flow control: %v", e))
	}
}
