package upload

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

type QCOWHeader struct {
	Size   uint64
}

func main() {
	f, err := os.Open("/tmp/cirros-0.4.0-x86_64-disk.img")

	if err != nil {
		panic(err)
	}

	header := make([]byte, 32)
	_, err = f.Read(header)

	if err != nil {
		log.Printf("couldn't read header")
	}

	qcowHeader, err := Parse(header)

	log.Printf("qemu header: %v", qcowHeader)

	defer f.Close()
}

func Parse(header []byte) (*QCOWHeader, error) {
	h := QCOWHeader{}
	isQCOW := string(header[0:4]) == "QFI\xfb"
	if !isQCOW {
		return nil, fmt.Errorf("not a qcow header")
	}
	h.Size = binary.BigEndian.Uint64(header[24:32])
	return &h, nil
}
