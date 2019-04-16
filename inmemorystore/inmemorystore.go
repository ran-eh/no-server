package inmemorystore

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"

	"no-server/store"

	"github.com/akamensky/base58"
)

type Store struct {
	files map[string]store.File
}

func New() *Store {
	return &Store{map[string]store.File{}}
}

func (s *Store) NewFile() store.File {
	name := newID()
	f := &File{Steps: []interface{}{}, FileName: name}
	s.files[name] = f
	return f
}

func (s Store) GetFile(name string) (f store.File, err error) {
	f, ok := s.files[name]
	if !ok {
		err = errors.New("File not found")
	}
	return
}

// Generate a new file id.
func newID() string {
	data := make([]byte, 4)
	_, err := rand.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	return base58.Encode(data)
}

type File struct {
	FileName string
	Steps    []interface{}
}

func (f File) Name() string {
	return f.FileName
}

func (f *File) AddSteps(newSteps []interface{}, clientID int) {
	for _, newStep := range newSteps {
		f.Steps = append(f.Steps, map[string]interface{}{
			"clientID": clientID,
			"step":     newStep})
	}
}

func (f File) Version() int {
	return len(f.Steps)
}

func (f File) StepsSince(version int) ([]interface{}, error) {
	if version < 0 {
		return nil, fmt.Errorf("Invalid version: %d", version)
	}
	if version > len(f.Steps) {
		return nil, fmt.Errorf("Version %d ahead of file version %d", version, len(f.Steps))
	}
	return f.Steps[version:], nil
}
