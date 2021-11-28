package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"strconv"
)

// const pngImage string = "test.png"

const IENDchunk string = "IEND"

// PngHeader
type PngHeader struct {
	Header uint64
}

// ReadHeader
func (p *PngHeader) ReadHeader(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, &p.Header)
}

// Validate
func (p *PngHeader) Validate() (bool, error) {
	if p.Header == 0 {
		return false, nil
	}

	h := strconv.FormatUint(p.Header, 16)

	data, err := hex.DecodeString(h)
	if err != nil {
		return false, err
	}

	if string(data[1:4]) == "PNG" {
		return true, nil
	}

	return false, nil
}

// ChunkType
type ChunkType uint32

// String
func (c ChunkType) String() (string, error) {
	data, err := hex.DecodeString(strconv.FormatUint(uint64(c), 16))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// strToInt
func strToInt(n string) uint32 {
	return binary.BigEndian.Uint32([]byte(n))
}

// PngMetadata structure that holds png chunk fields.
type PngMetadata struct {
	Length uint32
	Type   ChunkType
	Data   []byte
	CRC    uint32
}

const NewChunkTypeName string = "pUNK"

// NewPngMetadata create a new png chunk function accept data as []byte and
// returns pointer to PngMetadata or an error.
func NewPngMetadata(data []byte) (*PngMetadata, error) {
	chunk := &PngMetadata{
		Length: uint32(len(data)),
		Type:   ChunkType(strToInt(NewChunkTypeName)),
		Data:   data,
	}
	if err := chunk.generateCRC(); err != nil {
		return nil, err
	}

	return chunk, nil
}

// generateCRC generates a new CRC for the new png chunk.
func (m *PngMetadata) generateCRC() error {
	buffer := &bytes.Buffer{}
	if err := binary.Write(buffer, binary.BigEndian, m.Type); err != nil {
		return err
	}

	if err := binary.Write(buffer, binary.BigEndian, m.Data); err != nil {
		return err
	}

	m.CRC = crc32.ChecksumIEEE(buffer.Bytes())
	return nil
}

func (m *PngMetadata) readLength(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, &m.Length)
}

func (m *PngMetadata) readType(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, &m.Type)
}

func (m *PngMetadata) readData(r io.Reader) error {
	buff := make([]byte, m.Length)
	if _, err := r.Read(buff); err != nil {
		return err
	}
	m.Data = buff

	return nil
}

func (m *PngMetadata) readCRC(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, &m.CRC)
}

// PNG represent a png image.
type PNG struct {
	Header PngHeader
	Chunks []*PngMetadata

	newChunkIndex int
}

// ReadChunks reads all the PNG chunks.
func (p *PNG) ReadChunks(r io.Reader) error {

	stop := false
	counter := 0
	for {

		metadata := new(PngMetadata)
		if err := metadata.readLength(r); err != nil {
			return err
		}

		if err := metadata.readType(r); err != nil {
			return err
		}

		typAsStr, err := metadata.Type.String()
		if err != nil {
			return err
		}

		if typAsStr == IENDchunk {
			stop = true
		}

		if err := metadata.readData(r); err != nil {
			return err
		}

		if err := metadata.readCRC(r); err != nil {
			return err
		}

		p.Chunks = append(p.Chunks, metadata)
		if stop {
			p.newChunkIndex = counter - 1
			return nil
		}

		counter++

	}

	return nil
}

func (p *PNG) PrintChunks(seek io.Writer) {
	for _, n := range p.Chunks {
		typ, _ := n.Type.String()
		fmt.Fprintf(seek, "--------------\n")
		fmt.Fprintf(seek, "Length %v\n", n.Length)
		fmt.Fprintf(seek, "Type %v\n", typ)
		fmt.Fprintf(seek, "CRC %v\n", n.CRC)
	}
}

