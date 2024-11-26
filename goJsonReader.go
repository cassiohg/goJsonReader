package goJsonReader

import (
	"fmt"
	"strconv"
	"unsafe"
	"reflect"
	"bytes"
	"github.com/tidwall/gjson"
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

type JsonEndsAbruptlyError struct {}
func (e *JsonEndsAbruptlyError) Error () string { return "JSON ends abruptly." }
type JsonBadSyntaxError struct {c byte; i int}
func (e *JsonBadSyntaxError) Error () string { return fmt.Sprintf("Bad JSON syntax '%c' at %d.", e.c, e.i) }


func jsonUnescape(json []byte) []byte {
	unescapedString := make([]byte, 0, len(json))
	for i := 0; i < len(json); i++ {
		if json[i] != '\\'{ unescapedString = append(unescapedString, json[i]); continue }
		switch json[i+1] {
		case 'b': unescapedString = append(unescapedString, '\b')
		case 'f': unescapedString = append(unescapedString, '\f')
		case 'n': unescapedString = append(unescapedString, '\n')
		case 't': unescapedString = append(unescapedString, '\t')
		case 'r': unescapedString = append(unescapedString, '\r')
		case '"': unescapedString = append(unescapedString, '"')
		case '\\': unescapedString = append(unescapedString, '\\')
		}
		i++
	}
	return unescapedString;
}

// Receives a string to be read as json as 1st argument followed by one or more strings that compose one path to a value 
// inside the json structure. Returns the value as string, the value data type as int and may return a non nil error. There 
// may be short cuts implemented which impose constraints on how the json can be formed. The trade off is parsing speed.
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
			for json[i] <= ' ' { i++; if i>=length{return &JsonEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.

			switch json[i] { // matching char with a type of value.
			case '{': // objects.
				// fmt.Printf("'{'\n") // debug
				i++
				if i>=length{return &JsonEndsAbruptlyError{}}
				for json[i] <= ' ' { i++; if i>=length{return &JsonEndsAbruptlyError{}} } // skipping spaces, tab, returns and new lines.
				if path == 1 {
					if json[i] != '}' {
						stack = append(stack, '{') // when an object, we have another structure to traverse.
						value = false
						continue
					}
				} else if path == 2 {
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
							for nest := 0 ; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': Str0:for{i++;if i>=length{return &JsonEndsAbruptlyError{}};for json[i]!='"' { i++; if i>=length{return &JsonEndsAbruptlyError{}} };if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str0};j--}}
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
					for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': Str1:for{i++;if i>=length{return &JsonEndsAbruptlyError{}};for json[i]!='"' { i++; if i>=length{return &JsonEndsAbruptlyError{}} };if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str1};j--}}
						case '{': nest++
						case '}': nest--
						}
					}
					// fmt.Printf("skipped whole object: %v\n", json[valueStart:i]) // debug.
				}
			case '[': // arrays.
				// fmt.Printf("'['\n") // debug.
				i++
				if i>=length{return &JsonEndsAbruptlyError{}}
				for json[i] <= ' ' { i++; if i>=length{return &JsonEndsAbruptlyError{}} } // skipping spaces, tab, returns and new lines.
				if path == 1 {
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
					bufferEach = true
					if json[i] != ']' {
						valueStart := i
						for nest := 0; nest > -1; i++ { // will consider only brackets out of strings (both keys and values).
							switch json[i] {
							case '"': Str2:for{i++;if i>=length{return &JsonEndsAbruptlyError{}};for json[i]!='"' { i++; if i>=length{return &JsonEndsAbruptlyError{}} };if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str2};j--}}
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
					for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': Str3:for{i++;if i>=length{return &JsonEndsAbruptlyError{}};for json[i]!='"' { i++; if i>=length{return &JsonEndsAbruptlyError{}} };if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str3};j--}}
						case '[': nest++
						case ']': nest--
						}
					}
					// fmt.Printf("skipped whole array: %v\n", json[valueStart:i]) // debug.
				}
			case '"': // strings.
				// fmt.Printf("start of a string\n") // debug.
				i++
				if i>=length{return &JsonEndsAbruptlyError{}}
				k := i
				// searching end of string.
				Str4:for{for json[i]!='"' { i++; if i>=length{return &JsonEndsAbruptlyError{}} };if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str4};j--};i++}
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
			for json[i] <= ' ' { i++; if i>=length{return &JsonEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.

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
			for json[i] <= ' ' { i++; if i>=length{return &JsonEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.
		} else { // reading key
			// fmt.Printf("-- [keyIndex: %d] reading key.\n", keyIndex) // debug
			value = true
			for json[i] != '"' { i++; if i>=length{return &JsonEndsAbruptlyError{}} } // skipping every char that isn't a double quote.
			i++
			if i>=length{return &JsonEndsAbruptlyError{}}

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
					// searching key end.
					Str5:for{for json[i]!='"' { i++; if i>=length{return &JsonEndsAbruptlyError{}} };if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str5};j--};i++}
					// fmt.Printf("key found %s not correct.\n", json[k-j-1:i+1]) // debug
				}
			} else {
			// 	// k := i // debug
			// 	for json[i] != '"' { i++ } // searching key end.
			// 	// fmt.Printf("ignoring key '%s'.\n", json[k-1:i+1]) // debug
			// 	for json[i] != ':' { i++ } // searching key-value separator.
				k := i
				Str6:for{for json[i]!='"' { i++; if i>=length{return &JsonEndsAbruptlyError{}} };if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str6};j--};i++}
				lastKey = btos(json[k:i])
				// fmt.Printf("lastKey '%s'.\n", lastKey) // debug
			}
			for json[i] != ':' { i++; if i>=length{return &JsonEndsAbruptlyError{}} }
			i++
			if i>=length{return &JsonEndsAbruptlyError{}}
			for json[i] <= ' ' { i++; if i>=length{return &JsonEndsAbruptlyError{}} } // skipping spaces, tabs, returns and new lines.
		}
	}

	return fmt.Errorf(`this part should not be reached when getting a field from a json`)
}

