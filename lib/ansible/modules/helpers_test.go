package modules

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"lcp.io/lcp/lib/ansible"
	"lcp.io/lcp/lib/ansible/variable"
)

// mockConnector implements connector.Connector for testing across all module tests.
type mockConnector struct {
	// ExecuteCommand
	execFn     func(ctx context.Context, cmd string) ([]byte, []byte, error)
	execOutput []byte
	execErr    error

	// PutFile
	putFileData  []byte
	putFileDest  string
	putFileMode  fs.FileMode
	putFileCalls []putFileCall
	putFileErr   error

	// FetchFile
	fetchFileContent []byte
	fetchFileErr     error
}

type putFileCall struct {
	src  []byte
	dst  string
	mode fs.FileMode
}

func (m *mockConnector) Init(context.Context) error  { return nil }
func (m *mockConnector) Close(context.Context) error { return nil }
func (m *mockConnector) ExecuteCommand(ctx context.Context, cmd string) ([]byte, []byte, error) {
	if m.execFn != nil {
		return m.execFn(ctx, cmd)
	}
	return m.execOutput, nil, m.execErr
}
func (m *mockConnector) PutFile(_ context.Context, src []byte, dst string, mode fs.FileMode) error {
	m.putFileData = src
	m.putFileDest = dst
	m.putFileMode = mode
	m.putFileCalls = append(m.putFileCalls, putFileCall{src: src, dst: dst, mode: mode})
	return m.putFileErr
}
func (m *mockConnector) FetchFile(_ context.Context, _ string, dst io.Writer) error {
	if m.fetchFileErr != nil {
		return m.fetchFileErr
	}
	if m.fetchFileContent != nil {
		_, err := dst.Write(m.fetchFileContent)
		return err
	}
	return nil
}

// mockGatherFactsConnector implements both connector.Connector and connector.GatherFacts.
type mockGatherFactsConnector struct {
	mockConnector
	hostInfoFn func(ctx context.Context) (map[string]any, error)
}

func (m *mockGatherFactsConnector) HostInfo(ctx context.Context) (map[string]any, error) {
	if m.hostInfoFn != nil {
		return m.hostInfoFn(ctx)
	}
	return nil, nil
}

// mockSource provides in-memory file content for Source.ReadFile.
type mockSource struct {
	files map[string][]byte
}

func (s *mockSource) ReadFile(path string) ([]byte, error) {
	data, ok := s.files[path]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return data, nil
}

// newTestVariable creates a Variable with the given hosts pre-populated.
func newTestVariable(hosts ...string) variable.Variable {
	hostMap := make(map[string]map[string]any, len(hosts))
	for _, h := range hosts {
		hostMap[h] = make(map[string]any)
	}
	return variable.New(ansible.Inventory{
		Hosts: hostMap,
	})
}

// newTestVariableWithVars creates a Variable for a single host with pre-populated host vars.
func newTestVariableWithVars(host string, hostVars map[string]any) variable.Variable {
	return variable.New(ansible.Inventory{
		Hosts: map[string]map[string]any{
			host: hostVars,
		},
	})
}

// getHostRuntimeVars reads a host's RuntimeVars from the Variable.
func getHostRuntimeVars(v variable.Variable, host string) map[string]any {
	result := v.Get(func(val *variable.Value) any {
		hv, ok := val.Hosts[host]
		if !ok {
			return map[string]any{}
		}
		return hv.RuntimeVars
	})
	if m, ok := result.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// getHostRemoteVars reads a host's RemoteVars from the Variable.
func getHostRemoteVars(v variable.Variable, host string) map[string]any {
	result := v.Get(func(val *variable.Value) any {
		hv, ok := val.Hosts[host]
		if !ok {
			return map[string]any{}
		}
		return hv.RemoteVars
	})
	if m, ok := result.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
