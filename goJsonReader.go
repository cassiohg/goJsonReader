package goJsonReader

import (
	"fmt"
	"strconv"
	"unsafe"
	"reflect"
	"bytes"
)


type DataType int8
const (
	JsonObject = DataType(iota)
	JsonArray 
	JsonString 
	JsonNumber 
	JsonBoolean 
	JsonNull 
)
func (d DataType) String() string {
	switch d {
	case JsonObject: return "Object"
	case JsonArray: return "Array"
	case JsonString: return "String"
	case JsonNumber: return "Number"
	case JsonBoolean: return "Boolean"
	case JsonNull: return "Null"
	default:
		return fmt.Sprintf("Type[%v]",int8(d))
	}
}

// func btos(b *[]byte) string { return *(*string)(unsafe.Pointer(b)) }
func btos(b []byte) string { return *(*string)(unsafe.Pointer(&b)) }
func stob(s string) []byte { return (*[0x7fff0000]byte)(unsafe.Pointer((*reflect.StringHeader)(unsafe.Pointer(&s)).Data))[:len(s):len(s)] }

type JSONEndsAbruptlyError struct {}
func (e *JSONEndsAbruptlyError) Error () string { return "JSON ends abruptly." }
type JSONBadSyntaxError struct {c byte; i int}
func (e *JSONBadSyntaxError) Error () string { return fmt.Sprintf("Bad JSON syntax '%c' at %d.", e.c, e.i) }



