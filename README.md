# goJsonReader
A package to extract json values from a json string.

This code has been sitting in my computer for a long time so I decided to share it on github.

My main objective was to make a json reader faster than the other public golang json readers in order to use it in on my own projects and to exercise golang.

# basic usage
```
package main

import (
	"fmt"
	"githu.com/cassiohg/goJsonReader"
)

func main() {
	json := []byte(`{"a": {"b": 10, "c": 20, "d": 30}, b: "lala"}`)

	str, d, err := goJsonReader.Get(json, []string{"a", "b"})
	if err != nil {
		panic(err)
	}
	fmt.Printf("value=%s, dataType=%v\n", str, d)

	err := goJsonReader.ForEach(json, []string{"a"}, func (index int, key, value string, d DataType) bool {
		fmt.Printf("index=%d, key=%s, value=%s, dataType=%v\n", index, key, value, d)
		return true;
	})
	if err != nil {
		panic(err)
	}
}
```