package handlers

import "net/http"

func Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, `{"ok":true,"service":"verified-bases-api","phase":"1-manual-curated-store"}`)
}
