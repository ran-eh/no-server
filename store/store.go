// This package define the interfaces for edit history storage
package store

type Store interface {
	NewFile() File
	GetFile(name string) (f File, err error)
}

type File interface {
	Name() string
	AddSteps(newSteps []interface{}, clientID int)
	Version() int
	StepsSince(version int) ([]interface{}, error)
}
