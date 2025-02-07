package utils

import (
	"bytes"
	"dofus-sniffer/proto/game/game_message"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func IsOfType(message proto.Message, messageType protoreflect.MessageType) (bool, bool) {
	gameMessage, ok := message.(*game_message.Message)
	if !ok {
		return false, false
	}

	var any *anypb.Any
	switch gameMessage.Content.(type) {
	case *game_message.Message_Request:
		any = gameMessage.GetRequest().GetContent()
	case *game_message.Message_Event:
		any = gameMessage.GetEvent().GetContent()
	case *game_message.Message_Response:
		any = gameMessage.GetResponse().GetContent()
	default:
		return false, false
	}

	msg := messageType.New().Interface()
	err := proto.Unmarshal(any.GetValue(), msg)
	if err != nil {
		return false, false
	}

	json, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
	if err != nil {
		return false, false
	}
	fmt.Printf("%s\n%s\n", any.GetTypeUrl(), string(json))

	buffer, err := proto.Marshal(msg)
	if err != nil {
		return false, false
	}

	return bytes.Equal(any.GetValue(), buffer), true
}
