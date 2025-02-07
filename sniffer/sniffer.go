package sniffer

import (
	"fmt"

	"encoding/binary"
	"net"
	"reflect"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"

	"dofus-sniffer/messages"
	"dofus-sniffer/proto/connection/connection_message"
	"dofus-sniffer/proto/game/game_message"

	"github.com/google/gopacket/layers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Handler = func(packet gopacket.Packet)

const CONNECTION_SERVER = "dofus2-co-production.ankama-games.com"

func Listen(device string, filter string, handle Handler) {
	pcapHandle, err := pcap.OpenLive(device, 1600, true, pcap.BlockForever)
	if err != nil {
		return
	}
	defer pcapHandle.Close()

	err = pcapHandle.SetBPFFilter(filter)
	if err != nil {
		return
	}

	packetSource := gopacket.NewPacketSource(pcapHandle, pcapHandle.LinkType())
	fmt.Printf("Listening...\n")

	for packet := range packetSource.Packets() {
		handle(packet)
	}
}
func MakeHandler() (Handler, error) {
	fragmentBuffer := make(map[string][]byte)
	connectionServer, err := net.LookupIP(CONNECTION_SERVER)
	if err != nil || len(connectionServer) == 0 {
		return nil, err
	}
	connectionServerIp := connectionServer[0]
	var gameServerIp net.IP

	return func(packet gopacket.Packet) {
		if len(packet.Data()) == 0 {
			return
		}

		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			return
		}
		ip, ok := ipLayer.(*layers.IPv4)
		if !ok {
			return
		}

		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer == nil {
			return
		}
		tcp, ok := tcpLayer.(*layers.TCP)
		if !ok {
			return
		}

		fragmentBuffer[ip.SrcIP.String()] = append(fragmentBuffer[ip.SrcIP.String()], tcp.Payload...)

		for {
			size, sizeLength := binary.Uvarint(fragmentBuffer[ip.SrcIP.String()])
			if sizeLength <= 0 || uint64(len(fragmentBuffer[ip.SrcIP.String()])-sizeLength) < size {
				break
			}

			payload := fragmentBuffer[ip.SrcIP.String()][sizeLength : size+uint64(sizeLength)]

			switch true {
			case ip.SrcIP.Equal(connectionServerIp):
				fallthrough
			case ip.DstIP.Equal(connectionServerIp):
				handleConnectionMessage(payload, size, &gameServerIp)
			case ip.SrcIP.Equal(gameServerIp):
				fallthrough
			case ip.DstIP.Equal(gameServerIp):
				handleGameMessage(payload, size)
			}

			fragmentBuffer[ip.SrcIP.String()] = fragmentBuffer[ip.SrcIP.String()][uint64(sizeLength)+size:]
		}
	}, nil
}

func handleConnectionMessage(payload []byte, size uint64, gameServerIp *net.IP) {
	fmt.Printf("[Connection] Received %d bytes\n", size)

	message, ok := reflect.New(reflect.TypeOf(&connection_message.Message{}).Elem()).Interface().(proto.Message)
	if !ok {
		return
	}

	err := proto.Unmarshal(payload, message)
	if err != nil {
		return
	}

	connectionMessage, ok := message.(*connection_message.Message)
	if !ok {
		return
	}

	_, ok = connectionMessage.Content.(*connection_message.Message_Response)
	if !ok {
		return
	}
	response := connectionMessage.GetResponse()

	_, ok = response.Content.(*connection_message.Response_SelectServer)
	if !ok {
		return
	}
	selectServer := response.GetSelectServer()

	_, ok = selectServer.Result.(*connection_message.SelectServerResponse_Success_)
	if !ok {
		return
	}
	success := selectServer.GetSuccess()

	gameServer, err := net.LookupIP(success.Host)
	if err != nil || len(gameServer) == 0 {
		return
	}
	*gameServerIp = gameServer[0]
}

func handleGameMessage(payload []byte, size uint64) {
	message, ok := reflect.New(reflect.TypeOf(&game_message.Message{}).Elem()).Interface().(proto.Message)
	if !ok {
		return
	}

	err := proto.Unmarshal(payload, message)
	if err != nil {
		return
	}

	gameMessage, ok := message.(*game_message.Message)
	if !ok {
		return
	}

	var any *anypb.Any
	switch gameMessage.Content.(type) {
	case *game_message.Message_Request:
		any = gameMessage.GetRequest().GetContent()
	case *game_message.Message_Event:
		any = gameMessage.GetEvent().GetContent()
	case *game_message.Message_Response:
		any = gameMessage.GetResponse().Content
	default:
		return
	}

	messageType, ok := messages.KnownMessages[any.GetTypeUrl()]
	if !ok {
		fmt.Printf("[Game] Received %d bytes unknown type %s: %x\n", size, any.GetTypeUrl(), payload)
	} else {
		fmt.Printf("[Game] Received %d bytes %s\n", size, messageType.Descriptor().Name())
	}

}
