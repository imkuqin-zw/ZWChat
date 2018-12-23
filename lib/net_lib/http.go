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

func GetMethod(r *Reader) ([]byte, error) {
	i := 4
	for {
		data, err := r.Peek(i)
		if err != nil {
			return nil, err
		}
		last := i-1
		if data[last] == ' ' {
			return data[:last], nil
		}
		i++
		if i > 8 {
			return nil, nil
		}
	}
}

func CheckMethod(method string) bool {
	_, ok := methodMap[method]
	return ok
}

func IsHttp(r *Reader) (bool, error) {
	data, err := GetMethod(r)
	if err != nil {
		return false, err
	}
	if data == nil {
		return false, nil
	}
	method := strings.ToUpper(string(data))
	return CheckMethod(method), nil
}
