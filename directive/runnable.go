package directive

import "github.com/suborbital/reactr/rwasm/moduleref"

// Runnable is the structure of a .runnable.yaml file
type Runnable struct {
	Name         string                   `yaml:"name" json:"name"`
	Namespace    string                   `yaml:"namespace" json:"namespace"`
	Lang         string                   `yaml:"lang" json:"lang"`
	Version      string                   `yaml:"version" json:"version"`
	DraftVersion string                   `yaml:"draftVersion,omitempty" json:"draftVersion,omitempty"`
	APIVersion   string                   `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	FQFN         string                   `yaml:"fqfn,omitempty" json:"fqfn,omitempty"`
	FQFNURI      string                   `yaml:"fqfnUri" json:"fqfnURI,omitempty"`
	ModuleRef    *moduleref.WasmModuleRef `yaml:"-" json:"moduleRef,omitempty"`
}
