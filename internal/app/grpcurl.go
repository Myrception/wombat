package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/google/shlex"
)

type multiString []string

func (s multiString) String() string {
	return strings.Join(s, ",")
}

func (s *multiString) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type grpcurlArguments struct {
	Target   string  `json:"target"`
	Method   string  `json:"method"`
	Metadata headers `json:"metadata"`
	Data     string  `json:"data"`
}

// shellQuote quotes s for a POSIX shell, so exported commands can be pasted
// into a terminal (and imported again) whatever the request data contains.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// parseGrpcurlCommand extracts the target, method, request data and metadata
// from a grpcurl command line. Connection flags are accepted but ignored; the
// workspace settings are used instead.
func parseGrpcurlCommand(command string) (*grpcurlArguments, error) {
	words, err := shlex.Split(command)
	if err != nil {
		return nil, fmt.Errorf("invalid grpcurl command: %v", err)
	}
	// A line continuation ("\" at the end of a line) splits into a lone
	// newline word.
	args := words[:0]
	for _, w := range words {
		if strings.TrimSpace(w) != "" {
			args = append(args, w)
		}
	}

	if len(args) == 0 {
		return nil, errors.New("empty grpcurl command")
	}

	if strings.ToLower(args[0]) != "grpcurl" {
		return nil, errors.New("invalid grpcurl command: must start with 'grpcurl'")
	}

	// ContinueOnError: the default (ExitOnError) would quit Wombat on any
	// flag it does not know.
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	// ignore flags
	_ = flags.Bool("help", false, "")
	_ = flags.Bool("version", false, "")
	_ = flags.Bool("plaintext", false, "")
	_ = flags.Bool("insecure", false, "")
	_ = flags.String("cacert", "", "")
	_ = flags.String("cert", "", "")
	_ = flags.String("key", "", "")
	_ = flags.Bool("expand-headers", false, "")
	_ = flags.String("authority", "", "")
	_ = flags.String("user-agent", "", "")
	_ = flags.Bool("allow-unknown-fields", false, "")
	_ = flags.Float64("connect-timeout", 0, "")
	_ = flags.Bool("format-error", false, "")
	_ = flags.Float64("keepalive-time", 0, "")
	_ = flags.Float64("max-time", 0, "")
	_ = flags.Int("max-msg-sz", 0, "")
	_ = flags.Bool("emit-defaults", false, "")
	_ = flags.String("protoset-out", "", "")
	_ = flags.String("proto-out-dir", "", "")
	_ = flags.Bool("msg-template", false, "")
	_ = flags.Bool("v", false, "")
	_ = flags.Bool("vv", false, "")
	_ = flags.String("servername", "", "")
	_ = flags.Bool("use-reflection", false, "")
	_ = flags.Bool("unix", false, "")
	_ = flags.Bool("alts", false, "")
	_ = flags.String("alts-handshaker-service", "", "")
	var altsAccounts multiString
	flags.Var(&altsAccounts, "alts-target-service-account", "")

	var data, format string
	flags.StringVar(&data, "d", "", "")
	flags.StringVar(&format, "format", "json", "")

	var protoset, protoFiles, importPaths, addlHeaders, rpcHeaders, reflHeaders multiString
	flags.Var(&addlHeaders, "H", "")
	flags.Var(&rpcHeaders, "rpc-header", "")
	flags.Var(&reflHeaders, "reflect-header", "")
	flags.Var(&protoset, "protoset", "")
	flags.Var(&protoFiles, "proto", "")
	flags.Var(&importPaths, "import-path", "")

	if err := flags.Parse(args[1:]); err != nil {
		return nil, fmt.Errorf("invalid grpcurl command: %v", err)
	}

	if format != "json" {
		return nil, fmt.Errorf("-format %s is not supported, only json requests can be imported", format)
	}
	if data == "@" {
		return nil, errors.New("-d @ reads the request from stdin; paste the request data into the command instead")
	}

	grpcurlArgs := flags.Args()
	if len(grpcurlArgs) != 2 {
		return nil, fmt.Errorf("invalid grpcurl command: expected an address and a method, got %d arguments", len(grpcurlArgs))
	}

	var metadata headers
	for _, h := range append(addlHeaders, rpcHeaders...) {
		// Same rules as grpcurl: split at the first colon only (values may
		// contain colons), and trim the space after it.
		key, val, ok := strings.Cut(h, ":")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid header %q: must be in the form 'name: value'", h)
		}
		metadata = append(metadata, header{
			Key: key,
			Val: strings.TrimSpace(val),
		})
	}

	return &grpcurlArguments{
		Target:   grpcurlArgs[0],
		Method:   grpcurlArgs[1],
		Data:     data,
		Metadata: metadata,
	}, nil
}
