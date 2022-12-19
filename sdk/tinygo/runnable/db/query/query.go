//go:build tinygo.wasm

package query

type Argument struct {
	Name, Value string
}

func NewArgument(name, value string) Argument {
	return Argument{
		Name:  name,
		Value: value,
	}
}

type QueryType int32

const (
	QueryInsert QueryType = iota
	QuerySelect
	QueryUpdate
	QueryDelete
)
