package importer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

// curlArgFlags are the curl options that take a value but have no bearing on
// the request Chapar stores (output, timeouts, TLS and proxy settings, ...).
// They are listed so their value is skipped rather than read as the URL.
var curlArgFlags = map[string]bool{
	"-o": true, "--output": true, "-m": true, "--max-time": true, "--connect-timeout": true,
	"-x": true, "--proxy": true, "-U": true, "--proxy-user": true, "--noproxy": true,
	"--cacert": true, "--capath": true, "-E": true, "--cert": true, "--cert-type": true,
	"--key": true, "--key-type": true, "--pass": true, "--ciphers": true,
	"-w": true, "--write-out": true, "-c": true, "--cookie-jar": true, "-D": true, "--dump-header": true,
	"--retry": true, "--retry-delay": true, "--retry-max-time": true, "-r": true, "--range": true,
	"-K": true, "--config": true, "--resolve": true, "--connect-to": true, "--limit-rate": true,
	"-y": true, "--speed-time": true, "-Y": true, "--speed-limit": true, "--max-redirs": true,
	"-z": true, "--time-cond": true, "--interface": true, "--unix-socket": true, "--abstract-unix-socket": true,
	"--aws-sigv4": true, "--trace": true, "--trace-ascii": true, "--stderr": true, "-Q": true, "--quote": true,
	"--local-port": true, "--dns-servers": true, "--expect100-timeout": true, "--keepalive-time": true,
	"--max-filesize": true, "--proto": true, "--proto-redir": true, "--tls-max": true, "--variable": true,
}

// curlShortArgs are the single-letter options that take a value, so a cluster
// like -sXPOST knows where the flags stop and the value starts.
const curlShortArgs = "XHdbuAeFTomxUEwcDrKyYzQ"

// IsCurlCommand reports whether s reads as a curl command line.
func IsCurlCommand(s string) bool {
	s = strings.TrimSpace(s)
	return s == "curl" || strings.HasPrefix(s, "curl ") || strings.HasPrefix(s, "curl\t") ||
		strings.HasPrefix(s, "curl\\") || strings.HasPrefix(s, "curl\n")
}

// ParseCurl turns a curl command line, as copied from API docs or a browser's
// "Copy as cURL", into an HTTP request. The request keeps the default name so
// Chapar labels it from its URL.
func ParseCurl(cmd string) (*domain.Request, error) {
	args, err := splitCurlArgs(cmd)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 || args[0] != "curl" {
		return nil, errors.New("not a curl command: it must start with curl")
	}

	var (
		c       curlCommand
		rawURL  string
		argsEnd bool
	)
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if argsEnd || !strings.HasPrefix(arg, "-") || arg == "-" {
			if rawURL == "" {
				rawURL = arg
			}
			continue
		}
		if arg == "--" {
			argsEnd = true
			continue
		}

		// value returns the option's value: attached (-XPOST) or the next arg.
		value := func(name, attached string) (string, error) {
			if attached != "" {
				return attached, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("curl option %s needs a value", name)
			}
			i++
			return args[i], nil
		}

		if strings.HasPrefix(arg, "--") {
			name := arg
			if !c.takesValue(name) {
				c.flag(name)
				continue
			}
			v, err := value(name, "")
			if err != nil {
				return nil, err
			}
			if name == "--url" {
				if rawURL == "" {
					rawURL = v
				}
				continue
			}
			c.option(name, v)
			continue
		}

		// A cluster of short options: -sSL, -XPOST, -sH 'a: b'.
		for j := 1; j < len(arg); j++ {
			name := "-" + string(arg[j])
			if !strings.ContainsRune(curlShortArgs, rune(arg[j])) {
				c.flag(name)
				continue
			}
			v, err := value(name, arg[j+1:])
			if err != nil {
				return nil, err
			}
			c.option(name, v)
			break
		}
	}

	if rawURL == "" {
		return nil, errors.New("curl command has no URL")
	}
	return c.request(rawURL)
}

// ImportCurl parses a curl command and saves it as a standalone request.
func ImportCurl(cmd string, repo repository.RepositoryV2) (*domain.Request, error) {
	req, err := ParseCurl(cmd)
	if err != nil {
		return nil, err
	}
	if err := repo.CreateRequest(req, nil); err != nil {
		return nil, fmt.Errorf("error saving request: %w", err)
	}
	return req, nil
}

type curlDataKind int

const (
	curlData       curlDataKind = iota // -d, --data-raw, --data-binary: sent as written
	curlDataEncode                     // --data-urlencode: the value part gets encoded
)

type curlDataPart struct {
	kind  curlDataKind
	value string
}

