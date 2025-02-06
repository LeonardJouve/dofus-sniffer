package sniffer

import (
	"fmt"

	"encoding/binary"
	"net"
	"reflect"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"

	"dofus-sniffer/proto/connection/connection"
	game "dofus-sniffer/proto/game/message"

	"github.com/google/gopacket/layers"
	"google.golang.org/protobuf/proto"
)

type Handler = func(packet gopacket.Packet)

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

const CONNECTION_SERVER = "dofus2-co-production.ankama-games.com"

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

		fragmentBuffer[string(ip.SrcIP)] = append(fragmentBuffer[string(ip.SrcIP)], tcp.Payload...)

		for {
			size, sizeLength := binary.Uvarint(fragmentBuffer[string(ip.SrcIP)])
			if sizeLength <= 0 || uint64(len(fragmentBuffer[string(ip.SrcIP)])-sizeLength) < size {
				break
			}

			payload := fragmentBuffer[string(ip.SrcIP)][sizeLength : size+uint64(sizeLength)]

			switch true {
			case ip.SrcIP.Equal(connectionServerIp):
				handleConnectionServerMessage(payload, size, &gameServerIp)
			case ip.DstIP.Equal(connectionServerIp):
				handleConnectionClientMessage(payload, size)
			case ip.SrcIP.Equal(gameServerIp):
				handleGameServerMessage(payload, size)
			case ip.DstIP.Equal(gameServerIp):
				handleGameClientMessage(payload, size)
			}

			fragmentBuffer[string(ip.SrcIP)] = fragmentBuffer[string(ip.SrcIP)][uint64(sizeLength)+size:]
		}
	}, nil
}

func handleConnectionServerMessage(payload []byte, size uint64, gameServerIp *net.IP) {
	fmt.Printf("[Connection S -> C] Received %d bytes\n", size)

	message, ok := reflect.New(reflect.TypeOf(&connection.Message{}).Elem()).Interface().(proto.Message)
	if !ok {
		return
	}

	err := proto.Unmarshal(payload, message)
	if err != nil {
		return
	}

	connectionMessage, ok := message.(*connection.Message)
	if !ok {
		return
	}

	_, ok = connectionMessage.Content.(*connection.Message_Response)
	if !ok {
		return
	}
	response := connectionMessage.GetResponse()

	_, ok = response.Content.(*connection.Response_SelectServer)
	if !ok {
		return
	}
	selectServer := response.GetSelectServer()

	_, ok = selectServer.Result.(*connection.SelectServerResponse_Success_)
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

func handleConnectionClientMessage(payload []byte, size uint64) {
	fmt.Printf("[Connection C -> S] Received %d bytes\n", size)

	message, ok := reflect.New(reflect.TypeOf(&connection.Message{}).Elem()).Interface().(proto.Message)
	if !ok {
		return
	}

	err := proto.Unmarshal(payload, message)
	if err != nil {
		return
	}
}

func handleGameServerMessage(payload []byte, size uint64) {
	fmt.Printf("[Game S -> C] Received %d bytes\n", size)

	message, ok := reflect.New(reflect.TypeOf(&game.Message{}).Elem()).Interface().(proto.Message)
	if !ok {
		return
	}

	err := proto.Unmarshal(payload, message)
	if err != nil {
		return
	}
}

func handleGameClientMessage(payload []byte, size uint64) {
	fmt.Printf("[Game C -> S] Received %d bytes\n", size)

	message, ok := reflect.New(reflect.TypeOf(&game.Message{}).Elem()).Interface().(proto.Message)
	if !ok {
		return
	}

	err := proto.Unmarshal(payload, message)
	if err != nil {
		return
	}
}