// Receives a string to be read as json as 1st argument followed by one or more strings that compose a path to a value 
// inside the json structure. Returns the value as string, the value data type as int and may return a non nil error. There 
// may be short cuts implemented which impose constraints on how the json can be formed. The trade off is parsing speed.
func getPanic (json []byte, keys []string) (string, DataType, error) {
	// length := len(json)
	amountOfKeys := len(keys)
	if amountOfKeys == 0 { return getValue(json, 0) }
	keyIndex := 0 // index in keys slice.
	key := keys[keyIndex] // first key.
	i := 0 // index inside the json.

	Structure:
	for {
		// fmt.Printf("[%d] c='%c'. value=%v, keyIndex=%d, len(stack)=%d, path=%d\n", i, json[i], value, keyIndex, len(stack), path) // debug.
		// fmt.Printf("-- [keyIndex: %d] reading value.\n", keyIndex) // debug.

		/* Between 'space', 'tab', 'return' and 'new line' chars, the 'space' char has the biggest byte number and they 
		are all bellow any other important char. */
		for json[i] <= ' '{ i++ } // skipping spaces, tabs, returns and new lines.

		switch json[i] { // matching char with a type of value.
		case '{': // objects.
			// fmt.Printf("'{'\n") // debug
			i++
			for json[i] <= ' '{ i++ } // skipping spaces, tab, returns and new lines.
			if json[i] == '}' { return "", 0, fmt.Errorf(`JSON path %v could not be found.`, keys[:keyIndex+1]) }

			// reading keys.
			for {
				// fmt.Printf("-- [keyIndex: %d] reading key.\n", keyIndex) // debug
				for json[i] != '"' { i++ } // skipping every char that isn't a double quote.
				keyStart := i
				// fmt.Printf("key start at '%v'.\n", keyStart) // debug.
				// finding end of this key.
				hasEscape := false;
				Str0:for{i++;for json[i]!='"' {i++;if json[i] == '\\'{hasEscape=true}};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str0};j--}}
				// fmt.Printf("trying to match with key '%s'.\n", key) // debug.
				// fmt.Printf("key end at '%v'.\n", i) // debug.
				// trying to match this key with current given key.
				s := json[keyStart+1:i]
				if (hasEscape) { s = jsonUnescape(s) }
				// fmt.Printf("found key '%s'.\n", s) // debug.
				keyRead := btos(s)

				for json[i] != ':' { i++ } // searching key-value separator.
				i++

				if len(key) == len(keyRead) && key == keyRead {
					// fmt.Printf("keyIndex: %d, keys %v, key: %s.\n", keyIndex, keys, key) // debug.
					keyIndex++
					if amountOfKeys > keyIndex { // if there are more keys to traverse.
						key = keys[keyIndex]
						continue Structure
					} else { // final value.
						for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
						switch json[i] {
						case '"':
							valueStart := i
							hasEscape := false;
							Str1:for{i++;for json[i]!='"' {i++;if json[i] == '\\'{hasEscape=true}};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str1};j--}}
							s := json[valueStart+1:i]
							if (hasEscape) { s = jsonUnescape(s) }
							return btos(s), JsonString, nil
						case '{':
							valueStart := i
							i++
							for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': Str2:for{for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str2};j--};i++}
								case '{': nest++
								case '}': nest--
								}
							}
							return btos(json[valueStart:i]), JsonObject, nil
						case '[':
							valueStart := i
							i++
							for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': Str3:for{for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str3};j--};i++}
								case '[': nest++
								case ']': nest--
								}
							}
							return btos(json[valueStart:i]), JsonArray, nil
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
							valueStart := i
							i++
							for json[i] >= '0' && json[i] <= '9' { i++ }
							if json[i]=='.' {
								i++;
								for json[i] >= '0' && json[i] <='9' { i++ }
							}
							return btos(json[valueStart:i]), JsonNumber, nil
						case 't': return btos(json[i:i+4]), JsonBoolean, nil
						case 'f': return btos(json[i:i+5]), JsonBoolean, nil
						case 'n': return btos(json[i:i+4]), JsonNull, nil
						default: return "", 0, &JsonBadSyntaxError{c: json[i], i: i}
						}
					}
				}
				// wrong key, ignoring value.
				// fmt.Printf("key found '%s' not correct.\n", json[keyStart:i]) // debug

				for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
				switch json[i] {
				case '"':
					Str4:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str4};j--}}
					i++
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
					i++
					for json[i] >= '0' && json[i] <= '9' { i++ }
					if json[i]=='.' {
						i++;
						for json[i] >= '0' && json[i] <='9' { i++ }
					}
				case '{':
					i++
					for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': Str5:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str5};j--}}
						case '{': nest++
						case '}': nest--
						}
					}
				case '[':
					i++
					for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': Str6:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str6};j--}}
						case '[': nest++
						case ']': nest--
						}
					}
				case 't': i += 4
				case 'f': i += 5
				case 'n': i += 4
				default: return "", 0, &JsonBadSyntaxError{c: json[i], i: i}
				}
				for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.

				if json[i] == '}' {
					// fmt.Printf("-- pos: %d, sample: '%s', char: %c\n", i, string(json[i-10:i+10]), json[i])
					return "", JsonObject, fmt.Errorf(`JSON key "%s" not found in structure %v.`, key, keys[:keyIndex])
				}
				if json[i] != ',' {
					return "", JsonObject, fmt.Errorf(`Unrecognized character '%c' at position %d. Expected comma ',' or closing curly braces '}'.`, json[i], i)
				}
				i++
			}
			continue
		case '[': // arrays.
			// fmt.Printf("'['\n") // debug.
			i++
			for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
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
						default: return "", 0, fmt.Errorf("JSON value at path %v is not an structure. Can't traverse further to find path %v.", keys[:keyIndex], keys)
						}
					} else { // final value.
						switch json[i] {
						case '"':
							valueStart := i
							hasEscape := false;
							Str7:for{i++;for json[i]!='"' {i++;if json[i] == '\\'{hasEscape=true}};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str7};j--}}
							s := json[valueStart+1:i]
							if (hasEscape) { s = jsonUnescape(s) }
							return btos(s), JsonString, nil
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
							valueStart := i
							i++
							for json[i] >= '0' && json[i] <= '9' { i++ }
							if json[i]=='.' {
								i++;
								for json[i] >= '0' && json[i] <='9' { i++ }
							}
							return btos(json[valueStart:i]), JsonNumber, nil
						case '{':
							valueStart := i
							i++
							for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': Str8:for{for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str8};j--};i++}
								case '{': nest++
								case '}': nest--
								}
							}
							return btos(json[valueStart:i]), JsonObject, nil
						case '[':
							valueStart := i
							i++
							for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
								switch json[i] {
								case '"': Str9:for{for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str9};j--};i++}
								case '[': nest++
								case ']': nest--
								}
							}
							return btos(json[valueStart:i]), JsonArray, nil
						case 't': return btos(json[i:i+4]), JsonBoolean, nil
						case 'f': return btos(json[i:i+5]), JsonBoolean, nil
						case 'n': return btos(json[i:i+4]), JsonNull, nil
						default: return "", 0, &JsonBadSyntaxError{c: json[i], i: i}
						}
					}
				}
				// ignoring value.
				switch json[i] {
				case '"':
					Str10:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str10};j--}}
					i++
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
					i++
					for json[i] >= '0' && json[i] <= '9' { i++ }
					if json[i]=='.' {
						i++;
						for json[i] >= '0' && json[i] <='9' { i++ }
					}
				case '{':
					i++
					for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': Str11:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str11};j--}}
						case '{': nest++
						case '}': nest--
						}
					}
				case '[':
					i++
					for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
						switch json[i] {
						case '"': Str12:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str12};j--}}
						case '[': nest++
						case ']': nest--
						}
					}
				case 't': i+=4
				case 'f': i+=5
				case 'n': i+=4
				default: return "", 0, &JsonBadSyntaxError{c: json[i], i: i}
				}
				// fmt.Printf("-- >> sample '%s', char: '%c'\n", string(json[i-10:i+10]), json[i]) // debug.
				for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
				if json[i] == ',' {
					i++ // skips comma.
					for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
					arrayIndex++
				} else if json[i] == ']' {
					// fmt.Printf("-- pos: %d, sample: '%s', char: %c\n", i, string(json[i-10:i+10]), json[i])
					return "", JsonArray, fmt.Errorf(`Index "%d" not found in structure %v.`, keyNumeric, keys[:keyIndex])
				} else {
					// fmt.Printf("-- __ sample '%s', char: %c\n", string(json[i-10:i+10]), json[i]) // debug.
					return "", JsonArray, fmt.Errorf(`Unrecognized character '%c' at position %d. Expected comma ',' or closing brackets '].`, json[i], i)
				}
			}
		default: return "", 0, fmt.Errorf("JSON value at path %v is not an structure. Can't traverse further to find path %v.", keys[:keyIndex], keys)
		}
	}

	return "", 0, fmt.Errorf(`this part should not be reached when getting a field from a json.`)
}

