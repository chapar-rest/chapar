// SPDX-License-Identifier: Unlicense OR MIT

//go:build linux && !android
// +build linux,!android

package explorer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/io/event"
	"github.com/godbus/dbus/v5"
)

// explorer opens file explorers using the xdg-desktop-portal dbus protocol
// defined here:
// https://flatpak.github.io/xdg-desktop-portal/#gdbus-org.freedesktop.portal.FileChooser
type explorer struct {
	X11Window uintptr
}

func newExplorer(w *app.Window) *explorer {
	return new(explorer)
}

func (e *Explorer) listenEvents(ev event.Event) {
	switch ev := ev.(type) {
	case app.X11ViewEvent:
		e.X11Window = ev.Window
	}
}

// randString generates a string of the form prefix+hexnumber, where hexnumber
// is the hex-encoded form of 16 bytes of cryptographically random data.
func randString(prefix string) (string, error) {
	var bytes [16]byte
	n, err := rand.Read(bytes[:])
	if err != nil {
		return "", fmt.Errorf("unable to generate random handle: %w", err)
	} else if n != len(bytes) {
		return "", fmt.Errorf("unable to read enough random data for handle")
	}
	return prefix + hex.EncodeToString(bytes[:]), nil
}

// extractURIsFromSignal locates the list of file URIs within the body of the
// signal and converts them to a slice of strings. If there were no URIs or
// if they are not a slice of strings, it returns the empty slice.
func extractURIsFromSignal(sig *dbus.Signal) []string {
	var uris []string
	for _, element := range sig.Body {
		asMap, ok := element.(map[string]dbus.Variant)
		if !ok {
			continue
		}
		urisVariant := asMap["uris"]
		uris, ok = urisVariant.Value().([]string)
		if !ok {
			return nil
		}
		break
	}
	return uris
}

// exportFile requests that a dialog be opened to write a file with the given
// name somewhere in the filesystem.
func (e *Explorer) exportFile(fileName string) (io.WriteCloser, error) {
	var filepath string
	if err := e.withDesktopPortal(func(conn *dbus.Conn, desktopPortal dbus.BusObject, config config) error {
		// Invoke the OpenFile method.
		requestHandle := ""
		err := desktopPortal.Call("org.freedesktop.portal.FileChooser.SaveFile", 0, config.parentWindow, "Choose Save Location", map[string]dbus.Variant{
			"handle_token": dbus.MakeVariant(config.handleToken),
			"current_name": dbus.MakeVariant(fileName),
		}).Store(&requestHandle)
		if err != nil {
			return fmt.Errorf("failed to call OpenFile: %w", err)
		}

		// Make sure we got the request object's path right. Update our subscription otherwise.
		if requestHandle != config.expectedRequestHandle {
			if err := conn.AddMatchSignal(dbus.WithMatchObjectPath(dbus.ObjectPath(requestHandle))); err != nil {
				return fmt.Errorf("failed to subscribe to request: %w", err)
			}
			// Reset signal handling.
			signals := make(chan *dbus.Signal, 1)
			conn.Signal(signals)
			config.signals = signals
		}

		// Wait for the response from the file dialog.
		response := <-config.signals
		uris := extractURIsFromSignal(response)

		// Error if no files were selected.
		if len(uris) < 1 {
			return ErrUserDecline
		}

		// Remove the protocol from the URI.
		parsedURL, err := url.Parse(uris[0])
		if err != nil {
			return fmt.Errorf("failed parsing file path %s: %w", uris[0], err)
		}
		filepath = parsedURL.Path
		return nil
	}); err != nil {
		return nil, err
	}
	return os.Create(filepath)
}

// sanitizeSenderName converts the dbusSenderName into the form required in the
// response object path.
// https://flatpak.github.io/xdg-desktop-portal/#gdbus-org.freedesktop.portal.Request
func sanitizeSenderName(dbusSenderName string) string {
	return strings.TrimPrefix(strings.ReplaceAll(dbusSenderName, ".", "_"), ":")
}

type config struct {
	parentWindow          string
	expectedRequestHandle string
	handleToken           string
	signals               chan *dbus.Signal
}

// withDesktopPortal connects to the session dbus and finds the service
// implementing the freedesktop.org portals. It accepts a function that
// it will run with access to the connection, portal, and a set of
// parameters that are useful for making requests against the portal.
func (e *Explorer) withDesktopPortal(work func(conn *dbus.Conn, desktopPortal dbus.BusObject, config config) error) error {
	// Connect to the session bus.
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("unable to connect to session bus: %w", err)
	}
	defer conn.Close()
	// Figure out our own connection name.
	senderName := sanitizeSenderName(conn.Names()[0])

	// Determine parameters for the methods we will call.
	obj := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	parentWindow := ""
	if e.X11Window != 0 {
		parentWindow = "x11:" + fmt.Sprintf("%x", e.X11Window)
	}
	handle, err := randString("giox")
	if err != nil {
		return fmt.Errorf("unable to export file: %w", err)
	}

	// Predict the request object's path.
	expectedRequestHandle := fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", senderName, handle)

	// Subscribe to signals on the request object's path before submitting the request to avoid
	// race conditions.
	if err := conn.AddMatchSignal(dbus.WithMatchObjectPath(dbus.ObjectPath(expectedRequestHandle))); err != nil {
		return fmt.Errorf("failed to subscribe to request: %w", err)
	}
	// Prepare for signal handling.
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)

	// Perform some work while connected.
	if err := work(conn, obj, config{
		parentWindow:          parentWindow,
		expectedRequestHandle: expectedRequestHandle,
		handleToken:           handle,
		signals:               signals,
	}); err != nil {
		return err
	}
	return nil
}

