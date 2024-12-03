package goJsonReader
// package main

import (
	"fmt"
	"strconv"
	"unsafe"
	"reflect"
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
	default: return fmt.Sprintf("Type[%v]",int8(d))
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

/* Arguments:
	1) byte array to be read as json.
	2) a callback function that will be executed at every element belonging to the top level structure found in the json.

	That callback function arguments:
	1) the numeric index representing the order when an element is found in the structure.
	2) the string key of that element, only in case the structure is an object, otherwise it's an empty string.
	3) the string containing the value of the element.
	4) the data type of that element.
	callback return:
	1) when false is returned, the callback will not be called again. This is the interrupt/break for the ForEach. When true
	is returned, the next element will be read and passed to the next call of this callback.

Return:
	1) error found during json read. This function does not validate the json, but will not panic if the json isn't
	correct. This is a trade off for speed. In case of a malformed json, the values read might be incorrect, or strange.
*/
func ForEach (json []byte, keys []string, each func(int, string, string, DataType) bool) (e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf(`Panic reading Json: %s`, err)
		}
	}()

	// If not path given, we get the value at that path before executing foreach.
	if keys != nil && len(keys) > 0 && keys[0] != "" {
		value, _, _ := getPanic(json, keys)
		json = stob(value)
	} else {
		i := 0
		for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
		json = json[i:]
	}
	return forEachPanic(json, each)
}
func forEachPanic (json []byte, each func(int, string, string, DataType) bool) error {
	i := 0

	var d DataType // The type of structure we are reading.
	switch json[i] {
	case '{': d = JsonObject
	case '[': d = JsonArray
	default: return fmt.Errorf("Cannot execute ForEach for value that isn't either an object or an array.")
	}

	index := 0
	keyRead := ""
	Loop: for {
		// Getting key if given json is an object. If it's an array 'keyRead' stays an empty string.
		if d == JsonObject {
			for json[i] != '"' { i++ } // skipping every char that isn't a double quote.
			keyStart := i
			// fmt.Printf("key start at '%v'.\n", keyStart) // debug.
			// Finding end of this key.
			hasEscape := false;
			Str0:for{i++;for json[i]!='"' {i++;if json[i] == '\\'{hasEscape=true}};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str0};j--}}
			// fmt.Printf("key end at '%v'.\n", i) // debug.
			s := json[keyStart+1:i]
			if (hasEscape) { s = jsonUnescape(s) }
			// fmt.Printf("found key '%s'.\n", s) // debug.
			keyRead = btos(s)

			for json[i] != ':' { i++ } // searching key-value separator.
		}
		i++
		for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.

		// Getting value.
		var value string
		var dataType DataType
		switch json[i] {
		case '"':
			dataType = JsonString
			valueStart := i
			hasEscape := false;
			Str1:for{i++;for json[i]!='"' {i++;if json[i] == '\\'{hasEscape=true}};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str1};j--}}
			s := json[valueStart+1:i]
			if (hasEscape) { s = jsonUnescape(s) }
			value = btos(s)
			i++
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			dataType = JsonNumber
			valueStart := i
			i++
			for json[i] >= '0' && json[i] <= '9' { i++ }
			if json[i]=='.' {
				i++;
				for json[i] >= '0' && json[i] <='9' { i++ }
			}
			value = btos(json[valueStart:i])
		case '{':
			dataType = JsonObject
			valueStart := i
			i++
			for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
				switch json[i] {
				case '"': Str2:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str2};j--}}
				case '{': nest++
				case '}': nest--
				}
			}
			value = btos(json[valueStart:i])
		case '[':
			dataType = JsonArray
			valueStart := i
			i++
			for nest := 0; nest > -1; i++ { // will consider only curly braces out of strings (both keys and values).
				switch json[i] {
				case '"': Str3:for{i++;for json[i]!='"' {i++};if json[i-1]!='\\'{break};j := i-2;for {if json[j]!='\\'{break};j--;if json[j]!='\\'{break Str3};j--}}
				case '[': nest++
				case ']': nest--
				}
			}
			value = btos(json[valueStart:i])
		case 't':
			dataType = JsonBoolean
			valueStart := i
			i += 4
			value = btos(json[valueStart:i])
		case 'f':
			dataType = JsonBoolean
			valueStart := i
			i += 5
			value = btos(json[valueStart:i])
		case 'n':
			dataType = JsonNull
			valueStart := i
			i += 4
			value = btos(json[valueStart:i])
		default: return &JsonBadSyntaxError{c: json[i], i: i}
		}

		if !each(index, keyRead, value, dataType) { return nil }
		index++

		for json[i] <= ' ' { i++ } // skipping spaces, tabs, returns and new lines.
		switch json[i] { // Looking for comma or end of object/array.
		case ',': continue
		case '}', ']': break Loop
		default: return &JsonBadSyntaxError{c: json[i], i: i}
		}
	}
	return nil
}

/* Arguments:
	1) byte array to be read as json.
	2) the path where the target value is located inside the given json

Return:
	1) the string containing the target value inside the given json.
	2) the data type of that value.
	3) the error found during json read. This function does not validate the json, but will not panic if the json isn't
	correct. This is a trade off for speed. In case of a malformed json, the values read might be incorrect, or strange.
*/
func Get (json []byte, path []string) (s string, d DataType, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf(`Panic reading Json: %s`, err)
		}
	}()
	return getPanic(json, path)
}
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


func Get2 (json []byte, keys []string) (s string, d DataType, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf(`Panic reading Json: %s`, err)
		}
	}()
	return getPanic2(json, keys)
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

