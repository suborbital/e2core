package plugin

// The Plugin interface is all that needs to be implemented by an E2 Core plugin.
type Plugin interface {
	Run(input []byte) ([]byte, error)
}
