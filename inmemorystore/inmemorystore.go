// This package implements in-memory storage for edit histories
package inmemorystore

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"

	"no-server/store"

	"github.com/akamensky/base58"
)

// File "names" are generated using this interface
type NameGenerator interface {
	New() string
}

type randomNameGenerate struct{}

// Generate ramdom file "name"
func (ng randomNameGenerate) New() string {
	data := make([]byte, 4)
	_, err := rand.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	return base58.Encode(data)
}

// An in-memory implementation of store.Store
type Store struct {
	files         map[string]store.File
	// Injected name generator.  Allows mocking for test.
	nameGenerator NameGenerator
}

func New(nameGenerator NameGenerator) *Store {
	// By default, use random name generator
	if nameGenerator == nil {
		nameGenerator = &randomNameGenerate{}
	}
	return &Store{files: map[string]store.File{}, nameGenerator: nameGenerator}
}

func (s *Store) NewFile() store.File {
	name := s.nameGenerator.New()
	f := &File{Steps: []interface{}{}, FileName: name}
	s.files[name] = f
	return f
}

func (s Store) GetFile(name string) (f store.File, err error) {
	f, ok := s.files[name]
	if !ok {
		err = errors.New("file not found")
	}
	return
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
		return nil, fmt.Errorf("invalid version: %d", version)
	}
	if version > len(f.Steps) {
		return nil, fmt.Errorf("version %d ahead of file version %d", version, len(f.Steps))
	}
	return f.Steps[version:], nil
}
