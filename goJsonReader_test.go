package goJsonReader

import (
	"testing"
	"strings"
)

func printError(t *testing.T, path []string,
	expectedValue string, expectedDataType DataType,
	returnedValue string, returnedDataType DataType,
	e error,
) {
	if e != nil {
		t.Errorf("Getting path=%v from json object returned the following error %v.\n", path, e)
	}
	if returnedValue != expectedValue {
		t.Errorf("Getting path=%v from json object returned unexpect value \"%v\".\n", path, returnedValue)
	}
	if returnedDataType != expectedDataType {
		t.Errorf("Getting path=%v from json object returned unexpect dataType=%v.\n", path, returnedDataType)
	}
}

func TestGettingTopLevelKeyWithStringValue(t *testing.T) {
	path := []string{"abc"}
	expectedValue := "aaaaaac"
	expectedDataType := JsonString
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingDeeplyNestedKeyWithStringValue(t *testing.T) {
	path := strings.Split("bigObject.service.attributes.commission.test", ".")
	expectedValue := "test"
	expectedDataType := JsonString
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingTopLevelKeyWithNumberValue(t *testing.T) {
	path := []string{"bcryptCost"}
	expectedValue := "10"
	expectedDataType := JsonNumber
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}


func TestGettingDeeplyNestedKeyWitNegativeFloatingPointNumberValue(t *testing.T) {
	path := strings.Split("bigObject.service.attributes.commission.number", ".")
	expectedValue := "-123456789.123"
	expectedDataType := JsonNumber
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingTopLevelKeyWithNullValue(t *testing.T) {
	path := []string{"aNUllValue"}
	expectedValue := "null"
	expectedDataType := JsonNull
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingDeeplyNestedKeyWitNullValue(t *testing.T) {
	path := strings.Split("bigObject.service.attributes.commission.nullValue", ".")
	expectedValue := "null"
	expectedDataType := JsonNull
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingTopLevelKeyWithBooleanTrueValue(t *testing.T) {
	path := []string{"aTrueValue"}
	expectedValue := "true"
	expectedDataType := JsonBoolean
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingDeeplyNestedKeyWitBooleanTrueValue(t *testing.T) {
	path := strings.Split("bigObject.service.attributes.commission.aTrueValue", ".")
	expectedValue := "true"
	expectedDataType := JsonBoolean
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingTopLevelKeyWithBooleanFalseValue(t *testing.T) {
	path := []string{"aFalseValue"}
	expectedValue := "false"
	expectedDataType := JsonBoolean
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingDeeplyNestedKeyWitBooleanFalseValue(t *testing.T) {
	path := strings.Split("bigObject.service.attributes.commission.aFalseValue", ".")
	expectedValue := "false"
	expectedDataType := JsonBoolean
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingTopLevelKeyWithEmptyArrayValue(t *testing.T) {
	path := []string{"emptyArray"}
	expectedValue := `[]`
	expectedDataType := JsonArray
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingTopLevelKeyWithArrayValue(t *testing.T) {
	path := []string{"anArray"}
	expectedValue := arrayTopLevel
	expectedDataType := JsonArray
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingDeeplyNestedKeyWitArrayValue(t *testing.T) {
	path := strings.Split("bigObject.service.attributes.commission.array", ".")
	expectedValue := arrayDeeplyNested
	expectedDataType := JsonArray
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingAnElementInArrayJson(t *testing.T) {
	path := strings.Split("3.1.aaa", ".")
	expectedValue := aNumber
	expectedDataType := JsonNumber
	s, d, e := Get([]byte(arrayDeeplyNested), path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingTopLevelKeyWithEmptyObjectValue(t *testing.T) {
	path := []string{"emptyObject"}
	expectedValue := `{}`
	expectedDataType := JsonObject
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingTopLevelKeyWithObjectValue(t *testing.T) {
	path := []string{"bigObject"}
	expectedValue := bigObject
	expectedDataType := JsonObject
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestGettingDeeplyNestedKeyWitObjectValue(t *testing.T) {
	path := strings.Split("bigObject.service.attributes.commission.array.3.1", ".")
	expectedValue := smallObject
	expectedDataType := JsonObject
	s, d, e := Get(jsonBytes, path)
	printError(t, path, expectedValue, expectedDataType, s, d, e)
}

func TestForEachKeysInTopLevelObject(t *testing.T) {
	path := []string{""}
	e := ForEach([]byte(smallObject), path, func (key, value string, d DataType, e error) bool {
		expectedValue := aNumber
		expectedDataType := JsonNumber
		if key != "aaa" {
			t.Errorf("Unexpected key=\"%v\" from path=%v in json object.\n", key, path)
		}
		printError(t, path, expectedValue, expectedDataType, value, d, e)
		return true
	})
	if e != nil {
		t.Errorf("Getting path=%v from json object returned the following error %v.\n", path, e)
	}
}

var client = `{
				"attributes": {
					"full name": "string",
					"short name": "string",
					"initials": "string",
					"birth date": "date",
					"national ID": {
						"type": "string",
						"mask": "national-id-of-country-x"
					}
				},
				"key with escaped double quotes\"": "string value with escaped double quotes\"",
				"key with \" escaped double quotes\"": "string value \" with escaped double quotes",
				"key with \"\" \"more escaped double quotes\"": "string value \"\" \"more with escaped double quotes",
				"key with escaped back slash\\": "string value with escaped back slash\\",
				"key with 2 escaped back slashes\\\\": "string value with 2 escaped back slashes\\\\",
				"key with 3 escaped back slashes\\\\\\": "string value with 3 escaped back slashes\\\\\\",
				"key with 6 escaped back slashes\\\\\\\\\\\\": "string value with 6 escaped back slashes\\\\\\\\\\\\",
				otherValue: null
			}`
var aNumber = `10`
var smallObject = `{"aaa": `+aNumber+`}`
var arrayDeeplyNested = `[1232340, true, "abcdefghij", ["ds", `+smallObject+`, 10, false, null], "abcdefghijaaaa", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl", "abcdefgh", "abcdefghij", "abcdefghij", "abcdefghijkl"]`
var service = `{
				"attributes": {
					"full name": "string",
					"short name": "string",
					"initials": "string",
					"price": {
						"type": "float",
						"mask": "price",
						"embed": true
					},
					"commission": {
						"type": "float",
						"mask": "percentage",
						"embed": true,
						"number": -123456789.123,
						"nullValue": null,
						"aTrueValue": true,
						"aFalseValue": false,
						"array": `+arrayDeeplyNested+`,
						"test": "test"
					}
				}
			}`
var bigObject = `{
			"client": `+client+`,
			"service": `+service+`
		}`
var arrayTopLevel = `[1, "asas", {a: 10, b: "hello"}, null, [], {}]`
var jsonBytes = []byte(`{`+
		`"bigObject": `+bigObject+`,`+
		`"anArray": `+arrayTopLevel+`,
		"emptyArray": [],
		"emptyObject": {},
		"bcryptCost": 10,
		"abc": "aaaaaac",
		"aNUllValue": null,
		"aTrueValue": true,
		"aFalseValue": false
	}`)