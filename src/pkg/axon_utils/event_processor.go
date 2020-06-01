package axon_utils

import (
	context "context"
	log "log"

	proto "github.com/golang/protobuf/proto"
	uuid "github.com/google/uuid"

	axon_server "github.com/dendrite2go/dendrite/src/pkg/grpc/axon_server"
)

func ProcessEvents(labelPrefix string, host string, port int, processorName string, projection interface{}, prepareUnmarshal func(payloadType string) Event, tokenStore TokenStore) *ClientConnection {
	label := labelPrefix + " event processor"

	clientConnection, stream := WaitForServer(host, port, label)
	log.Printf("%v: Connection and client info: %v: %v", label, clientConnection, stream)

	if e := registerProcessor(label, processorName, stream); e != nil {
		return clientConnection
	}

	go eventProcessorWorker(label+" worker", clientConnection, processorName, projection, prepareUnmarshal, tokenStore)

	return clientConnection
}

func registerProcessor(label string, processorName string, stream *axon_server.PlatformService_OpenStreamClient) error {
	processorInfo := axon_server.EventProcessorInfo{
		ProcessorName:    processorName,
		Mode:             "Tracking",
		ActiveThreads:    0,
		Running:          true,
		Error:            false,
		SegmentStatus:    nil,
		AvailableThreads: 1,
	}
	log.Printf("%v: event processor info: %v", label, processorInfo)
	subscriptionRequest := axon_server.PlatformInboundInstruction_EventProcessorInfo{
		EventProcessorInfo: &processorInfo,
	}

	id := uuid.New()
	inbound := axon_server.PlatformInboundInstruction{
		Request:       &subscriptionRequest,
		InstructionId: id.String(),
	}
	log.Printf("%v: event processor info: instruction ID: %v", label, inbound.InstructionId)

	e := (*stream).Send(&inbound)
	if e != nil {
		log.Printf("%v: Error sending registration: %v", label, e)
		return e
	}

	e = eventProcessorReceivePlatformInstruction(label, stream)
	if e != nil {
		log.Printf("%v: Error while waiting for acknowledgement of registration", label)
		return e
	}
	return nil
}

func eventProcessorWorker(label string, clientConnection *ClientConnection, processorName string, projection interface{}, prepareUnmarshal func(payloadType string) Event, tokenStore TokenStore) {
	conn := clientConnection.Connection
	clientInfo := clientConnection.ClientInfo

	token := tokenStore.ReadToken()

	eventStoreClient := axon_server.NewEventStoreClient(conn)
	log.Printf("%v: Event store client: %v", label, eventStoreClient)
	client, e := eventStoreClient.ListEvents(context.Background())
	if e != nil {
		log.Printf("%v: Error while opening ListEvents stream: %v", label, e)
		return
	}
	log.Printf("%v: List events client: %v", label, client)

	getEventsRequest := axon_server.GetEventsRequest{
		NumberOfPermits: 1,
		ClientId:        clientInfo.ClientId,
		ComponentName:   clientInfo.ComponentName,
		Processor:       processorName,
	}
	if token != nil {
		getEventsRequest.TrackingToken = *token + 1
	}
	log.Printf("%v: Get events request: %v", label, getEventsRequest)

	log.Printf("%v: Ready to process events", label)
	defer func() {
		log.Printf("Configuration event processor worker stopped")
	}()
	for true {
		e = client.Send(&getEventsRequest)
		if e != nil {
			log.Printf("%v: Error while sending get events request: %v", label, e)
			return
		}

		eventMessage, e := client.Recv()
		if e != nil {
			log.Printf("%v: Error while receiving next event: %v", label, e)
			return
		}
		log.Printf("%v: Next event message: %v", label, eventMessage)
		getEventsRequest.TrackingToken = eventMessage.Token

		if eventMessage.Event == nil || eventMessage.Event.Payload == nil {
			continue
		}

		payloadType := eventMessage.Event.Payload.Type
		event := prepareUnmarshal(payloadType)
		if event == nil {
			log.Printf("%v: Skipped unknown event: %v", label, payloadType)
			continue
		}
		if e = proto.Unmarshal(eventMessage.Event.Payload.Data, event.(proto.Message)); e != nil {
			log.Printf("%v: Unmarshalling of %v failed: %v", label, payloadType, e)
			return
		}
		event.ApplyTo(projection)

		e = tokenStore.WriteToken(getEventsRequest.TrackingToken)
		if e != nil {
			log.Printf("%v: Error while writing token: %v", label, e)
			return
		}
	}
}

func eventProcessorReceivePlatformInstruction(label string, stream *axon_server.PlatformService_OpenStreamClient) error {
	log.Printf("%v: Receive platform instruction: Waiting for outbound", label)
	outbound, e := (*stream).Recv()
	if e != nil {
		log.Printf("%v: Receive platform instruction: Error on receive, %v", label, e)
		return e
	}
	log.Printf("%v: Receive platform instruction: Outbound: %v", label, outbound)
	return nil
}
