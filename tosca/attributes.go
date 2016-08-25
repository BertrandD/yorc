package tosca

type AttributeDefinition struct {
	Type        ValueAssignment `yaml:"type"`
	Description string          `yaml:"description,omitempty"`
	Default     string          `yaml:"default,omitempty"`
	Status      string          `yaml:"status,omitempty"`
	//EntrySchema string `yaml:"entry_schema,omitempty"`
}

func (r *AttributeDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ras ValueAssignment
	if err := unmarshal(&ras); err == nil {
		r.Type = ras
		return nil
	}

	var ra struct {
		Type        ValueAssignment `yaml:"type"`
		Description string          `yaml:"description,omitempty"`
		Default     string          `yaml:"default,omitempty"`
		Status      string          `yaml:"status,omitempty"`
	}

	if err := unmarshal(&ra); err == nil {
		r.Description = ra.Description
		r.Type = ra.Type
		r.Default = ra.Default
		r.Status = ra.Status
		return nil
	}

	return nil
}