type curlCommand struct {
	method    string
	headers   []domain.KeyValue
	data      []curlDataPart
	dataFile  string // -d @file / --data-binary @file / -T file
	upload    bool   // -T: PUT by default
	json      bool   // --json
	form      []domain.FormField
	get       bool // -G: data goes in the query string
	head      bool // -I
	basicAuth *domain.BasicAuth
	token     string
}

func (c *curlCommand) takesValue(name string) bool {
	switch name {
	case "--request", "--header", "--data", "--data-ascii", "--data-raw", "--data-binary",
		"--data-urlencode", "--json", "--form", "--form-string", "--user", "--user-agent",
		"--referer", "--cookie", "--url", "--upload-file", "--oauth2-bearer":
		return true
	}
	return curlArgFlags[name]
}

// flag records a curl option that takes no value.
func (c *curlCommand) flag(name string) {
	switch name {
	case "-G", "--get":
		c.get = true
	case "-I", "--head":
		c.head = true
	}
}

// option records a curl option and its value.
func (c *curlCommand) option(name, v string) {
	switch name {
	case "-X", "--request":
		c.method = strings.ToUpper(v)
	case "-H", "--header":
		c.header(v)
	case "-A", "--user-agent":
		c.addHeader("User-Agent", v)
	case "-e", "--referer":
		c.addHeader("Referer", v)
	case "-b", "--cookie":
		// Without "=" the value names a cookie file, which has no request form.
		if strings.Contains(v, "=") {
			c.addHeader("Cookie", v)
		}
	case "-u", "--user":
		user, pass, _ := strings.Cut(v, ":")
		c.basicAuth = &domain.BasicAuth{Username: user, Password: pass}
	case "--oauth2-bearer":
		c.token = v
	case "-d", "--data", "--data-ascii", "--data-binary":
		if strings.HasPrefix(v, "@") {
			c.dataFile = v[1:]
			return
		}
		c.data = append(c.data, curlDataPart{kind: curlData, value: v})
	case "--data-raw":
		c.data = append(c.data, curlDataPart{kind: curlData, value: v})
	case "--data-urlencode":
		c.data = append(c.data, curlDataPart{kind: curlDataEncode, value: v})
	case "--json":
		c.json = true
		c.data = append(c.data, curlDataPart{kind: curlData, value: v})
	case "-F", "--form", "--form-string":
		c.formField(v, name == "--form-string")
	case "-T", "--upload-file":
		c.upload = true
		c.dataFile = v
	}
}

