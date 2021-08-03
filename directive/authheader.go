package directive

type DomainAuth struct {
	HeaderType string `yaml:"headerType" json:"headerType"`
	Value      string `yaml:"value" json:"value"`
}
