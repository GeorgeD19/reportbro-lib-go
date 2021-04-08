package reportbro

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
)

type DataType int

const (
	DataTypeNone             DataType = 0
	DataTypeInt              DataType = 1
	DataTypeString           DataType = 2
	DataTypeFloat            DataType = 3
	DataTypeDatetime         DataType = 4
	DataTypeTableTextElement DataType = 5
)

var DataTypes = [...]string{
	"none",
	"int",
	"string",
	"float",
	"datetime",
	"tabletextelement",
}

func (DataType DataType) String() string {
	return DataTypes[DataType]
}

func getDataType(data interface{}) DataType {
	switch data.(type) {
	case int:
		return DataTypeInt
	case float64:
		return DataTypeFloat
	case time.Time:
		return DataTypeDatetime
	case string:
		return DataTypeString
	case TableTextElement:
		return DataTypeTableTextElement
	default:
		return DataTypeNone
	}
}

func GetArrayChildren(data string, key string) []string {
	results := make([]string, 0)
	jsonparser.ArrayEach([]byte(data), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		results = append(results, string(value))
	}, key)
	return results
}

func GetObjectChildren(data string, key string) []string {
	results := make([]string, 0)
	jsonparser.ObjectEach([]byte(data), func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		results = append(results, string(value))
		return nil
	}, key)
	return results
}

func Get(data string, key string) string {
	value, _, _, err := jsonparser.Get([]byte(data), key)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return string(value)
}

func GetValue(data map[string]interface{}, key string) interface{} {
	if value, ok := data[key]; ok {
		return value
	}
	return nil
}

func GetStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key]; ok {
		if typeValue, typeOk := value.(string); typeOk {
			return typeValue
		}
	}
	return ""
}

// GetIntValue retrieves int value if exists, if not returns 0
func GetIntValue(data map[string]interface{}, key string) int {
	if value, ok := data[key]; ok {
		if typeValue, typeOk := value.(float64); typeOk {
			return int(typeValue)
		}
		if typeValue, typeOk := value.(string); typeOk {
			parseValue, err := strconv.Atoi(typeValue)
			if err == nil {
				return parseValue
			}
		}
	}
	return 0
}

func GetFloatValue(data map[string]interface{}, key string) float64 {
	if value, ok := data[key]; ok {
		if typeValue, typeOk := value.(float64); typeOk {
			return typeValue
		}
		if typeValue, typeOk := value.(string); typeOk {
			parseValue, err := strconv.ParseFloat(typeValue, 64)
			if err == nil {
				return parseValue
			}
		}
	}
	return 0.0
}

func GetBoolValue(data map[string]interface{}, key string) bool {
	if value, ok := data[key]; ok {
		if typeValue, typeOk := value.(bool); typeOk {
			return typeValue
		}
	}
	return false
}

func Round(x float64) float64 {
	return math.Ceil(x*100) / 100
}

// Pythons equivalent to if not in for strings
func notIn(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return false
		}
	}
	return true
}

func notInParameter(a string, list []Parameter) bool {
	for _, b := range list {
		if b.Name == a {
			return false
		}
	}
	return true
}

func notInElement(a int, list map[int]bool) bool {
	result := true
	for i, b := range list {
		if a == i && b {
			result = false
		}
	}
	return result
}

