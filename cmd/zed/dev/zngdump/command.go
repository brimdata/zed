package zngdump

import (
	"bufio"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/brimdata/zed/cmd/zed/dev"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zcode"
)

var Cmd = &charm.Spec{
	Name:  "zngdump",
	Usage: "zngdump file",
	Short: "prints framing information of ZNG",
	Long: `
This command is used for test and debug of the ZNG format.`,
	New: New,
}

func init() {
	dev.Cmd.Add(Cmd)
}

type Command struct {
	*root.Command
	from int
	to   int
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.IntVar(&c.from, "from", 0, "seek range from")
	f.IntVar(&c.to, "to", 0, "seek range to")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zngdump: one file name argument required")
	}
	f := os.Stdin
	fname := args[0]
	if fname != "-" {
		var err error
		f, err = os.Open(fname)
		if err != nil {
			return err
		}
	}
	if c.from != 0 || c.to != 0 {
		chunk := io.NewSectionReader(f, int64(c.from), int64(c.to)-int64(c.from))
		_, err := io.Copy(os.Stdout, chunk)
		return err
	}
	r := &reader{reader: bufio.NewReader(f)}
	// We should output metadata as Zed instead of text and make
	// other improvements here.  See issue #3470.
	for {
		pos := r.pos
		code, err := r.ReadByte()
		if err != nil {
			return noEOF(err)
		}
		fmt.Printf("% 10d ", pos)
		if code == 0xff {
			fmt.Println("ff EOS")
			continue
		}
		if (code & 0x80) != 0 {
			return errors.New("zngio: encountered wrong version bit in framing")
		}
		switch typ := (code >> 4) & 3; typ {
		case 0:
			fmt.Printf("%02x TYPES ", code)
		case 1:
			fmt.Printf("%02x VALUES ", code)
		case 2:
			fmt.Printf("%02x CONTROL ", code)
		default:
			return fmt.Errorf("unknown ZNG message block type: %d", typ)
		}
		if (code & 0x40) != 0 {
			format, size, zlen, err := r.readComp(code)
			if err != nil {
				return noEOF(err)
			}
			fmt.Printf("comp fmt %d zlen %d size %d (net %d)\n", format, zlen, size, size-zlen)
		} else {
			size, err := r.readUncomp(code)
			if err != nil {
				return noEOF(err)
			}
			fmt.Printf("reg size %d\n", size)
		}
	}
}

type reader struct {
	reader *bufio.Reader
	pos    int64
}

func (r *reader) ReadByte() (byte, error) {
	code, err := r.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	r.pos++
	return code, nil
}

func (r *reader) readUncomp(code byte) (int, error) {
	size, err := r.decodeLength(code)
	if err != nil {
		return 0, err
	}
	return size, r.skip(size)
}

func (r *reader) readComp(code byte) (int, int, int, error) {
	zlen, err := r.decodeLength(code)
	if err != nil {
		return 0, 0, 0, err
	}
	format, err := r.ReadByte()
	if err != nil {
		return 0, 0, 0, err
	}
	size, err := r.uvarint()
	if err != nil {
		return 0, 0, 0, err
	}
	// The size of the compressed buffer needs to be adjusted by the
	// byte for the format and the variable-length bytes to encode
	// the original size.
	zlen -= 1 + zcode.SizeOfUvarint(uint64(size))
	err = r.skip(zlen)
	if err != nil && err != io.EOF {
	}
	return int(format), size, zlen, err
}

func (r *reader) skip(n int) error {
	if n > 25*1024*1024 {
		return fmt.Errorf("buffer length too big: %d", n)
	}
	got, err := r.reader.Discard(n)
	if n != got {
		return fmt.Errorf("short read: wanted to discard %d but got only %d", n, got)
	}
	r.pos += int64(n)
	return err
}

func (r *reader) decodeLength(code byte) (int, error) {
	v, err := r.uvarint()
	if err != nil {
		return 0, err
	}
	return (v << 4) | (int(code) & 0xf), nil
}

func (r *reader) uvarint() (int, error) {
	u64, err := binary.ReadUvarint(r)
	return int(u64), err
}

func noEOF(err error) error {
	if err == io.EOF {
		err = nil
	}
	return err
}