// header adds a -H value. "Name;" sends the header empty; "Name:" with no
// value tells curl to drop a header it would add itself, so it is skipped.
func (c *curlCommand) header(v string) {
	if name, ok := strings.CutSuffix(strings.TrimSpace(v), ";"); ok && !strings.Contains(name, ":") {
		c.addHeader(name, "")
		return
	}
	name, value, ok := strings.Cut(v, ":")
	if !ok {
		return
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	c.addHeader(strings.TrimSpace(name), value)
}

func (c *curlCommand) addHeader(name, value string) {
	c.headers = append(c.headers, domain.KeyValue{ID: uuid.NewString(), Key: name, Value: value, Enable: true})
}

func (c *curlCommand) hasHeader(name string) bool {
	for _, h := range c.headers {
		if strings.EqualFold(h.Key, name) {
			return true
		}
	}
	return false
}

func (c *curlCommand) headerValue(name string) string {
	for _, h := range c.headers {
		if strings.EqualFold(h.Key, name) {
			return h.Value
		}
	}
	return ""
}

// formField adds a -F value: name=value, or name=@path for a file, with any
// ;type= or ;filename= attributes dropped.
func (c *curlCommand) formField(v string, literal bool) {
	name, value, _ := strings.Cut(v, "=")
	field := domain.FormField{ID: uuid.NewString(), Key: name, Type: domain.FormFieldTypeText, Enable: true}
	if !literal && strings.HasPrefix(value, "@") {
		path, _, _ := strings.Cut(value[1:], ";")
		field.Type = domain.FormFieldTypeFile
		field.Files = []string{strings.Trim(path, `"`)}
	} else {
		if !literal {
			value = strings.TrimPrefix(value, "<")
			value, _, _ = strings.Cut(value, ";type=")
		}
		field.Value = value
	}
	c.form = append(c.form, field)
}

// joinedData is the body curl sends for the -d family: each part joined by &.
func (c *curlCommand) joinedData() string {
	parts := make([]string, 0, len(c.data))
	for _, d := range c.data {
		if d.kind == curlDataEncode {
			parts = append(parts, encodeDataPart(d.value))
			continue
		}
		parts = append(parts, d.value)
	}
	return strings.Join(parts, "&")
}

// encodeDataPart applies curl's --data-urlencode rules: "content" and
// "=content" encode the whole content, "name=content" only the content.
func encodeDataPart(v string) string {
	name, content, ok := strings.Cut(v, "=")
	if !ok {
		return url.QueryEscape(v)
	}
	if name == "" {
		return url.QueryEscape(content)
	}
	return name + "=" + url.QueryEscape(content)
}

func (c *curlCommand) hasBody() bool {
	return len(c.data) > 0 || c.dataFile != "" || len(c.form) > 0
}

func (c *curlCommand) request(rawURL string) (*domain.Request, error) {
	if !strings.Contains(rawURL, "://") && !strings.HasPrefix(rawURL, "{{") {
		rawURL = "http://" + rawURL
	}
	if c.get && len(c.data) > 0 {
		sep := "?"
		if strings.Contains(rawURL, "?") {
			sep = "&"
		}
		rawURL += sep + c.joinedData()
		c.data = nil
	}

	method := c.method
	switch {
	case method != "":
	case c.head:
		method = domain.RequestMethodHEAD
	case c.upload:
		method = domain.RequestMethodPUT
	case c.hasBody() && !c.get:
		method = domain.RequestMethodPOST
	default:
		method = domain.RequestMethodGET
	}

	if c.json {
		if !c.hasHeader("Content-Type") {
			c.addHeader("Content-Type", "application/json")
		}
		if !c.hasHeader("Accept") {
			c.addHeader("Accept", "application/json")
		}
	}

	req := domain.NewHTTPRequest(domain.DefaultRequestName)
	spec := req.Spec.HTTP
	spec.Method = method
	spec.URL = rawURL
	// body may add a Content-Type, so it runs before headers are copied.
	spec.Request.Body = c.body()
	spec.Request.Headers = c.headers
	if spec.Request.Headers == nil {
		spec.Request.Headers = []domain.KeyValue{}
	}
	base, query, _ := strings.Cut(rawURL, "?")
	spec.Request.QueryParams = domain.ParseQueryParams(query)
	spec.Request.PathParams = domain.ParsePathParams(base)

	switch {
	case c.basicAuth != nil:
		spec.Request.Auth = domain.Auth{Type: domain.AuthTypeBasic, BasicAuth: c.basicAuth}
	case c.token != "":
		spec.Request.Auth = domain.Auth{Type: domain.AuthTypeToken, TokenAuth: &domain.TokenAuth{Token: c.token}}
	}

	req.SetDefaultValues()
	return req, nil
}

func (c *curlCommand) body() domain.Body {
	switch {
	case len(c.form) > 0:
		return domain.Body{Type: domain.RequestBodyTypeFormData, FormData: domain.FormData{Fields: c.form}}
	case c.dataFile != "":
		return domain.Body{Type: domain.RequestBodyTypeBinary, BinaryFilePath: c.dataFile}
	case len(c.data) == 0:
		return domain.Body{Type: domain.RequestBodyTypeNone}
	}

	data := c.joinedData()
	ct := strings.ToLower(c.headerValue("Content-Type"))
	switch {
	case c.json || strings.Contains(ct, "json"):
		return domain.Body{Type: domain.RequestBodyTypeJSON, Data: data}
	case strings.Contains(ct, "xml"):
		return domain.Body{Type: domain.RequestBodyTypeXML, Data: data}
	case strings.Contains(ct, "x-www-form-urlencoded"):
		if kv, ok := parseURLEncoded(data); ok {
			return domain.Body{Type: domain.RequestBodyTypeUrlencoded, URLEncoded: kv}
		}
	case ct != "":
		return domain.Body{Type: domain.RequestBodyTypeText, Data: data}
	}

	// No Content-Type: curl sends the data as a form, but docs often omit
	// the header on a JSON body, so pick what the data looks like.
	if trimmed := strings.TrimSpace(data); (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) && json.Valid([]byte(trimmed)) {
		if !c.hasHeader("Content-Type") {
			c.addHeader("Content-Type", "application/json")
		}
		return domain.Body{Type: domain.RequestBodyTypeJSON, Data: data}
	}
	if kv, ok := parseURLEncoded(data); ok {
		return domain.Body{Type: domain.RequestBodyTypeUrlencoded, URLEncoded: kv}
	}
	return domain.Body{Type: domain.RequestBodyTypeText, Data: data}
}

// parseURLEncoded splits a form body into fields, in order. It fails when a
// part has no "=", which means the body is not a form after all.
func parseURLEncoded(data string) ([]domain.KeyValue, bool) {
	var out []domain.KeyValue
	for _, part := range strings.Split(data, "&") {
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok || k == "" {
			return nil, false
		}
		if dk, err := url.QueryUnescape(k); err == nil {
			k = dk
		}
		if dv, err := url.QueryUnescape(v); err == nil {
			v = dv
		}
		out = append(out, domain.KeyValue{ID: uuid.NewString(), Key: k, Value: v, Enable: true})
	}
	return out, len(out) > 0
}

// splitCurlArgs splits a command line the way a POSIX shell would for the
// quoting curl commands use: '...', "...", $'...', backslash escapes and
// line continuations. Windows cmd continuations (^ at line end) are folded too.
func splitCurlArgs(cmd string) ([]string, error) {
	var (
		args  []string
		cur   strings.Builder
		inArg bool
	)
	s := []rune(strings.TrimSpace(cmd))
	flush := func() {
		if inArg {
			args = append(args, cur.String())
			cur.Reset()
			inArg = false
		}
	}
	for i := 0; i < len(s); i++ {
		r := s[i]
		switch {
		case r == '\\':
			if i+1 < len(s) && (s[i+1] == '\n' || s[i+1] == '\r') {
				// Line continuation: skip the newline (and \r\n).
				i++
				if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
					i++
				}
				continue
			}
			if i+1 < len(s) {
				i++
				cur.WriteRune(s[i])
				inArg = true
			}
		case r == '^' && i+1 < len(s) && (s[i+1] == '\n' || s[i+1] == '\r'):
			i++
			if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
				i++
			}
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		case r == '\'':
			end := indexRune(s, i+1, '\'')
			if end < 0 {
				return nil, errors.New("unterminated ' quote in curl command")
			}
			cur.WriteString(string(s[i+1 : end]))
			inArg = true
			i = end
		case r == '$' && i+1 < len(s) && s[i+1] == '\'':
			n, err := readANSIQuoted(s, i+2, &cur)
			if err != nil {
				return nil, err
			}
			inArg = true
			i = n
		case r == '"':
			n, err := readDoubleQuoted(s, i+1, &cur)
			if err != nil {
				return nil, err
			}
			inArg = true
			i = n
		default:
			cur.WriteRune(r)
			inArg = true
		}
	}
	flush()
	return args, nil
}

