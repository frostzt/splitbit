package internals

import "testing"

func TestGetRequestType(t *testing.T) {
	request := "GET / HTTP/1.1"

	requestType, err := GetRequestType(request)
	if err != nil {
		t.Errorf("GetRequestType(%s) returned an error: %s", request, err.Error())
	}

	if requestType != "GET" {
		t.Errorf("GetRequestType returned wrong value for requestType %s", requestType)
	}
}
