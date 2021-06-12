package directive

import "github.com/suborbital/reactr/rwasm/moduleref"

// Runnable is the structure of a .runnable.yaml file
type Runnable struct {
	Name       string                   `yaml:"name" json:"name"`
	Namespace  string                   `yaml:"namespace" json:"namespace"`
	Lang       string                   `yaml:"lang" json:"lang"`
	APIVersion string                   `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	FQFN       string                   `yaml:"fqfn,omitempty" json:"fqfn,omitempty"`
	ModuleRef  *moduleref.WasmModuleRef `yaml:"-" json:"moduleRef"`
}