func Get (json []byte, keys []string) (s string, d DataType, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf(`Panic reading Json: %s`, err)
		}
	}()
	return getPanic(json, keys)
}

// Same function but uses function calls instead of inlining. Varies between 20% and 4% worse than its equivalent.
func getPanic2 (json []byte, keys []string) (string, DataType, error) {
	// length := len(json)
	amountOfKeys := len(keys)
	if amountOfKeys == 0 { return getValue(json, 0) }
	keyIndex := 0 // index in keys slice.
	key := keys[keyIndex] // first key.
	i := 0 // index inside the json.

	Structure:
	for {
		// fmt.Printf("[%d] c='%c'. value=%v, keyIndex=%d, len(stack)=%d, path=%d\n", i, json[i], value, keyIndex, len(stack), path) // debug.
		// fmt.Printf("-- [keyIndex: %d] reading value.\n", keyIndex) // debug.

		/* Between 'space', 'tab', 'return' and 'new line' chars, the 'space' char has the biggest byte number and they 
		are all bellow any other important char. */
		for json[i] <= ' '{ i++ } // skipping spaces, tabs, returns and new lines.

		switch json[i] { // matching char with a type of value.
		case '{': // objects.
			// fmt.Printf("'{'\n") // debug
			i++
			for json[i] <= ' '{ i++ } // skipping spaces, tab, returns and new lines.
			if json[i] == '}' { return "", 0, fmt.Errorf(`JSON path %v could not be found.`, keys[:keyIndex+1]) }

			// reading keys.
			for {
				// fmt.Printf("-- [keyIndex: %d] reading key.\n", keyIndex) // debug
				for json[i] != '"' { i++ } // skipping every char that isn't a double quote.
				keyStart := i
				// fmt.Printf("key start at '%v'.\n", keyStart) // debug.
				// finding end of this key.
				hasEscape := false;
				Str0:for{i++;for json[i]!='"' {i++;if json[i] == '\\'{hasEscape=true}};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str0};j--}}
				/// fmt.Printf("trying to match with key '%s'.\n", key) // debug.
				// fmt.Printf("key end at '%v'.\n", i) // debug.
				// trying to match this key with current given key.
				s := json[keyStart+1:i]
				if (hasEscape) { s = jsonUnescape(s) }
				// fmt.Printf("found key '%s'.\n", s) // debug.
				keyRead := btos(s)

				for json[i] != ':' { i++ } // searching key-value separator.
				i++

				if len(key) == len(keyRead) && key == keyRead {
					// fmt.Printf("keyIndex: %d, keys %v, key: %s.\n", keyIndex, keys, key) // debug.
					keyIndex++
					if amountOfKeys > keyIndex { // if there are more keys to traverse.
						key = keys[keyIndex]
						continue Structure
					} else { // final value.
						return getValue(json, i)
					}
				}

				// wrong key, ignoring value.
				// fmt.Printf("key found '%s' not correct.\n", json[keyStart:i]) // debug
				for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
				if err := skipValue(json, &i); err != nil { return "", JsonObject, err }

				for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.

				if json[i] == '}' {
					// fmt.Printf("-- pos: %d, sample: '%s', char: %c\n", i, string(json[i-10:i+10]), json[i])
					return "", JsonObject, fmt.Errorf(`JSON key "%s" not found in structure %v.`, key, keys[:keyIndex])
				}
				if json[i] != ',' {
					return "", JsonObject, fmt.Errorf(`Unrecognized character '%c' at position %d. Expected comma ',' or closing curly braces '}'.`, json[i], i)
				}
				i++
			}
			continue
		case '[': // arrays.
			// fmt.Printf("'['\n") // debug.
			i++
			for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
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
						default: return "", 0, fmt.Errorf("JSON value at path %v is not an structure. Can't traverse further to find path %v.", keys[:keyIndex], keys)
						}
					} else { // final value.
						return getValue(json, i)
					}
				}

				// ignoring value.
				if err := skipValue(json, &i); err != nil { return "", JsonArray, err }

				// fmt.Printf("-- >> sample '%s', char: '%c'\n", string(json[i-10:i+10]), json[i]) // debug.
				for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
				if json[i] == ',' {
					i++ // skips comma.
					for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
					arrayIndex++
				} else if json[i] == ']' {
					// fmt.Printf("-- pos: %d, sample: '%s', char: %c\n", i, string(json[i-10:i+10]), json[i])
					return "", JsonArray, fmt.Errorf(`Index "%d" not found in structure %v.`, keyNumeric, keys[:keyIndex])
				} else {
					// fmt.Printf("-- __ sample '%s', char: %c\n", string(json[i-10:i+10]), json[i]) // debug.
					return "", JsonArray, fmt.Errorf(`Unrecognized character '%c' at position %d. Expected comma ',' or closing brackets '].`, json[i], i)
				}
			}
		default: return "", 0, fmt.Errorf("JSON value at path %v is not an structure. Can't traverse further to find path %v.", keys[:keyIndex], keys)
		}
	}

	return "", 0, fmt.Errorf(`this part should not be reached when getting a field from a json.`)
}

