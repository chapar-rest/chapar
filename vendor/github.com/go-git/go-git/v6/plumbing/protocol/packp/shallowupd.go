package packp

import (
	"bytes"
	"fmt"
	"io"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/format/pktline"
)

const (
	shallowLineLen   = 48
	unshallowLineLen = 50
)

type ShallowUpdate struct {
	Shallows   []plumbing.Hash
	Unshallows []plumbing.Hash
}

func (r *ShallowUpdate) Decode(reader io.Reader) error {
	var (
		p   []byte
		err error
		l   int
	)
	for {
		l, p, err = pktline.ReadLine(reader)
		if err != nil {
			break
		}

		line := bytes.TrimSpace(p)
		switch {
		case bytes.HasPrefix(line, shallow):
			err = r.decodeShallowLine(line)
		case bytes.HasPrefix(line, unshallow):
			err = r.decodeUnshallowLine(line)
		case l == pktline.Flush:
			return nil
		default:
			err = fmt.Errorf("unexpected shallow line: %q", line)
		}

		if err != nil {
			return err
		}
	}

	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func (r *ShallowUpdate) decodeShallowLine(line []byte) error {
	hash, err := r.decodeLine(line, shallow, shallowLineLen)
	if err != nil {
		return err
	}

	r.Shallows = append(r.Shallows, hash)
	return nil
}

func (r *ShallowUpdate) decodeUnshallowLine(line []byte) error {
	hash, err := r.decodeLine(line, unshallow, unshallowLineLen)
	if err != nil {
		return err
	}

	r.Unshallows = append(r.Unshallows, hash)
	return nil
}

func (r *ShallowUpdate) decodeLine(line, prefix []byte, expLen int) (plumbing.Hash, error) {
	if len(line) != expLen {
		return plumbing.ZeroHash, fmt.Errorf("malformed %s%q", prefix, line)
	}

	raw := string(line[expLen-40 : expLen])
	return plumbing.NewHash(raw), nil
}

func (r *ShallowUpdate) Encode(w io.Writer) error {
	for _, h := range r.Shallows {
		if _, err := pktline.Writef(w, "%s%s\n", shallow, h.String()); err != nil {
			return err
		}
	}

	for _, h := range r.Unshallows {
		if _, err := pktline.Writef(w, "%s%s\n", unshallow, h.String()); err != nil {
			return err
		}
	}

	return pktline.WriteFlush(w)
}
