package util

type Command interface {
	Init([]string) error
	Execute() error
	Name() string
}