func getValue (json []byte, i int) (string, DataType, error) {
	// length := len(json)
	for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
	switch json[i] {
	case '"':
		valueStart := i
		hasEscape := false;
		Str0:for{i++;for json[i]!='"' {i++;if json[i] == '\\'{hasEscape=true}};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str0};j--}}
		s := json[valueStart+1:i]
		if (hasEscape) { s = jsonUnescape(s) }
		return btos(s), JsonString, nil
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
		valueStart := i
		i++
		for json[i] >= '0' && json[i] <= '9' { i++ }
		if json[i]=='.' {
			i++;
			for json[i] >= '0' && json[i] <='9' { i++ }
		}
		return btos(json[valueStart:i]), JsonNumber, nil
	case '{':
		valueStart := i
		i++
		for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
			switch json[i] {
			case '"': Str1:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str1};j--}}
			case '{': nest++
			case '}': nest--
			}
		}
		return btos(json[valueStart:i]), JsonObject, nil
	case '[':
		valueStart := i
		i++
		for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
			switch json[i] {
			case '"': Str2:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str2};j--}}
			case '[': nest++
			case ']': nest--
			}
		}
		return btos(json[valueStart:i]), JsonArray, nil
	case 't': return btos(json[i:i+4]), JsonBoolean, nil
	case 'f': return btos(json[i:i+5]), JsonBoolean, nil
	case 'n': return btos(json[i:i+4]), JsonNull, nil
	default: return "", 0, &JsonBadSyntaxError{c: json[i], i: i}
	}
}
func skipValue (json []byte, i *int) error {
	switch json[*i] {
	case '"':
		Str10:for{*i++;for json[*i]!='"' {*i++};if json[*i-1]!='\\'{break};j := *i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str10};j--}}
		*i++
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
		*i++
		for json[*i] >= '0' && json[*i] <= '9' { *i++ }
		if json[*i]=='.' {
			*i++;
			for json[*i] >= '0' && json[*i] <='9' { *i++ }
		}
	case '{':
		*i++
		for nest := 0; nest > -1; *i++ { // will consider only curly braces out of strings (both keys and values).
			switch json[*i] {
			case '"': Str11:for{*i++;for json[*i]!='"' {*i++};if json[*i-1]!='\\'{break};j := *i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str11};j--}}
			case '{': nest++
			case '}': nest--
			}
		}
	case '[':
		*i++
		for nest := 0; nest > -1; *i++ { // will consider only curly braces out of strings (both keys and values).
			switch json[*i] {
			case '"': Str12:for{*i++;for json[*i]!='"' {*i++};if json[*i-1]!='\\'{break};j := *i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str12};j--}}
			case '[': nest++
			case ']': nest--
			}
		}
	case 't': *i+=4
	case 'f': *i+=5
	case 'n': *i+=4
	default: return &JsonBadSyntaxError{c: json[*i], i: *i}
	}
	return nil
}
func Get2 (json []byte, keys []string) (s string, d DataType, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf(`Panic reading Json: %s`, err)
		}
	}()
	return getPanic2(json, keys)
}
