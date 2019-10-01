package ui

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/komuw/leaf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebUI(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "leaf.db")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	db, err := leaf.OpenBoltStore(tmpfile.Name())
	require.NoError(t, err)

	dm, err := leaf.NewDeckManager("../fixtures", db, leaf.OutputFormatOrg)
	require.NoError(t, err)

	srv := NewServer(dm)

	t.Run("listDecks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/decks", nil)
		w := httptest.NewRecorder()

		srv.listDecks(w, req)
		res := w.Result()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		decks := make([]*leaf.DeckStats, 0)
		require.NoError(t, json.NewDecoder(w.Body).Decode(&decks))
		assert.Len(t, decks, 2)
	})

	t.Run("deckStats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/stats/Org-mode", nil)
		w := httptest.NewRecorder()

		srv.deckStats(w, req)
		res := w.Result()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		stats := make([]map[string]interface{}, 0)
		require.NoError(t, json.NewDecoder(w.Body).Decode(&stats))
		require.Len(t, stats, 10)
		assert.Equal(t, "/emphasis/", stats[0]["card"])
	})

	t.Run("startReview", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://example.com/start/Hiragana", nil)
		w := httptest.NewRecorder()

		srv.startSession(w, req)
		res := w.Result()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		state := new(SessionState)
		require.NoError(t, json.NewDecoder(w.Body).Decode(state))
		assert.Equal(t, 20, state.Total)
		assert.Equal(t, 20, state.Left)
	})

	t.Run("advanceSession", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://example.com/advance", strings.NewReader("{\"score\":0}"))
		w := httptest.NewRecorder()

		srv.advanceSession(w, req)
		res := w.Result()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		state := new(SessionState)
		require.NoError(t, json.NewDecoder(w.Body).Decode(state))
		assert.Equal(t, 20, state.Total)
		assert.Equal(t, 20, state.Left)
	})

	t.Run("resolveAnswer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/resolve", nil)
		w := httptest.NewRecorder()

		srv.resolveAnswer(w, req)
		res := w.Result()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		result := make(map[string]string)
		require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
		assert.Equal(t, "i", result["answer"])
	})
}