func inArray(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func StringPointer(i string) *string {
	return &i
}

func BoolPointer(i bool) *bool {
	return &i
}

func ColorPointer(i Color) *Color {
	return &i
}

func ReportBandPointer(i reportBand) *reportBand {
	return &i
}

// Max is equivalent to Python max()
func max(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxFloatSlice(v []float64) float64 {
	sort.Float64s(v)
	return v[len(v)-1]
}

// Remove element from array
func remove(slice []int, s int) []int {
	return append(slice[:s], slice[s+1:]...)
}

func removeElement(slice []DocElementBaseProvider, index int) []DocElementBaseProvider {
	return append(slice[:index], slice[index+1:]...)
}

func removeElementWithID(slice []DocElementBaseProvider, ID int) []DocElementBaseProvider {
	for i, element := range slice {
		if element.base().ID == ID {
			slice = removeElement(slice, i)
		}
	}
	return slice
}

// pyRange returns an int array from start to stop moving up in steps
func pyRange(start int, stop int, step int) []int {
	results := make([]int, 0)
	for x := start; x != stop; x = x + step {
		results = append(results, x)
	}
	return results
}

// pyMin Return the lowest number
func pyMin(a float64, b float64) float64 {
	if a > b {
		return b
	}
	return a
}

// pyStrip returns a copy of the string in which all default whitespace chars have been stripped from the beginning and the end of the string
func pyStrip(s string, chars string) string {
	if chars == "" {
		return strings.TrimSpace(s)
	}
	return strings.Trim(s, chars)
}

// pyLstrip returns a copy of the string in which all default whitespace chars have been stripped from the beginning of the string
func pyLstrip(s string, chars string) string {
	return strings.TrimLeft(s, chars)
}

// pyLstrip returns a copy of the string in which all default whitespace chars have been stripped from the end of the string
func pyRstrip(s string, chars string) string {
	return strings.TrimRight(s, chars)
}

// It determines if string str occurs in string, or in a substring of string if starting index beg and ending index end are given.
func pyFind(s string, char string, beg int, end int) int {
	if end == 0 {
		end = len(s)
	}
	s = s[beg:end]
	p := strings.Index(s, char)
	if p > -1 {
		return strings.Index(s, char) + beg
	} else {
		return -1
	}
}

func instanceOf(objectPtr, typePtr interface{}) bool {
	return reflect.TypeOf(objectPtr) == reflect.TypeOf(typePtr)
}

var watermark = "data:image/png;base64,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"
var dateFormat = ""

func formatLocale() {

}

func Merge(primary map[string]interface{}, secondary map[string]interface{}) map[string]interface{} {
	for k, v := range secondary {
		primary[k] = v
	}
	return primary
}

func Substring(str, start, end string) ([]byte, error) {
	var match []byte
	index := strings.Index(str, start)

	if index == -1 {
		return match, errors.New("Not found")
	}

	index += len(start)

	for {
		char := str[index]

		if strings.HasPrefix(str[index:index+len(match)], end) {
			break
		}

		match = append(match, char)
		index++
	}

	return match, nil
}

func parseExpression(expr string) (string, error) {
	if strings.Contains(expr, "if") {
		tVal, err := Substring(expr, ``, ` if `)
		trueValue := ""
		if err == nil {
			trueValue = string(tVal)
			trueValue = strings.Replace(trueValue, ` `, ``, -1)
			quoted := false
			if strings.Index(trueValue, `'`) != -1 {
				quoted = true
				trueValue = strings.Replace(trueValue, `'`, ``, -1)
				expr = strings.Replace(expr, `'`+trueValue+`'`, ``, -1)
			} else {
				expr = strings.Replace(expr, trueValue, ``, -1)
			}

			expr = strings.Replace(expr, `'`, `"`, -1)
			expr = strings.Replace(expr, ` if `, ``, -1)
			if quoted {
				expr = strings.Replace(expr, `else`, `? "`+trueValue+`" :`, -1)
			} else {
				expr = strings.Replace(expr, `else`, `? `+trueValue+` :`, -1)
			}
		} else {
			return "", fmt.Errorf("Not an expression")
		}
	}
	return expr, nil
}

// var locale map[string]func(string) string

// func init() {
// 	locale = make(map[string]func(string) string)
// 	locale["DE-DE"] = normalizeGerman
// 	locale["US"] = normalizeAmerican
// }

// var locales = [
// 	"ar": "ar_SY",
// 	"bg": "bg_BG",
// 	"bs": "bs_BA",
// 	"ca": "ca_ES",
// 	"cs": "cs_CZ",
// 	"da": "da_DK",
// 	"de": "de_DE",
// 	"el": "el_GR",
// 	"en": "en_US",
// 	"es": "es_ES",
// 	"et": "et_EE",
// 	"fa": "fa_IR",
// 	"fi": "fi_FI",
// 	"fr": "fr_FR",
// 	"gl": "gl_ES",
// 	"he": "he_IL",
// 	"hu": "hu_HU",
// 	"id": "id_ID",
// 	"is": "is_IS",
// 	"it": "it_IT",
// 	"ja": "ja_JP",
// 	"km": "km_KH",
// 	"ko": "ko_KR",
// 	"lt": "lt_LT",
// 	"lv": "lv_LV",
// 	"mk": "mk_MK",
// 	"nl": "nl_NL",
// 	"nn": "nn_NO",
// 	"no": "nb_NO",
// 	"pl": "pl_PL",
// 	"pt": "pt_PT",
// 	"ro": "ro_RO",
// 	"ru": "ru_RU",
// 	"sk": "sk_SK",
// 	"sl": "sl_SI",
// 	"sv": "sv_SE",
// 	"th": "th_TH",
// 	"tr": "tr_TR",
// 	"uk": "uk_UA"
// ]
