package main

import (
	"bytes"
	"encoding/json"
	"no-server/mocks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendSteps(t *testing.T) {
	t.Run("sendSteps returns the steps and file name for the file when version is valid", func(t *testing.T) {
		w := httptest.NewRecorder()
		f := &mocks.MockFile{MockName: "aNewFile"}
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
			"fileName": "aNewFile",
			"steps":    []interface{}{float64(19)},
		}, asMap)
	})
	t.Run("sendSteps fails when version is invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		f := &mocks.MockFile{}
		version := -1
		sendSteps(w, f, version)
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestNewHandler(t *testing.T) {
	mss := &mocks.MockSendSteps{}
	sendSteps = mss.Do
	newMockFile := &mocks.MockFile{MockName: "aNewFile"}
	ms := &mocks.MockStore{
		File: newMockFile,
	}
	oldFiles = ms

	t.Run("NewHandler create new file and returns its handle as version 0", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://example.com/new", nil)
		w := httptest.NewRecorder()
		ms.On("NewFile").Return(newMockFile)
		mss.On("Do", w, newMockFile, 0)
		halndleNew(w, req)
		mss.AssertExpectations(t)
		ms.AssertExpectations(t)
	})
}

func TestGetHandler(t *testing.T) {
	t.Run("Get fails for an invalid file name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/get?name=&version=0", nil)
		w := httptest.NewRecorder()
		handleGet(w, req)
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	t.Run("GetHandler obtains steps for the file and version requested ", func(t *testing.T) {
		mss := &mocks.MockSendSteps{}
		sendSteps = mss.Do
		existingMockFile := &mocks.MockFile{MockName: "anExistingFile"}
		ms := &mocks.MockStore{
			File: existingMockFile,
		}
		oldFiles = ms

		req := httptest.NewRequest("GET", "http://example.com/get?name=aFile&version=18", nil)
		w := httptest.NewRecorder()
		ms.On("GetFile", mock.Anything).Return(existingMockFile)
		mss.On("Do", w, &mocks.MockFile{MockName: "anExistingFile"}, 18)
		handleGet(w, req)
		mss.AssertExpectations(t)
		ms.AssertExpectations(t)
	})
}

func TestUpdateHandler(t *testing.T) {
	t.Run("UpdateHandler updates returns provided steps when client/server versions match", func(t *testing.T) {
		mss := &mocks.MockSendSteps{}
		sendSteps = mss.Do
		existingMockFile := &mocks.MockFile{MockName: "anExistingFile"}
		ms := &mocks.MockStore{
			File: existingMockFile,
		}
		oldFiles = ms
		var b bytes.Buffer
		ui := &updateInfo{
			ClientID:      1,
			ClientSteps:   []interface{}{"clientSteps"},
			ClientVersion: 10,
			FileName:      "anExistingFile",
		}
		_ = json.NewEncoder(&b).Encode(ui)
		req := httptest.NewRequest("PUT", "http://example.com/get?name=aFile&version=18", &b)
		w := httptest.NewRecorder()
		ms.On("GetFile", mock.AnythingOfType("string")).Return(&mocks.MockFile{MockName: "anExistingFile"})
		ms.File.On("AddSteps", []interface{}{"clientSteps"}, 1)
		mss.On("Do", w, existingMockFile, 10)
		handleUpdate(w, req)
		mss.AssertExpectations(t)
		ms.AssertExpectations(t)
		ms.File.AssertExpectations(t)
	})
	t.Run("UpdateHandler updates returned outstanding steps when client/server versions do not match", func(t *testing.T) {
		mss := &mocks.MockSendSteps{}
		sendSteps = mss.Do
		existingMockFile := &mocks.MockFile{MockName: "anExistingFile"}
		ms := &mocks.MockStore{
			File: existingMockFile,
		}
		oldFiles = ms
		var b bytes.Buffer
		ui := &updateInfo{
			ClientID:      1,
			ClientSteps:   []interface{}{"clientSteps"},
			ClientVersion: 4,
			FileName:      "anExistingFile",
		}
		_ = json.NewEncoder(&b).Encode(ui)
		req := httptest.NewRequest("PUT", "http://example.com/get?name=aFile&version=18", &b)
		w := httptest.NewRecorder()
		ms.On("GetFile", mock.AnythingOfType("string")).Return(&mocks.MockFile{MockName: "anExistingFile"})
		// No call to AddSteps!
		mss.On("Do", w, existingMockFile, 4)
		handleUpdate(w, req)
		mss.AssertExpectations(t)
		ms.AssertExpectations(t)
		ms.File.AssertExpectations(t)
	})
}
