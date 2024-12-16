package baserespone

import (
	"encoding/json"
	"net/http"
)

func DatadogBaseRespone(resp interface{}, r *http.Response, err error) ([]byte, *http.Response, error) {
	if err != nil {
		return nil, r, err
	}

	responseContent, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, r, err
	}

	return responseContent, r, err
}
