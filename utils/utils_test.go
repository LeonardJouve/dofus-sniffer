package utils_test

import (
	"dofus-sniffer/messages"
	"dofus-sniffer/proto/game/game_message"
	"dofus-sniffer/utils"
	"encoding/hex"
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
)

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

	isOfType, ok := utils.IsOfType(message, protoType)
	if !ok {
		t.Fatalf("Could not execute IsOfType\n")
	}

	if !isOfType {
		t.Fatalf("Invalid type\n")
	}
}

func TestIyc(t *testing.T) {
	testMessage(t, "1a670a650a13747970652e616e6b616d612e636f6d2f697963124e0a1a4163686574652047656c616e6f2050412f504d20446d206d6f6910061a19323032352d30322d30375432303a30383a30352b30313a303020c082c0d612289096e24a3a0856756c766f6d6178", "type.ankama.com/iyc")
}

func TestIgg(t *testing.T) {
	testMessage(t, "1a2c0a2a0a13747970652e616e6b616d612e636f6d2f69676712130a04db02a502100718dee3feffffffffffff01", "type.ankama.com/igg")
}