// makeFilter constructs a file type filter appropriate for the provided extensions
// and encodes it as a dbus variant.
func makeFilter(extensions []string) dbus.Variant {
	// Resolve the provided extensions to their corresponding mime types.
	type mimetype struct {
		// Field names _must_ be exported so that they are available via reflection,
		// otherwise they will not be sent.
		Kind uint
		Name string
	}
	mimes := make([]mimetype, len(extensions))
	for i := range extensions {
		ext := extensions[i]
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		mt := mime.TypeByExtension(ext)
		if mt != "" {
			mimes[i] = mimetype{
				Kind: 1,
				Name: mt,
			}
		} else {
			mimes[i] = mimetype{
				Kind: 0,
				Name: "*" + ext,
			}
		}
	}

	// Transform the filter into its dbus variant form.
	filter := []struct {
		// Field names must be exported so they are available via reflection, otherwise
		// they will not be sent.
		Name  string
		Value []mimetype
	}{
		{
			Name:  "Filter",
			Value: mimes,
		},
	}
	return dbus.MakeVariantWithSignature(filter, dbus.ParseSignatureMust("a(sa(us))"))
}

type configOpen struct {
	label      string
	extensions []string
	multi      bool
	dir        bool
}

// importFile opens a file picker to choose a file.
func (e *Explorer) importFile(extensions ...string) (io.ReadCloser, error) {
	vs, err := e.open(configOpen{
		label:      "Choose File",
		extensions: extensions,
	})
	if err != nil {
		return nil, err
	}
	return vs[0], nil
}

// importFiles opens a multi-file picker to choose multiple files.
func (e *Explorer) importFiles(extensions ...string) ([]io.ReadCloser, error) {
	vs, err := e.open(configOpen{
		label:      "Choose Files",
		extensions: extensions,
		multi:      true,
	})
	if err != nil {
		return nil, err
	}
	return vs, nil
}

func (e *Explorer) open(cfg configOpen) ([]io.ReadCloser, error) {
	var filepaths []string
	if err := e.withDesktopPortal(func(conn *dbus.Conn, desktopPortal dbus.BusObject, config config) error {
		// Invoke the OpenFile method.
		requestHandle := ""
		options := map[string]dbus.Variant{
			"handle_token": dbus.MakeVariant(config.handleToken),
			"multiple":     dbus.MakeVariant(cfg.multi),
			"directory":    dbus.MakeVariant(cfg.dir),
		}
		if len(cfg.extensions) > 0 {
			options["filters"] = makeFilter(cfg.extensions)
		}
		err := desktopPortal.Call("org.freedesktop.portal.FileChooser.OpenFile", 0, config.parentWindow, cfg.label, options).Store(&requestHandle)
		if err != nil {
			return fmt.Errorf("failed to call OpenFile: %w", err)
		}

		// Make sure we got the request object's path right. Update our subscription otherwise.
		if requestHandle != config.expectedRequestHandle {
			if err := conn.AddMatchSignal(dbus.WithMatchObjectPath(dbus.ObjectPath(requestHandle))); err != nil {
				return fmt.Errorf("failed to subscribe to request: %w", err)
			}
			// Reset signal handling.
			signals := make(chan *dbus.Signal, 1)
			conn.Signal(signals)
			config.signals = signals
		}

		// Wait for the response from the file dialog.
		response := <-config.signals
		uris := extractURIsFromSignal(response)

		// Error if no files were selected.
		if len(uris) < 1 {
			return ErrUserDecline
		}

		filepaths = make([]string, len(uris))
		for i, uri := range uris {
			// Remove the protocol from the URI.
			parsedURL, err := url.Parse(uri)
			if err != nil {
				return fmt.Errorf("failed parsing file path %s: %w", uri, err)
			}
			filepaths[i] = parsedURL.Path
		}
		return nil
	}); err != nil {
		return nil, err
	}

	rcs := make([]io.ReadCloser, 0, len(filepaths))
	for _, fname := range filepaths {
		rc, err := os.Open(fname)
		if err != nil {
			for _, rc := range rcs {
				_ = rc.Close()
			}
			return nil, err
		}
		rcs = append(rcs, rc)
	}

	return rcs, nil
}
