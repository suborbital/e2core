package directive

// Runnable is the structure of a .runnable.yaml file
type Runnable struct {
	Name       string `yaml:"name"`
	Namespace  string `yaml:"namespace"`
	Lang       string `yaml:"lang"`
	APIVersion string `yaml:"apiVersion,omitempty"`
}
