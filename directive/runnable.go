package directive

// Runnable is the structure of a .runnable.yaml file.
type Runnable struct {
	Name         string         `yaml:"name" json:"name"`
	Namespace    string         `yaml:"namespace" json:"namespace"`
	Lang         string         `yaml:"lang" json:"lang"`
	Version      string         `yaml:"version" json:"version"`
	DraftVersion string         `yaml:"draftVersion,omitempty" json:"draftVersion,omitempty"`
	APIVersion   string         `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	FQFN         string         `yaml:"fqfn,omitempty" json:"fqfn,omitempty"`
	FQFNURI      string         `yaml:"fqfnUri" json:"fqfnURI,omitempty"`
	ModuleRef    *WasmModuleRef `yaml:"-" json:"moduleRef,omitempty"`
	TokenHash    []byte         `yaml:"-" json:"-"`
}

// WasmModuleRef is a reference to a Wasm module
// This is a duplicate of sat/engine/moduleref/WasmModuleRef (for JSON serialization purposes)
type WasmModuleRef struct {
	Name string `json:"name"`
	FQFN string `json:"fqfn"`
	Data []byte `json:"data"`
}

func NewWasmModuleRef(name, fqfn string, data []byte) *WasmModuleRef {
	w := &WasmModuleRef{
		Name: name,
		FQFN: fqfn,
		Data: data,
	}

	return w
}
