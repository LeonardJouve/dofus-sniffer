package messages_test

import (
	"bytes"
	"dofus-sniffer/messages"
	"dofus-sniffer/proto/game/game_message"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func isOfType(message proto.Message, messageType protoreflect.MessageType) (bool, bool) {
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

func testMessage(t *testing.T, hexPayload string, messageType string) {
	payload, err := hex.DecodeString(hexPayload)
	if err != nil {
		t.Fatalf("Could not decode payload\n")
	}

	message, ok := reflect.New(reflect.TypeOf(&game_message.Message{}).Elem()).Interface().(proto.Message)
	if !ok {
		t.Fatalf("Could not instantiate message\n")
	}

	err = proto.Unmarshal(payload, message)
	if err != nil {
		t.Fatalf("Could not unmarshal payload\n")
		return
	}

	protoType, ok := messages.KnownMessages[messageType]
	if !ok {
		t.Fatalf("Could not find type\n")
	}

	isOfType, ok := isOfType(message, protoType)
	if !ok {
		t.Fatalf("Could not execute IsOfType\n")
	}

	if !isOfType {
		t.Fatalf("Invalid type\n")
	}
}

func TestChat(t *testing.T) {
	// ChatChannelMessageEvent
	testMessage(t, "1a670a650a13747970652e616e6b616d612e636f6d2f697963124e0a1a4163686574652047656c616e6f2050412f504d20446d206d6f6910061a19323032352d30322d30375432303a30383a30352b30313a303020c082c0d612289096e24a3a0856756c766f6d6178", "type.ankama.com/iyc")
}

func TestGamemap(t *testing.T) {
	// MapMovementEvent
	testMessage(t, "1a2c0a2a0a13747970652e616e6b616d612e636f6d2f69676712130a04db02a502100718dee3feffffffffffff01", "type.ankama.com/igg")
}

func TestArena(t *testing.T) {
	// ArenaRegisterRequest
	testMessage(t, "0a19085a12150a13747970652e616e6b616d612e636f6d2f6a7267", "type.ankama.com/jrg")
	// ArenaUnregisterRequest
	testMessage(t, "0a19085b12150a13747970652e616e6b616d612e636f6d2f6a7268", "type.ankama.com/jrh")
}

func TestQuest(t *testing.T) {
	// QuestObjectiveUnfollowRequest
	testMessage(t, "0a3208ffffffffffffffffff0112250a13747970652e616e6b616d612e636f6d2f686c72120e08f60c10ffffffffffffffffff01", "type.ankama.com/hlr")
	// QuestObjectiveFollowRequest
	testMessage(t, "0a3208ffffffffffffffffff0112250a13747970652e616e6b616d612e636f6d2f686c71120e08f60c10ffffffffffffffffff01", "type.ankama.com/hlq")
}
