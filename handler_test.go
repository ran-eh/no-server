package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"regexp"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"net/http"
	"net/http/httptest"
	"testing"
)

type mockFile struct{}

func (f mockFile) Name() string {
	return "aName"
}

func (f mockFile) StepsSince(version int) ([]interface{}, error) {
	if version >= 0 {
		return []interface{}{version}, nil
	}
	return nil, errors.New("an Error")
}

func (f mockFile) AddSteps(newSteps []interface{}, clientID int) {}

func (f mockFile) Version() int { return 0 }

func TestSendSteps(t *testing.T) {
	t.Run("sendSteps returns the steps and file name for the file when version is valid", func(t *testing.T) {
		w := httptest.NewRecorder()
		f := mockFile{}
		version := 19
		prodSendSteps(w, f, version)
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var raw interface{}
		err := json.NewDecoder(resp.Body).Decode(&raw)
		require.NoError(t, err, "Decoding returned message failed")
		asMap, ok := raw.(map[string]interface{})
		require.True(t, ok, "Casting result to expected type failed")
		assert.EqualValues(t, map[string]interface{}{
			"fileName": "aName",
			"steps":    []interface{}{float64(19)},
		}, asMap)
	})
	t.Run("sendSteps fails when version is invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		f := mockFile{}
		version := -1
		sendSteps(w, f, version)
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestNewHandler(t *testing.T) {
	t.Run("NewHandler create new file and returns its handle", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://example.com/new", nil)
		w := httptest.NewRecorder()
		newHandler(w, req)
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := ioutil.ReadAll(resp.Body)
		assert.Regexp(t, `{"fileName":".*","steps":\[\]}`, string(body))

		re := regexp.MustCompile(`^{"fileName":"(.*)","steps":\[\]}\n$`)
		fileName := re.ReplaceAllString(string(body), "${1}")
		f, err := files.GetFile(fileName)
		assert.NoError(t, err)
		steps, _ := f.StepsSince(0)
		assert.Equal(t, 0, len(steps))
	})
}


func TestGetHandler(t *testing.T) {
	t.Run("Get create new file and returns its handle", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://example.com/new", nil)
		w := httptest.NewRecorder()
		newHandler(w, req)
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := ioutil.ReadAll(resp.Body)
		// assert.Equal(t, "", string(body))
		assert.Regexp(t, `{"fileName":".*","steps":\[\]}`, string(body))

		re := regexp.MustCompile(`^{"fileName":"(.*)","steps":\[\]}\n$`)
		fileName := re.ReplaceAllString(string(body), "${1}")
		f, err := files.GetFile(fileName)
		assert.NoError(t, err)
		steps, _ := f.StepsSince(0)
		assert.Equal(t, 0, len(steps))
	})
}