/* Receives a string to be read as json as 1st argument followed by one or more strings that compose one path to a value 
inside the json structure. Returns the value as string, the value data type as int and may return a non nil error. There 
may be short cuts implemented which impose constraints on how the json can be formed. The trade off is parsing speed.*/
func ForEach (json []byte, keys []string, each func(string, string, DataType, error) bool) error {
	if len(keys) == 0 { return nil }
	length := len(json)

	keyIndex := 0 // index in keys slice.
	key := keys[keyIndex] // first key.

	// err := fmt.Errorf("JSON is not a object, cannot hold key \"%s\".", key)
	path := 1 // state of path search. 
	// 0 means last parsed key is not in the path we are searching.
	// 1 means we are in the given path.
	// 2 means we are at the last key given.
	value := true // searching for a value.
	stack := make([]byte, 0, len(keys)) // stack of structures when traversing in Breadth First Search.
	// byte for '{' means we entered an object.
	// byte for '[' means we entered an array.
	indexes := make([]int, 0, 2) // stack of indexes for each array we enter.
	var keyNumeric int // int to store current key value when it is an array index instead of a string.
	var bufferEach bool // when set to true, each key value pair will be saved before executing the given function.
	var lastKey string // last found key to be passed in argument to given function.

	i := 0 // index inside the string. pointing to runes.
	for i < len(json) {
		// fmt.Printf("[%d] c='%c'. value=%v, keyIndex=%d, len(stack)=%d, path=%d\n", i, json[i], value, keyIndex, len(stack), path) // debug.

		if value { // reading value
			// fmt.Printf("-- [keyIndex: %d] reading value.\n", keyIndex) // debug.

			/* Between 'space', 'tab', 'return' and 'new line' chars, the 'space' char has the biggest byte number and they 
			are all bellow any other important char. */
			for json[i] <= ' ' { i++; if i>=length{return &JSONEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.

			switch json[i] { // matching char with a type of value.
			case '{': // objects.
				// fmt.Printf("'{'\n") // debug
				if path == 1 {
					i++
					for json[i] <= ' ' { i++; if i>=length{return &JSONEndsAbruptlyError{}} } // skipping spaces, tab, returns and new lines.
					if json[i] != '}' {
						stack = append(stack, '{') // when an object, we have another structure to traverse.
						value = false
						continue
					}
				} else if path == 2 {
					i++
					for json[i] <= ' ' { i++ } // skipping spaces, tab, returns and new lines.
					// fmt.Printf("path: %d\n", path) // debug
					if bufferEach == false {
						bufferEach = true
						if json[i] != '}' {
							value = false
							continue
						}
					} else {
						if json[i] != '}' {
							valueStart := i
							nest := 0
							for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': for json[i] != '"' || json[i-1] == '\\' { i++ } // ignoring escaped quotes.
								case '{': nest++
								case '}': nest--
								}
							}
							// fmt.Printf(">>> executing FUNCTION\n") // debug
							if !each(lastKey, btos(json[valueStart:i]), JsonObject, nil) { return nil }
							value = false
							continue
						} else {
							return nil
						}
					}
				} else if path == 0 {
					// valueStart := i // debug.
					nest := 0
					for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': for json[i] != '"' || json[i-1] == '\\' { i++ } // ignoring escaped quotes.
						case '{': nest++
						case '}': nest--
						}
					}
					// fmt.Printf("skipped whole object: %v\n", json[valueStart:i]) // debug.
				}
			case '[': // arrays.
				// fmt.Printf("'['\n") // debug.
				if path == 1 {
					i++
					for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
					if json[i] != ']' {
						// inside an array there are more values, so we have to keep 'value = true'.
						stack = append(stack, '[')
						indexes = append(indexes, 0) // index 0 for first value of the array.
						var err error
						keyNumeric, err = strconv.Atoi(key)
						if err != nil {
							return fmt.Errorf("Could not convert '%s' to number in given keys %v", key, keys[:keyIndex+1])
						}
						if keyNumeric == 0 { path = 2 } else { path = 0 }
						continue
					}
				} else if path == 2 {
					for json[i] <= ' ' { i++ } // skipping spaces, tab, returns and new lines.
					bufferEach = true
					if json[i] != ']' {
						valueStart := i
						nest := 0
						for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
							switch json[i] {
							case '"': for json[i] != '"' || json[i-1] == '\\' { i++ } // ignoring escaped quotes.
							case '[': nest++
							case ']': nest--
							}
						}
						if !each(lastKey, btos(json[valueStart:i]), JsonArray, nil) { return nil }
					} else {
						return nil
					}
				} else {
					// valueStart := i // debug.
					// fmt.Printf("valueStart=%d.\n", valueStart) // debug.
					nest := 0
					for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': for json[i] != '"' || json[i-1] == '\\' { i++ } // ignoring escaped quotes.
						case '[': nest++
						case ']': nest--
						}
					}
					// fmt.Printf("skipped whole array: %v\n", json[valueStart:i]) // debug.
				}
			case '"': // strings.
				// fmt.Printf("start of a string\n") // debug.
				i++
				k := i
				for json[i] != '"' || json[i-1] == '\\' { i++ } // searching string end.
				// fmt.Printf("start=%d, end=%d.\n", k, i) // debug.
				if path == 1 {
					return fmt.Errorf("JSON %v is not object can't traverse further.", keys[:keyIndex+1])
				} else if path == 2{
					each(lastKey, btos(bytes.ReplaceAll(json[k:i], []byte{'\\','\\'}, []byte{'\\'})), JsonString, nil)
					return nil
				}
				i++
			case 't': // true
				// fmt.Printf("valueBool true.\n") // debug.
				// fmt.Printf("start=%d, end=%d.\n", i, i+4) // debug.
				if path == 1 {
					return fmt.Errorf("JSON %v is not object can't traverse further.", keys[:keyIndex+1])
				} else if path == 2 {
					each(lastKey, btos(json[i:i+4]), JsonBoolean, nil)
					return nil
				}
				i += 4
			case 'f': // false.
				// fmt.Printf("valueBool false.\n") // debug
				// fmt.Printf("start=%d, end=%d.\n", i, i+5) // debug
				if path == 1 {
					return fmt.Errorf("JSON %v is not object can't traverse further.", keys[:keyIndex+1])
				} else if path == 2 {
					each(lastKey, btos(json[i:i+5]), JsonBoolean, nil)
					return nil
				}
				i += 5
			case 'n': // null.
				// fmt.Printf("valueNull.\n") // debug
				// fmt.Printf("start=%d, end=%d.\n", i, i+4) // debug
				if path == 1 {
					return fmt.Errorf("JSON %v is not object can't traverse further.", keys[:keyIndex+1])
				} else if path == 2 {
					each(lastKey, btos(json[i:i+4]), JsonNull, nil)
					return nil
				}
				i += 4
			case '-': fallthrough // negative numbers
			default: // numbers.
				// fmt.Printf("valueNumber.\n") // debug
				k := i
				i++
				for c := json[i]; c >= '0' && c <= '9' || c == '.'; c = json[i] { i++ }
				// fmt.Printf("start=%d, end=%d.\n", k, i) // debug
				if path == 0 {
					break
				} else if path == 1 {
					return fmt.Errorf("JSON %v is not object can't traverse further.", keys[:keyIndex+1])
				} else if path == 2 {
					each(lastKey, btos(json[k:i]), JsonNumber, nil)
					return nil
				}
			}
			// fmt.Printf("value skipped\n") // debug.
			// fmt.Printf("-- finished reading value.\n") // debug
			for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.

			switch json[i] {
			// if we are in a object, we'll parse a key. if we are in an array, we'll parse a value.
			case ',':
				value = stack[len(stack)-1] == '['
				if value {
					indexes[len(indexes)-1]++ // last value has passed, we increment the array index.
					if keyNumeric == indexes[len(indexes)-1] { // if given key (converted to int) matched the array index.
						if len(keys) > keyIndex+1 { // if this is not the last key.
							keyIndex++
							key = keys[keyIndex]
							path = 1
							// fmt.Printf("next key is \"%s\".\n", key) // debug.
						} else { // if this is the last key.
							path = 2
							// fmt.Println("This key is the last one and belongs to an Array Index. Returning next value.") // debug.
						}
					}
				}
			case '}', ']':
				if json[i] == ']' { indexes = indexes[:len(indexes)-1] }
				stack = stack[:len(stack)-1]
				if len(stack) < keyIndex+1 {
					// fmt.Printf("stack length: %v, keyIndex: %v, key: %v\n", len(stack), keyIndex, key) // debug
					// return fmt.Errorf(`JSON path %v could not be found.`, keys[:keyIndex+1])
					return nil
				}
			}
			i++
			for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
		} else { // reading key
			// fmt.Printf("-- [keyIndex: %d] reading key.\n", keyIndex) // debug
			value = true
			for json[i] != '"' { i++ } // skipping every char that isn't a double quote.
			i++

			if bufferEach == false {
			// if stack level matches query string index and we are still traversing the structure.
			// fmt.Printf("stack length: %d, keyIndex: %d, path: %d.\n", len(stack), keyIndex, path) // debug.
			// if len(stack) == keyIndex+1 && path < 2 { // keyIndex+1 because stack is 1 at root level, and 0 when json is just a value.
				// fmt.Printf("trying to match with key '%s'.\n", key) // debug.
				// trying to match this key with current given key.
				j := 0
				keyMatch := true
				for j < len(key) {
					// fmt.Printf("checking key: a='%c' - b='%c'\n", json[i], key[j]) // debug.
					if json[i] != key[j] {
						keyMatch = false
						break
					}
					i++
					j++
				}
				if keyMatch && json[i] == '"' { // if keys have the same chars and json key doesn't have more chars.
					// fmt.Printf("key found %s correct found at position %d.\n", json[i-j-1:i+1], i-j-1) // debug.
					for json[i] != ':' { i++ } // searching key-value separator.
					if len(keys) > keyIndex+1 {
						keyIndex++
						key = keys[keyIndex]
						path = 1 // in the path to final key.
						// fmt.Printf("next key is \"%s\".\n", key) // debug.
					} else {
						path = 2 // final key. we should never see 'value = false' again.
						// fmt.Printf("This key is the last one. Returning next value.\n") // debug.
					}
				} else {
					path = 0 // deviated from correct path.
					// k := i // debug
					for json[i] != '"' { i++ } // searching key end.
					// fmt.Printf("key found %s not correct.\n", json[k-j-1:i+1]) // debug
					for json[i] != ':' { i++ } // searching key-value separator.
				}
			} else {
			// 	// k := i // debug
			// 	for json[i] != '"' { i++ } // searching key end.
			// 	// fmt.Printf("ignoring key '%s'.\n", json[k-1:i+1]) // debug
			// 	for json[i] != ':' { i++ } // searching key-value separator.
				k := i
				for json[i] != '"' || json[i-1] == '\\' { i++ }
				lastKey = btos(json[k:i])
				// fmt.Printf("lastKey '%s'.\n", lastKey) // debug
				i++
				for json[i] != ':' { i++ }
			}
			i++
			for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
		}
	}

	return fmt.Errorf(`this part should not be reached when getting a field from a json`)
}

