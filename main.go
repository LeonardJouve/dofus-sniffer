package main

import "dofus-sniffer/sniffer"

func main() {
	handle, err := sniffer.MakeHandler()
	if err != nil {
		return
	}

	sniffer.Listen("\\Device\\NPF_{C9738EAA-F05B-4ECC-BE72-55FC6DF19217}", "tcp port 5555", handle)
}
