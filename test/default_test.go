package test

import (
	"testing"
	"fmt"
	"net/http"
)

func Test_BitOperation(t *testing.T) {
	st := make([]byte, 5)
	ot := st
	for i := 0; i < 5; i++ {
		ot[i] = '5'
	}
	fmt.Println(string(st), string(ot))
	http.Serve()
}