/* Receives a string to be read as json as 1st argument followed by one or more strings that compose a path to a value 
inside the json structure. Returns the value as string, the value data type as int and may return a non nil error. There 
may be short cuts implemented which impose constraints on how the json can be formed. The trade off is parsing speed.*/
func Get (json []byte, keys []string) (string, DataType, error) {
	length := len(json)
	amountOfKeys := len(keys)
	i := 0 // index inside the string.

	if amountOfKeys == 0 {
		for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
		return getValue(json, i)
	}

	keyIndex := 0 // index in keys slice.
	key := keys[keyIndex] // first key.

	Structure:
	for {
		// fmt.Printf("[%d] c='%c'. value=%v, keyIndex=%d, len(stack)=%d, path=%d\n", i, json[i], value, keyIndex, len(stack), path) // debug.
		// fmt.Printf("-- [keyIndex: %d] reading value.\n", keyIndex) // debug.

		/* Between 'space', 'tab', 'return' and 'new line' chars, the 'space' char has the biggest byte number and they 
		are all bellow any other important char. */
		for json[i] <= ' '{ i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.

		// C:
		switch json[i] { // matching char with a type of value.
		case '{': // objects.
			// fmt.Printf("'{'\n") // debug
			i++
			for json[i] <= ' '{ i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // skipping spaces, tab, returns and new lines.
			if json[i] == '}' { return "", 0, fmt.Errorf(`JSON path %v could not be found.`, keys[:keyIndex+1]) }

			// reading keys.
			for {
				// fmt.Printf("-- [keyIndex: %d] reading key.\n", keyIndex) // debug
				for json[i] != '"' { i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // skipping every char that isn't a double quote.
				i++
				stringStart := i
				// finding end of this key.
				for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
				// fmt.Printf("trying to match with key '%s'.\n", key) // debug.
				// trying to match this key with current given key.
				keyRead := json[stringStart:i]
				for json[i] != ':' { i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // searching key-value separator.
				i++

				if len(key) == len(keyRead) && key == btos(keyRead) {
					// fmt.Printf("keyIndex: %d, keys %v, key: %s.\n", keyIndex, keys, key) // debug.
					keyIndex++
					if amountOfKeys > keyIndex { // if there are more keys to traverse.
						key = keys[keyIndex]
						continue Structure
					} else { // final value.
						for json[i] <= ' ' { i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.
						c := json[i]
						switch {
						case c == '"':
							i++
							valueStart := i
							for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
							s := json[valueStart:i]
							return btos(bytes.ReplaceAll(s, []byte{'\\','\\'}, []byte{'\\'})), JsonString, nil
						case c == '{':
							valueStart := i
							nest := 0
							for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
								case '{': nest++
								case '}': nest--
								}
							}
							s := json[valueStart:i]
							return btos(s), JsonObject, nil
						case c == '[':
							valueStart := i
							nest := 0
							for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
								case '[': nest++
								case ']': nest--
								}
							}
							s := json[valueStart:i]
							return btos(s), JsonArray, nil
						case c >= '0' && c <= '9' || c == '-':
							valueStart := i
							i++
							for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}
							if json[i]=='.'{i++;for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}}
							s := json[valueStart:i]
							return btos(s), JsonNumber, nil
						case c == 't':
							s := json[i:i+4]
							return btos(s), JsonBoolean, nil
						case c == 'f':
							s := json[i:i+5]
							return btos(s), JsonBoolean, nil
						case c == 'n':
							s := json[i:i+4]
							return btos(s), JsonNull, nil
						default: return "", 0, &JSONBadSyntaxError{c: c, i: i}
						}
					}
				}
				// wrong key, ignoring value.
				// fmt.Printf("key found '%s' not correct.\n", json[stringStart:i]) // debug

				for json[i] <= ' ' { i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.
				c := json[i]
				switch {
				case c == '"':
					i++
					for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
					i++
				case c == '{':
					nest := 0
					for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
						case '{': nest++
						case '}': nest--
						}
					}
				case c == '[':
					nest := 0
					for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
						case '[': nest++
						case ']': nest--
						}
					}
				case c >= '0' && c <= '9' || c == '-':
					i++
					for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}
					if json[i]=='.'{i++;for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}}
				case c == 't': i+=4
				case c == 'f': i+=5
				case c == 'n': i+=4
				default: return "", 0, &JSONBadSyntaxError{c: c, i: i}
				}
				for json[i] <= ' ' { i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.
				// comma will be skipped, eventually, because of the 'i++' and 'for json[i] <= ' ' { i++ }' later on.
				if json[i] == '}' {
					// fmt.Printf("-- pos: %d, sample: '%s', char: %c\n", i, string(json[i-10:i+10]), json[i])
					return "", 0, fmt.Errorf(`JSON key "%s" not found in %v.`, key, keys[:keyIndex+1])
				}
				if json[i] != ',' {
					return "", 0, fmt.Errorf(`Unrecognized character '%c' at position %d. Expected comma ',' or closing curly braces '}'.`, json[i], i)
				}
				i++
			}
			continue
		case '[': // arrays.
			// fmt.Printf("'['\n") // debug.
			i++
			if json[i] == ' ' {
				i++
				for json[i] <= ' ' { i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.
			}
			if json[i] == ']' { return "", 0, fmt.Errorf(`JSON path %v could not be found.`, keys[:keyIndex+1]) }
			keyNumeric, err := strconv.Atoi(key)
			if err != nil { return "", 0, fmt.Errorf("Could not convert '%s' to number in given keys %v.", key, keys[:keyIndex+1]) } 
			// else {
			// 	fmt.Printf("-- converted %s to number\n", key)
			// }
			arrayIndex := 0
			for { // looping over 'arrayIndex'.
				// c := json[i]
				// fmt.Printf("-- !! char: '%c'\n", json[i]) // debug
				if arrayIndex == keyNumeric { // we are at the array index we want.
					keyIndex++
					if amountOfKeys > keyIndex { // there are more keys.
						switch json[i] {
						case '{', '[': 
							key = keys[keyIndex]
							continue Structure
						default: return "", 0, fmt.Errorf("JSON %v is not an structure. Can't traverse further.", keys[:keyIndex+1])
						}
					} else { // final value.
						c := json[i]
						switch {
						case c == '"':
							valueStart := i
							i++
							for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
							s := json[valueStart:i]
							return btos(bytes.ReplaceAll(s, []byte{'\\','\\'}, []byte{'\\'})), JsonString, nil
						case c >= '0' && c <= '9' || c == '-':
							valueStart := i
							i++
							for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}
							if json[i]=='.'{i++;for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}}
							s := json[valueStart:i]
							return btos(s), JsonNumber, nil
						case c == '{':
							valueStart := i
							nest := 0
							for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
								case '{': nest++
								case '}': nest--
								}
							}
							s := json[valueStart:i]
							return btos(s), JsonObject, nil
						case c == '[':
							valueStart := i
							nest := 0
							for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
								case '[': nest++
								case ']': nest--
								}
							}
							s := json[valueStart:i]
							return btos(s), JsonArray, nil
						case c == 't':
							s := json[i:i+4]
							return btos(s), JsonBoolean, nil
						case c == 'f':
							s := json[i:i+5]
							return btos(s), JsonBoolean, nil
						case c == 'n':
							s := json[i:i+4]
							return btos(s), JsonNull, nil
						default: return "", 0, &JSONBadSyntaxError{c: c, i: i}
						}
					}
				}
				// ignoring value.
				c := json[i]
				switch {
				case c == '"':
					i++
					for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
					i++
				case c >= '0' && c <= '9' || c == '-':
					i++
					for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}
					if json[i]=='.'{i++;for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}}
				case c == '{':
					nest := 0
					for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
						case '{': nest++
						case '}': nest--
						}
					}
				case c == '[':
					nest := 0
					for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
						case '[': nest++
						case ']': nest--
						}
					}
				case c == 't': i+=4
				case c == 'f': i+=5
				case c == 'n': i+=4
				default: return "", 0, &JSONBadSyntaxError{c: c, i: i}
				}
				// fmt.Printf("-- >> sample '%s', char: '%c'\n", string(json[i-10:i+10]), json[i]) // debug.
				if json[i] == ' ' {
					i++
					for json[i] <= ' ' { i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.
				}
				if json[i] == ',' {
					i++ // skips comma.
					for json[i] <= ' ' { i++; if i>=length{return "",0,&JSONEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.
					arrayIndex++
				} else if json[i] == ']' {
					// fmt.Printf("-- pos: %d, sample: '%s', char: %c\n", i, string(json[i-10:i+10]), json[i])
					return "", 0, fmt.Errorf(`Index "%d" not found at structure %v.`, keyNumeric, keys[:keyIndex+1])
				} else {
					// fmt.Printf("-- __ sample '%s', char: %c\n", string(json[i-10:i+10]), json[i]) // debug.
					return "", 0, fmt.Errorf(`Unrecognized character '%c' at position %d. Expected comma ',' or closing brackets '].`, json[i], i)
				}
			}
		default: return "", 0, fmt.Errorf("JSON %v is not an structure. Can't traverse further.", keys[:keyIndex+1])
		}
	}

	return "", 0, fmt.Errorf(`this part should not be reached when getting a field from a json.`)
}

