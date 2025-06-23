package internals

import (
	"errors"
	"net/http"
)

var ErrUnknownRequestType = errors.New("unknown request type")

func GetRequestType(request string) (string, error) {
	if request[:4] == "GET " {
		return http.MethodGet, nil
	} else if request[:5] == "HEAD " {
		return http.MethodHead, nil
	} else if request[:4] == "PUT " {
		return http.MethodPut, nil
	} else if request[:6] == "PATCH " {
		return http.MethodPatch, nil
	} else if request[:7] == "DELETE " {
		return http.MethodDelete, nil
	}

	return "", ErrUnknownRequestType
}
