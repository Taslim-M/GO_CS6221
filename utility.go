package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

func WriteMono(filename string, data []float64, sampleRate uint32) error {
	bitsPerSample := 16
	channels := 1

	outFile := &File{
		sampleRate,
		uint16(bitsPerSample),
		uint16(channels),
	}

	// []int to []bytes (assuming 16-bit samples)
	bytes := make([]byte, 2*len(data))
	for i, val := range data {
		start := i * 2
		binary.LittleEndian.PutUint16(bytes[start:start+2], uint16(val))
	}

	ofile, oerr := os.Create(filename)
	if oerr != nil {
		return oerr
	}

	err := outFile.WriteData(ofile, bytes)

	if err != nil {
		return err
	}

	return nil
}

type File struct {
	SampleRate      uint32
	SignificantBits uint16
	Channels        uint16
}

func (f *File) WriteData(w io.Writer, data []byte) (err error) {
	defer func() {
		if e, ok := recover().(error); ok {
			err = e
		}
	}()
	var buf bytes.Buffer
	writeFmt(&buf, f)
	writeChunk(&buf, "data", data)
	write(w, []byte("RIFF"))
	write(w, uint32(buf.Len()))
	write(w, []byte("WAVE"))
	write(w, buf.Bytes())
	return
}

func writeFmt(w io.Writer, f *File) (err error) {
	var b bytes.Buffer
	write(&b, uint16(1)) // uncompressed/PCM
	write(&b, f.Channels)
	write(&b, f.SampleRate)
	write(&b, uint32(f.Channels)*f.SampleRate*uint32(f.SignificantBits)/8) // bytes per second
	write(&b, f.SignificantBits/8*f.Channels)                              // block align
	write(&b, f.SignificantBits)
	return writeChunk(w, "fmt ", b.Bytes())
}

func writeChunk(w io.Writer, id string, data []byte) (err error) {
	if len(id) != 4 {
		panic(errors.New("invalid chunk id"))
	}
	write(w, []byte(id))
	write(w, uint32(len(data)))
	write(w, data)
	return
}

func write(w io.Writer, data interface{}) {
	if b, ok := data.([]byte); ok {
		for c := 0; c < len(b); {
			n, err := w.Write(b[c:])
			if err != nil {
				panic(err)
			}
			c += n
		}
		return
	}
	if err := binary.Write(w, binary.LittleEndian, data); err != nil {
		panic(err)
	}
}

func deleteFiles(filenames []string) {
	for _, filename := range filenames {
		err := os.Remove(filename)
		if err != nil {
			fmt.Printf("Failed to delete %s: %v\n", filename, err)
		} else {
			fmt.Printf("Deleted %s successfully.\n", filename)
		}
	}
}