func getValue (json []byte, i int) (string, DataType, error) {
	length := len(json)
	c := json[i]
	switch {
	case c == '"':
		valueStart := i
		i++
		for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
		s := json[valueStart:i]
		return btos(bytes.ReplaceAll(s, []byte{'\\','\\'}, []byte{'\\'})), JsonString, nil
	case c == '{':
		valueStart := i
		nest := 0
		for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
			switch json[i] {
			case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
			case '{': nest++
			case '}': nest--
			}
		}
		s := json[valueStart:i]
		return btos(s), JsonObject, nil
	case c == '[':
		valueStart := i
		nest := 0
		for i++; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
			switch json[i] {
			case '"': for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]=='"'{if json[i-1]!='\\'{break};slashesCount := 1;for j:=i-2;json[j]=='\\';j--{slashesCount++};if slashesCount%2==0{break}};i++}
			case '[': nest++
			case ']': nest--
			}
		}
		s := json[valueStart:i]
		return btos(s), JsonArray, nil
	case c >= '0' && c <= '9' || c == '-':
		valueStart := i
		i++
		for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}
		if json[i]=='.'{i++;for{if i>=length{return "",0,&JSONEndsAbruptlyError{}};if json[i]>='0'&&json[i]<='9'{i++}else{break}}}
		s := json[valueStart:i]
		return btos(s), JsonNumber, nil
	case c == 't':
		s := json[i:i+4]
		return btos(s), JsonBoolean, nil
	case c == 'f':
		s := json[i:i+5]
		return btos(s), JsonBoolean, nil
	case c == 'n':
		s := json[i:i+4]
		return btos(s), JsonNull, nil
	default: return "", 0, &JSONBadSyntaxError{c: c, i: i}
	}
}

// func main () {
// 	s, d, e := Get([]byte(`{"aaa\\": "abc\\"}`), []string{"aaa"})
// 	fmt.Println(s,d,e)
// }