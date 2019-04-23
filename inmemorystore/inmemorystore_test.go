package inmemorystore_test

import (
	"no-server/inmemorystore"
	"no-server/mocks"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	t.Run("Store.NewFile places a new file in the store", func(t *testing.T) {
		store := inmemorystore.New(mocks.MockNameGenerator{Name: "mockName"})
		f := store.NewFile().(*inmemorystore.File)
		assert.Equal(t, "mockName", f.Name())
		assert.Empty(t, f.Steps)
		storedFile, _ := store.GetFile("mockName")
		assert.Equal(t, f, storedFile)
	})
}

func TestAddSteps(t *testing.T) {
	t.Run("Add no steps to no steps keeps empty steps", func(t *testing.T) {
		f := inmemorystore.File{Steps: []interface{}{}}
		f.AddSteps([]interface{}{}, 1)
		assert.Empty(t, f.Steps)
	})
	t.Run("Add steps to no steps adds them properly", func(t *testing.T) {
		f := inmemorystore.File{Steps: []interface{}{}}
		f.AddSteps([]interface{}{"a", "b", "c"}, 1)
		assert.Equal(t,
			[]interface{}{
				map[string]interface{}{"clientID": 1, "step": "a"},
				map[string]interface{}{"clientID": 1, "step": "b"},
				map[string]interface{}{"clientID": 1, "step": "c"},
			},
			f.Steps)
	})
}

func TestStepsSince(t *testing.T) {
	f := inmemorystore.File{FileName: "aName", Steps: []interface{}{1, 2, 3, 4, 5}}
	t.Run("StepsSince returns all steps for requested version 0", func(t *testing.T) {
		steps, err := f.StepsSince(0)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{1, 2, 3, 4, 5}, steps)
	})
	t.Run("StepsSince return partial steps for requested version > 0 and < file version", func(t *testing.T) {
		steps, err := f.StepsSince(2)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{3, 4, 5}, steps)
	})
	t.Run("StepsSince return no steps for requested version == file version", func(t *testing.T) {
		steps, err := f.StepsSince(5)
		require.NoError(t, err)
		assert.Empty(t, steps)
	})
	t.Run("StepsSince fails if requested version is ahead of file version", func(t *testing.T) {
		_, err := f.StepsSince(6)
		assert.Error(t, err)
	})
}
