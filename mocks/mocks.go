package mocks

import (
	"errors"
	"net/http"
	"no-server/store"

	"github.com/stretchr/testify/mock"
)

type MockSendSteps struct {
	w       http.ResponseWriter
	file    store.File
	version int
	mock.Mock
}

func (mss *MockSendSteps) Do(w http.ResponseWriter, file store.File, version int) {
	mss.w = w
	mss.file = file
	mss.version = version
	mss.Called(w, file, version)
}

type MockStore struct {
	File *MockFile
	mock.Mock
}

func (s *MockStore) NewFile() store.File {
	s.Called()
	return s.File
}

func (s *MockStore) GetFile(name string) (f store.File, err error) {
	s.Called(name)
	return s.File, nil
}

type MockFile struct {
	MockName string
	mock.Mock
}

func (f *MockFile) Name() string {
	return f.MockName
}

func (f *MockFile) StepsSince(version int) ([]interface{}, error) {
	if version >= 0 {
		return []interface{}{version}, nil
	}
	return nil, errors.New("an Error")
}

func (f *MockFile) AddSteps(newSteps []interface{}, clientID int) {
	f.Called(newSteps, clientID)
}

func (f *MockFile) Version() int { return 10 }

type MockNameGenerator struct {
	Name string
}

func (g MockNameGenerator) New() string {
	return g.Name
}
