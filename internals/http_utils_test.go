package internals

import "testing"

func TestGetRequestType(t *testing.T) {
	request := "GET / HTTP/1.1"

	isHttpRequest := IsHTTPRequest(request)
	if isHttpRequest != true {
		t.Errorf("Expected 'true' but got '%t'", isHttpRequest)
	}
}
