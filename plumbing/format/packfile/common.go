package packfile

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"sync"

	"github.com/goabstract/go-git/v5/plumbing/progress"
	"github.com/goabstract/go-git/v5/plumbing/storer"
	"github.com/goabstract/go-git/v5/utils/ioutil"
)

var signature = []byte{'P', 'A', 'C', 'K'}

const (
	// VersionSupported is the packfile version supported by this package
	VersionSupported uint32 = 2

	firstLengthBits = uint8(4)   // the first byte into object header has 4 bits to store the length
	lengthBits      = uint8(7)   // subsequent bytes has 7 bits to store the length
	maskFirstLength = 15         // 0000 1111
	maskContinue    = 0x80       // 1000 0000
	maskLength      = uint8(127) // 0111 1111
	maskType        = uint8(112) // 0111 0000
)

// UpdateObjectStorage updates the storer with the objects in the given
// packfile.
func UpdateObjectStorage(s storer.Storer, packfile io.Reader, pr *progress.Reporter) error {
	reader := packfile
	var pc *progress.Collector = nil
	if pr != nil {
		pc = progress.NewCollector(packfile, pr)
		reader = pc
	}
	if pw, ok := s.(storer.PackfileWriter); ok {
		if pc != nil {
			ctx, cancel := context.WithCancel(context.Background())
			pc.Start(ctx)
			defer cancel()
		}

		return WritePackfileToObjectStorage(pw, reader, pc)
	}

	if pc != nil {
		ctx, cancel := context.WithCancel(context.Background())
		pc.Start(ctx)
		defer cancel()
	}
	p, err := NewParserWithStorage(NewScanner(reader), s)
	p.ProgressCollector = pc
	if err != nil {
		return err
	}

	_, err = p.Parse()
	return err
}

// WritePackfileToObjectStorage writes all the packfile objects into the given
// object storage.
func WritePackfileToObjectStorage(
	sw storer.PackfileWriter,
	packfile io.Reader,
	pc *progress.Collector,
) (err error) {
	w, err := sw.PackfileWriterWithProgress(pc)
	if err != nil {
		return err
	}
	defer ioutil.CheckClose(w, &err)

	var n int64
	n, err = io.Copy(w, packfile)
	if err == nil && n == 0 {
		return ErrEmptyPackfile
	}

	return err
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

var zlibInitBytes = []byte{0x78, 0x9c, 0x01, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0x00, 0x01}

var zlibReaderPool = sync.Pool{
	New: func() interface{} {
		r, _ := zlib.NewReader(bytes.NewReader(zlibInitBytes))
		return r
	},
}