func indexRune(s []rune, from int, r rune) int {
	for i := from; i < len(s); i++ {
		if s[i] == r {
			return i
		}
	}
	return -1
}

// readDoubleQuoted copies a "..." string starting after the opening quote and
// returns the index of the closing quote. Only \" \\ \$ \` and a newline are
// escapes inside double quotes, as in a shell.
func readDoubleQuoted(s []rune, i int, out *strings.Builder) (int, error) {
	for ; i < len(s); i++ {
		switch s[i] {
		case '"':
			return i, nil
		case '\\':
			if i+1 < len(s) {
				switch s[i+1] {
				case '"', '\\', '$', '`':
					i++
					out.WriteRune(s[i])
					continue
				case '\n':
					i++
					continue
				}
			}
			out.WriteRune('\\')
		default:
			out.WriteRune(s[i])
		}
	}
	return 0, errors.New(`unterminated " quote in curl command`)
}

// readANSIQuoted copies a bash $'...' string, which browsers use when the body
// has quotes or newlines, and returns the index of the closing quote.
func readANSIQuoted(s []rune, i int, out *strings.Builder) (int, error) {
	for ; i < len(s); i++ {
		r := s[i]
		if r == '\'' {
			return i, nil
		}
		if r != '\\' || i+1 >= len(s) {
			out.WriteRune(r)
			continue
		}
		i++
		switch s[i] {
		case 'n':
			out.WriteRune('\n')
		case 't':
			out.WriteRune('\t')
		case 'r':
			out.WriteRune('\r')
		case '0':
			out.WriteRune(0)
		case 'x', 'u', 'U':
			size := map[rune]int{'x': 2, 'u': 4, 'U': 8}[s[i]]
			j := i + 1
			for j < len(s) && j < i+1+size && isHex(s[j]) {
				j++
			}
			if j == i+1 {
				out.WriteRune('\\')
				out.WriteRune(s[i])
				continue
			}
			n, _ := strconv.ParseUint(string(s[i+1:j]), 16, 32)
			if s[i] == 'x' {
				// \xHH is a byte; UTF-8 text arrives as a run of them.
				out.WriteByte(byte(n))
			} else {
				out.WriteRune(rune(n))
			}
			i = j - 1
		default:
			// \\ \' \" and anything unknown stand for the character itself.
			out.WriteRune(s[i])
		}
	}
	return 0, errors.New("unterminated $' quote in curl command")
}

func isHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}
