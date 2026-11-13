package app

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseGrpcurlCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    *grpcurlArguments
		wantErr string
	}{
		{
			name:    "simple",
			command: `grpcurl -d '{"name":"World"}' -plaintext localhost:50051 greeter.Greeter/SayHello`,
			want:    &grpcurlArguments{Target: "localhost:50051", Method: "greeter.Greeter/SayHello", Data: `{"name":"World"}`},
		},
		{
			name: "multi line with headers",
			command: "grpcurl -d '{}' \\\n" +
				"    -H 'authorization: Bearer abc' \\\n" +
				"    -rpc-header 'x-time:12:30' \\\n" +
				"    -reflect-header 'x-ignored: 1' \\\n" +
				"    -plaintext \\\n" +
				"    localhost:50051 greeter.Greeter/SayHello",
			want: &grpcurlArguments{
				Target: "localhost:50051", Method: "greeter.Greeter/SayHello", Data: "{}",
				Metadata: headers{{Key: "authorization", Val: "Bearer abc"}, {Key: "x-time", Val: "12:30"}},
			},
		},
		{
			name:    "escaped quote in data",
			command: `grpcurl -d '{"name":"O'\''Brien"}' localhost:50051 greeter.Greeter/SayHello`,
			want:    &grpcurlArguments{Target: "localhost:50051", Method: "greeter.Greeter/SayHello", Data: `{"name":"O'Brien"}`},
		},
		{
			name:    "explicit json format and flag forms",
			command: `grpcurl -format=json --plaintext=true -max-time 5 -d={} localhost:1 a.B/C`,
			want:    &grpcurlArguments{Target: "localhost:1", Method: "a.B/C", Data: "{}"},
		},
		{name: "empty", command: "  ", wantErr: "empty grpcurl command"},
		{name: "not grpcurl", command: "curl localhost", wantErr: "must start with 'grpcurl'"},
		{name: "unbalanced quote", command: `grpcurl -d '{"a":1} localhost:1 a.B/C`, wantErr: "closing quote"},
		// Used to call os.Exit and quit Wombat.
		{name: "unknown flag", command: `grpcurl -no-such-flag localhost:1 a.B/C`, wantErr: "flag provided but not defined: -no-such-flag"},
		{name: "text format", command: `grpcurl -format text -d 'name: "x"' localhost:1 a.B/C`, wantErr: "-format text is not supported"},
		{name: "stdin data", command: `grpcurl -d @ localhost:1 a.B/C`, wantErr: "stdin"},
		{name: "missing method", command: `grpcurl -d '{}' localhost:1`, wantErr: "got 1 arguments"},
		{name: "invalid header", command: `grpcurl -H 'novalue' localhost:1 a.B/C`, wantErr: `invalid header "novalue"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGrpcurlCommand(tt.command)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got error %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

// What "Edit as GRPCURL" exports must import back unchanged.
func TestGrpcurlExportImportRoundTrip(t *testing.T) {
	a, _ := newTestAPI(t)
	a.setWorkspaceOptions(options{ID: workspacePrefix + "default", Addr: "localhost:50051", Plaintext: true})
	a.changeWorkspace(workspacePrefix + "default")

	data := `{"name":"O'Brien","note":"it's \"quoted\" \\ and $HOME"}`
	md := headers{{Key: "authorization", Val: "Bearer abc"}, {Key: "x-when", Val: "12:30"}}
	cmd := a.ExportCommands("/greeter.Greeter/SayHello", data, md).Grpcurl
	t.Logf("exported:\n%s", cmd)

	got, err := parseGrpcurlCommand(cmd)
	if err != nil {
		t.Fatal(err)
	}
	want := &grpcurlArguments{Target: "localhost:50051", Method: "greeter.Greeter/SayHello", Data: data, Metadata: md}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}
