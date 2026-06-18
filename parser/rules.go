package parser

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type RuleConf struct {
	On RuleOn `yaml:"on,omitempty"`
}

func (r *RuleConf) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("rule must be a mapping, got %s", value.ShortTag())
	}
	foundOn := false
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i]
		if key.Value != "on" {
			continue
		}
		foundOn = true
		if err := value.Content[i+1].Decode(&r.On); err != nil {
			return err
		}
	}
	if !foundOn {
		r.On.Default = true
	}
	return nil
}

type RuleOn struct {
	Value   string
	Bool    *bool
	Default bool
}

func (o *RuleOn) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Tag == "!!bool" {
			var b bool
			if err := value.Decode(&b); err != nil {
				return err
			}
			o.Bool = &b
			return nil
		}
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		o.Value = s
		return nil
	default:
		return fmt.Errorf("rule on must be a bool or string, got %s", value.ShortTag())
	}
}
