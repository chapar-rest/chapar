package packp

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/format/pktline"
)

var (
	shallowLineLength       = len(shallow) + hashSize
	minCommandLength        = hashSize*2 + 2 + 1
	minCommandAndCapsLength = minCommandLength + 1
)

var (
	ErrEmpty                        = errors.New("empty update-request message")
	errNoCommands                   = errors.New("unexpected EOF before any command")
	errMissingCapabilitiesDelimiter = errors.New("capabilities delimiter not found")
	errNoFlush                      = errors.New("unexpected EOF before flush line")
)

func errMalformedRequest(reason string) error {
	return fmt.Errorf("malformed request: %s", reason)
}

func errInvalidHashSize(got int) error {
	return fmt.Errorf("invalid hash size: expected %d, got %d",
		hashSize, got)
}

func errInvalidHash(hash string) error {
	return fmt.Errorf("invalid hash: %s", hash)
}

func errInvalidShallowLineLength(got int) error {
	return errMalformedRequest(fmt.Sprintf(
		"invalid shallow line length: expected %d, got %d",
		shallowLineLength, got))
}

func errInvalidCommandCapabilitiesLineLength(got int) error {
	return errMalformedRequest(fmt.Sprintf(
		"invalid command and capabilities line length: expected at least %d, got %d",
		minCommandAndCapsLength, got))
}

func errInvalidCommandLineLength(got int) error {
	return errMalformedRequest(fmt.Sprintf(
		"invalid command line length: expected at least %d, got %d",
		minCommandLength, got))
}

func errInvalidShallowObjId(err error) error {
	return errMalformedRequest(
		fmt.Sprintf("invalid shallow object id: %s", err.Error()))
}

func errInvalidOldObjId(err error) error {
	return errMalformedRequest(
		fmt.Sprintf("invalid old object id: %s", err.Error()))
}

func errInvalidNewObjId(err error) error {
	return errMalformedRequest(
		fmt.Sprintf("invalid new object id: %s", err.Error()))
}

func errMalformedCommand(err error) error {
	return errMalformedRequest(fmt.Sprintf(
		"malformed command: %s", err.Error()))
}

// Decode reads the next update-request message form the reader and wr
func (req *UpdateRequests) Decode(r io.Reader) error {
	var rc io.ReadCloser
	var ok bool
	rc, ok = r.(io.ReadCloser)
	if !ok {
		rc = io.NopCloser(r)
	}

	d := &updReqDecoder{r: rc, pr: r}
	return d.Decode(req)
}

type updReqDecoder struct {
	r   io.ReadCloser
	pr  io.Reader
	req *UpdateRequests

	payload []byte
	length  int // length of the pktline payload
}

func (d *updReqDecoder) Decode(req *UpdateRequests) error {
	d.req = req
	funcs := []func() error{
		d.scanLine,
		d.decodeShallow,
		d.decodeCommands,
		d.decodeFlush,
		req.validate,
	}

	for _, f := range funcs {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

func (d *updReqDecoder) readLine(e error) error {
	l, p, err := pktline.ReadLine(d.pr)
	if errors.Is(err, io.EOF) {
		return e
	}
	if err != nil {
		return err
	}

	d.payload = p
	d.length = l

	return nil
}

func (d *updReqDecoder) scanLine() error {
	return d.readLine(ErrEmpty)
}

func (d *updReqDecoder) decodeShallow() error {
	b := d.payload

	if !bytes.HasPrefix(b, shallowNoSp) {
		return nil
	}

	if len(b) != shallowLineLength {
		return errInvalidShallowLineLength(len(b))
	}

	h, err := parseHash(string(b[len(shallow):]))
	if err != nil {
		return errInvalidShallowObjId(err)
	}

	if err := d.readLine(errNoCommands); err != nil {
		return err
	}

	d.req.Shallow = &h

	return nil
}

func (d *updReqDecoder) decodeCommands() error {
	// Process the first line which contains both command and capabilities
	payload := d.payload

	// The first line must contain capabilities separated by a null byte
	sep := bytes.IndexByte(payload, 0)
	if sep == -1 {
		return errMissingCapabilitiesDelimiter
	}

	if len(payload) < minCommandAndCapsLength {
		return errInvalidCommandCapabilitiesLineLength(len(payload))
	}

	// Extract and decode capabilities (everything after the null byte)
	if err := d.req.Capabilities.Decode(payload[sep+1:]); err != nil {
		return err
	}

	// Extract the command (everything before the null byte)
	payload = payload[:sep]

	// Read and process commands
	for {
		// Parse and add the command
		cmd, err := parseCommand(payload)
		if err != nil {
			return err
		}
		d.req.Commands = append(d.req.Commands, cmd)

		// Read the next line
		if err := d.readLine(errNoFlush); err != nil {
			return err
		}
		payload = d.payload

		// Stop reading once we reach the flush line
		if d.length == pktline.Flush {
			return nil
		}
	}
}

func (d *updReqDecoder) decodeFlush() error {
	// We should always have a flush line at the end of the request.
	if len(d.payload) != 0 || d.length != pktline.Flush {
		return errMalformedRequest("unexpected data after flush")
	}

	return nil
}

func parseCommand(b []byte) (*Command, error) {
	if len(b) < minCommandLength {
		return nil, errInvalidCommandLineLength(len(b))
	}

	var (
		os, ns string
		n      plumbing.ReferenceName
	)
	if _, err := fmt.Sscanf(string(b), "%s %s %s", &os, &ns, &n); err != nil {
		return nil, errMalformedCommand(err)
	}

	oh, err := parseHash(os)
	if err != nil {
		return nil, errInvalidOldObjId(err)
	}

	nh, err := parseHash(ns)
	if err != nil {
		return nil, errInvalidNewObjId(err)
	}

	return &Command{Old: oh, New: nh, Name: n}, nil
}

func parseHash(s string) (plumbing.Hash, error) {
	if len(s) != hashSize {
		return plumbing.ZeroHash, errInvalidHashSize(len(s))
	}

	h, ok := plumbing.FromHex(s)
	if !ok {
		return plumbing.ZeroHash, errInvalidHash(s)
	}

	return h, nil
}
