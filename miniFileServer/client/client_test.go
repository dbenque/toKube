package client

import (
	"fmt"
	"testing"
)

func TestClient(t *testing.T) {
	err := PostFile("client_test.go", "http://192.168.99.100:30666")
	fmt.Printf("%#v\n", err)
	err = PostFile("/tmp/test892221340/test", "http://192.168.99.100:30666")
	fmt.Printf("%#v\n", err)
}
