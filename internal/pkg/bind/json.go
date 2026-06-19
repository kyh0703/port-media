package bind

import (
	"encoding/json"
	"net/http"
)

func JSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
