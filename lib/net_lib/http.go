package net_lib

import (
	"strings"
)

var methodMap = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"POST":    true,
	"PUT":     true,
	"PATCH":   true,
	"DELETE":  true,
	"CONNECT": true,
	"OPTIONS": true,
	"TRACE":   true,
}


func IsHttp(r *Reader) bool {
	data, _ := r.Peek(8)
	method := strings.Split(string(data), " ")[0]
	method = strings.ToUpper(method)
	_, ok := methodMap[method]
	return ok
}
