package converter

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"lcp.io/lcp/lib/ansible"
)

// ParsePlaybook parses YAML bytes into a Playbook struct.
func ParsePlaybook(data []byte) (*ansible.Playbook, error) {
	var plays []ansible.Play
	if err := yaml.Unmarshal(data, &plays); err != nil {
		return nil, fmt.Errorf("parse playbook: %w", err)
	}
	pb := &ansible.Playbook{Play: plays}
	if err := pb.Validate(); err != nil {
		return nil, err
	}
	return pb, nil
}

// BlockToTaskSpec converts a Block to a TaskSpec, identifying the module from UnknownField.
// The moduleFinder function checks if a name is a registered module.
func BlockToTaskSpec(block ansible.Block, hosts []string, role string, moduleFinder func(string) bool) ansible.TaskSpec {
	spec := ansible.TaskSpec{
		Name:         block.Name,
		Hosts:        hosts,
		When:         block.When.Data,
		FailedWhen:   block.FailedWhen.Data,
		Loop:         block.Loop,
		LoopControl:  block.Task.LoopControl,
		Register:     block.Register,
		Retries:      block.Retries,
		Delay:        block.Delay,
		Until:        block.Until.Data,
		Become:       block.Become,
		BecomeUser:   block.BecomeUser,
		DelegateTo:   block.DelegateTo,
		Async:        block.AsyncVal,
		Poll:         block.Poll,
		Notify:       block.Notify,
		IgnoreErrors: block.IgnoreErrors,
	}

	// Identify module from UnknownField
	for name, args := range block.UnknownField {
		if moduleFinder(name) {
			argsMap := make(map[string]any)
			switch v := args.(type) {
			case map[string]any:
				argsMap = v
			case string:
				argsMap[name] = v
			default:
				argsMap[name] = v
			}
			spec.Module = ansible.ModuleRef{Name: name, Args: argsMap}
			break
		}
	}

	return spec
}

// GroupHostBySerial splits hosts into batches based on serial specification.
// Serial items can be integers or percentage strings (e.g., "50%").
// The last serial value repeats for any remaining hosts.
func GroupHostBySerial(hosts []string, serial []any) ([][]string, error) {
	if len(serial) == 0 || len(hosts) == 0 {
		return [][]string{hosts}, nil
	}

	// Convert serial values to integer batch sizes.
	sis := make([]int, len(serial))
	var count int
	for i, a := range serial {
		switch val := a.(type) {
		case int:
			sis[i] = val
		case string:
			if strings.HasSuffix(val, "%") {
				b, err := strconv.ParseFloat(val[:len(val)-1], 64)
				if err != nil {
					return nil, fmt.Errorf("convert serial %q to float: %w", val, err)
				}
				sis[i] = int(math.Ceil(float64(len(hosts)) * b / 100.0))
			} else {
				b, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("convert serial %q to int: %w", val, err)
				}
				sis[i] = b
			}
		default:
			return nil, fmt.Errorf("unknown serial type: only int or percentage string supported")
		}
		if sis[i] == 0 {
			return nil, fmt.Errorf("serial %v should not be zero", a)
		}
		count += sis[i]
	}

	// Repeat the last serial value to cover remaining hosts.
	if len(hosts) > count {
		lastVal := sis[len(sis)-1]
		for i := 0.0; i < float64(len(hosts)-count)/float64(lastVal); i++ {
			sis = append(sis, lastVal)
		}
	}

	// Slice hosts into batches.
	result := make([][]string, len(sis))
	var begin, end int
	for i, si := range sis {
		end += si
		if end > len(hosts) {
			end = len(hosts)
		}
		result[i] = hosts[begin:end]
		begin += si
	}

	return result, nil
}

// ConvertVarsNodes converts Vars yaml.Nodes to a merged map[string]any.
// Later nodes override earlier ones for duplicate keys.
func ConvertVarsNodes(nodes []yaml.Node) (map[string]any, error) {
	result := make(map[string]any)
	for _, node := range nodes {
		var m map[string]any
		if err := node.Decode(&m); err != nil {
			return nil, fmt.Errorf("decode vars: %w", err)
		}
		for k, v := range m {
			result[k] = v
		}
	}
	return result, nil
}