// Marshal Chunks method used to add a new png chunk into the original one.
func (p *PNG) Marshal() ([]byte, error) {

	buff := new(bytes.Buffer)

	if err := binary.Write(buff, binary.BigEndian, p.Header.Header); err != nil {
		return nil, err
	}

	for _, n := range p.Chunks {
		if err := binary.Write(buff, binary.BigEndian, n.Length); err != nil {
			return nil, err
		}
		if err := binary.Write(buff, binary.BigEndian, uint32(n.Type)); err != nil {
			return nil, err
		}
		if err := binary.Write(buff, binary.BigEndian, n.Data); err != nil {
			return nil, err
		}
		if err := binary.Write(buff, binary.BigEndian, n.CRC); err != nil {
			return nil, err
		}
	}

	return buff.Bytes(), nil
}

func main() {
	// you can use the tool in order to store a message into a png image or
	// file data . for the moment we don't support encryption of the data that
	// we are hiding into the png image but that will come in the newer
	// versions.
	// ./stegno --encode --png <image path> --to <out file>  --data <message> or --file <file path>
	// ./stegno --decode --png <image path> --to <file path> or --dump true

	var (
		pngImage string
		to       string
		//
		encode  bool
		message string
		file    string
		//
		decode bool
		dump   bool
	)

	flag.BoolVar(&encode, "encode", false, "set to true to hide the data to the png file")
	flag.BoolVar(&decode, "decode", false, "set to true to extract the data from png file")
	flag.BoolVar(&dump, "dump", false, "set to true to dump data into stdout")
	//
	flag.StringVar(&pngImage, "png", "", "set the png image file path.")
	flag.StringVar(&message, "message", "", "set the message that you want to hide into the png file.")
	flag.StringVar(&file, "file", "", "set the file path that you want to hide it data into the png file.")
	flag.StringVar(&to, "to", "", "set the output file to write")

	flag.Parse()

	if pngImage == "" {
		log.Printf("no image specified")
		flag.PrintDefaults()
		return
	}

	png, err := readPNG(pngImage)
	if err != nil {
		log.Fatal(err)
	}

	if encode {
		if message != "" && to != "" {
			if err := Encode(png, []byte(message), to); err != nil {
				log.Fatal(err)
			}
			log.Printf("[+] %s written", to)
			return
		}

		if file != "" && to != "" {
			data, err := os.ReadFile(file)
			if err != nil {
				log.Fatal(err)
			}
			if err := Encode(png, data, to); err != nil {
				log.Fatal(err)
			}

			log.Printf("[+] %s written", to)
			return
		}
	}

	if decode {
		data, err := Decode(png)
		if err != nil {
			log.Fatal(err)
		}
		if to != "" {
			if err := os.WriteFile(to, data, 0666); err != nil {
				log.Fatal(err)
			}
			log.Printf("[+] %s written", to)
			return
		}
		if dump {
			fmt.Printf("DATA:\n %v", string(data))
			return
		}
	}

	flag.PrintDefaults()
}

func readPNG(filename string) (*PNG, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	header := new(PngHeader)
	if err := header.ReadHeader(fd); err != nil {
		return nil, err
	}

	ok, err := header.Validate()
	if err != nil {
		return nil, err
	}

	if !ok {
		// TODO: export this error into a global variable.
		return nil, fmt.Errorf("Not a PNG file")
	}

	png := &PNG{
		Header: *header,
	}

	if err := png.ReadChunks(fd); err != nil {
		return nil, err
	}

	return png, nil
}

// TODO: we still don't encrypt the data that we are hiding
func Encode(png *PNG, data []byte, outpngName string) error {
	m, err := NewPngMetadata(data)
	if err != nil {
		return err
	}

	// append the new chunk into the `len(chunks)-2`
	// NOTE: we can change this way of appending the new chunk in the future.
	png.Chunks = append(png.Chunks[:png.newChunkIndex+1], []*PngMetadata{m, png.Chunks[len(png.Chunks)-1]}...)

	// get new png data
	pngdata, err := png.Marshal()
	if err != nil {
		return err
	}

	return os.WriteFile(outpngName, pngdata, 0666)
}

func Decode(png *PNG) ([]byte, error) {
	c := png.Chunks[png.newChunkIndex]
	typ, err := c.Type.String()
	if err != nil {
		return nil, err
	}
	if typ != NewChunkTypeName {
		return nil, fmt.Errorf("couldn't find the png chunk")
	}

	// TODO: we need to decrypt the data here.
	return c.Data, nil
}
