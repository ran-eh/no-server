package inmemorystore

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"

	"no-server/store"

	"github.com/akamensky/base58"
)

type NameGenerator interface {
	New() string
}

type prodNameGenerator struct{}

func (png prodNameGenerator) New() string {
	data := make([]byte, 4)
	_, err := rand.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	return base58.Encode(data)
}

type Store struct {
	files         map[string]store.File
	nameGenerator NameGenerator
}

func New(nameGenerator NameGenerator) *Store {
	if nameGenerator == nil {
		nameGenerator = &prodNameGenerator{}
	}
	return &Store{map[string]store.File{}, nameGenerator}
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
