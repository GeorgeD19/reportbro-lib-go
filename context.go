package reportbro

import (
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/PaesslerAG/gval"
	"github.com/dustin/go-humanize"
	"github.com/jinzhu/now"
	"github.com/spf13/cast"
	"github.com/vjeantet/jodaTime"
)

var EVAL_DEFAULT_NAMES = map[string]interface{}{
	"True":  true,
	"False": false,
	"None":  nil,
}

type Context struct {
	parameters            map[string]interface{}
	Data                  map[string]interface{}
	DataStr               string // For debugging
	Report                *report
	PatternLocale         string
	PatternCurrencySymbol string
	RootData              map[string]interface{}
}

func (self *Context) init(report report, parameters map[string]interface{}, data map[string]interface{}) {
	self.Report = &report
	self.PatternLocale = report.documentProperties.PatternLocale
	self.PatternCurrencySymbol = report.documentProperties.PatternCurrencySymbol
	self.parameters = parameters
	self.Data = data
	self.Data = Merge(self.Data, EVAL_DEFAULT_NAMES)
	self.RootData = self.Data
	self.RootData["page_number"] = 0
	self.RootData["page_count"] = 0
}

func (self *Context) getParameter(name string, parameters map[string]interface{}) interface{} {
	result := make(map[string]interface{}, 0)
	if !(len(parameters) > 0) {
		parameters = self.parameters
	}
	if val, ok := parameters[name]; ok {
		result[name] = val
		return result
	} else if _, ok := parameters["__parent"]; ok {
		return self.getParameter(name, parameters["__parent"].(map[string]interface{}))
	}
	return nil
}

func (self *Context) getData(name string, data map[string]interface{}) (interface{}, bool) {
	if data == nil {
		data = self.Data
	}
	if value, ok := data[name]; ok {
		return value, true
	} else if value, ok := data["__parent"]; ok {
		return self.getData(name, value.(map[string]interface{}))
	}
	return nil, false
}

func (self *Context) pushContext(parameters map[string]interface{}, data map[string]interface{}) {
	parameters["__parent"] = self.parameters
	self.parameters = parameters
	data["__parent"] = self.Data
	self.Data = data
}

func (self *Context) popContext() {
	parameters := self.parameters["__parent"]
	if parameters == nil {
		log.Println(Error{Message: "Context.pop_context failed - no parent available"})
	}
	delete(self.parameters, "__parent")
	if params, ok := parameters.(map[string]interface{}); ok {
		self.parameters = params
	}
	data := self.Data["__parent"]
	if data == nil {
		log.Println(Error{Message: "Context.pop_context failed - no parent available"})
	}
	delete(self.parameters, "__parent")
	self.Data = data.(map[string]interface{})
}

func (self *Context) fillParameters(expr string, objectID int, field string, pattern string) string {
	if !strings.Contains(expr, "${") {
		return expr
	}
	ret := ""
	prevC := ""
	parameterIndex := -1
	for i, char := range expr {
		c := string(char)
		if parameterIndex == -1 {
			if prevC == "$" && c == "{" {
				parameterIndex = i + 1
				if last := len(ret) - 1; last >= 0 {
					ret = ret[:last]
				}
			} else {
				ret += c
			}
		} else {
			if c == "}" {
				parameterName := expr[parameterIndex:i]
				var parameter interface{}
				var collectionName string
				var fieldName string
				if strings.Contains(parameterName, ".") {
					nameParts := strings.Split(parameterName, ".")
					collectionName = nameParts[0]
					fieldName = nameParts[1]
					parameter = self.getParameter(collectionName, nil).(map[string]interface{})[collectionName]
					if parameter == nil {
						log.Println(Error{Message: "errorMsgInvalidExpressionNameNotDefined", ObjectID: objectID, Field: field, Info: collectionName})
					}
				} else {
					parameter = self.getParameter(parameterName, nil).(map[string]interface{})[parameterName]
					if parameter == nil {
						log.Println(Error{Message: "errorMsgInvalidExpressionNameNotDefined", ObjectID: objectID, Field: field, Info: collectionName})
					}
				}

				var value interface{}
				if parameter != nil && parameter.(Parameter).Type == ParameterTypeMap {
					parameter = self.getParameter(fieldName, parameter.(Parameter).Fields).(map[string]interface{})[fieldName]
					if parameter == nil {
						log.Println(Error{Message: "errorMsgInvalidExpressionNameNotDefined", ObjectID: objectID, Field: field, Info: parameterName})
					}
					mapValue, _ := self.getData(collectionName, self.Data)
					if mapValue != nil && parameter != nil {
						value = GetValue(mapValue.(map[string]interface{}), fieldName)
					}
				} else {
					parameterExists := false
					value, parameterExists = self.getData(parameterName, self.Data)
					if !parameterExists {
						log.Println(Error{Message: "errorMsgMissingParameterData", ObjectID: objectID, Field: field, Info: parameterName})
					}
				}

				if value != nil {
					ret += self.getFormattedValue(value, parameter.(Parameter), objectID, pattern, false)
				}
				parameterIndex = -1
			}
		}
		prevC = c
	}
	return ret
}

// https://pypi.org/project/simpleeval/ basically evaluates mathematical expressions, go will use govaluate
func (self *Context) evaluateExpression(expr interface{}, objectID int, field string) interface{} {
	if expr != nil {
		data := map[string]interface{}{
			"True":  true,
			"False": false,
			"None":  nil,
		}
		expr = self.replaceParameters(expr.(string), &data)
		parsedExpr, err := parseExpression(cast.ToString(expr))
		if err == nil {
			expr = parsedExpr
		}
		for index, value := range data {
			if strValue, ok := value.(string); ok {
				strValue = strings.Replace(strValue, ",", "", -1)
				if floatValue, err := strconv.ParseFloat(strValue, 64); err == nil {
					data[index] = floatValue
				}
				// if strValue == "" {
				// 	data[index] = 0
				// }
			}
			// if value == nil {
			// 	data[index] = ""
			// }
		}

		expression, err := govaluate.NewEvaluableExpression(cast.ToString(expr))
		if err == nil {
			result, err := expression.Evaluate(data)
			if err == nil {
				return result
			} else {
				value, err := gval.Evaluate(cast.ToString(expr), data)
				if err == nil {
					return value
				}
			}
		} else {
			value, err := gval.Evaluate(cast.ToString(expr), data)
			if err == nil {
				return value
			}
		}

		return expr
	}
	return true
}

// stripParameterName @static
func stripParameterName(expr string) string {
	if expr != "" {
		expr = pyStrip(expr, "")
		expr = pyLstrip(expr, "${")
		expr = pyRstrip(expr, "}")
		return expr
	}
	return expr
}

// isParameterName @static
func isParameterName(expr string) bool {
	if expr != "" {
		stripExpr := pyStrip(expr, "")
		if strings.HasPrefix(stripExpr, "${") && strings.HasSuffix(stripExpr, "}") {
			return true
		}
	}
	return false
}

func (self *Context) getFormattedValue(value interface{}, parameter Parameter, objectID int, pattern string, isArrayItem bool) string {
	var rv interface{}
	var valueType ParameterType
	if isArrayItem && parameter.Type == ParameterTypeSimpleArray {
		valueType = parameter.ArrayItemType
	} else {
		valueType = parameter.Type
	}
	if valueType == ParameterTypeString {
		rv = value.(string)
	} else if valueType == ParameterTypeNumber || valueType == ParameterTypeAverage || valueType == ParameterTypeSum {
		var usedPattern string
		var patternHasCurrency bool
		if pattern != "" {
			usedPattern = pattern
			patternHasCurrency = strings.Contains(pattern, "$")
		} else {
			usedPattern = parameter.Pattern
			patternHasCurrency = parameter.PatternHasCurrency
		}

		if usedPattern != "" {
			usedPattern = strings.Replace(usedPattern, " ", "", -1)
			usedPattern = strings.Replace(usedPattern, "0", "#", -1)

			if strings.Count(cast.ToString(usedPattern), ".") < 1 {
				usedPattern += "."
			}

			runes := cast.ToString(value)
			out := []rune(runes)
			caught := false
			for i := len(runes); i > 0; i-- {
				if "," == string(runes[i-1]) {
					if !caught {
						caught = true
						out[i-1] = rune('.')
					}
				}
			}
			floatVal := cast.ToFloat64(strings.Replace(string(out), ",", "", -1))

			if patternHasCurrency {
				usedPattern = strings.Replace(usedPattern, "$", "", -1)

				value = humanize.FormatFloat(usedPattern, floatVal)
				value = self.PatternCurrencySymbol + cast.ToString(value)
			} else {
				if usedPattern != "" {
					value = humanize.FormatFloat(usedPattern, floatVal)
				}
			}
			rv = value
		} else {
			usedPattern += "####."
			runes := cast.ToString(value)
			out := []rune(runes)
			caught := false
			for i := len(runes); i > 0; i-- {
				if "," == string(runes[i-1]) {
					if !caught {
						caught = true
						out[i-1] = rune('.')
					}
				}
			}
			floatVal := cast.ToFloat64(strings.Replace(string(out), ",", "", -1))

			if floatVal != float64(int64(floatVal)) {
				usedPattern += "##"
			}

			value = humanize.FormatFloat(usedPattern, floatVal)

			rv = cast.ToString(value)
		}
	} else if valueType == ParameterTypeDate {
		usedPattern := parameter.Pattern
		if pattern != "" {
			usedPattern = pattern
		}
		rv = value
		if usedPattern != "" {
			t, err := now.Parse(cast.ToString(value))
			if err == nil {
				date := jodaTime.Format(usedPattern, t)
				value = date
			}
			rv = value
		}
	} else {
		rv = value
	}

	return cast.ToString(rv)
}

func (self *Context) replaceParameters(expr string, data *map[string]interface{}) interface{} {
	var value interface{}
	pos := pyFind(expr, "${", 0, 0)
	if pos == -1 {
		return expr
	}
	ret := ""
	pos2 := 0
	for pos != -1 {
		if pos != 0 {
			ret += expr[pos2:pos]
		}
		pos2 = pyFind(expr, "}", pos, 0)
		if pos2 != -1 {
			parameterName := expr[pos+2 : pos2]
			if data != nil {
				if pyFind(parameterName, ".", 0, 0) != -1 {
					nameParts := strings.Split(parameterName, ".")
					collectionName := nameParts[0]
					fieldName := nameParts[1]
					value, _ = self.getData(collectionName, nil)
					if reflect.TypeOf(value) == reflect.TypeOf(map[string]interface{}{}) {
						value = value.(map[string]interface{})[fieldName]
					} else {
						value = nil
					}
					// use valid python identifier for parameter name
					parameterName = collectionName + "_" + fieldName
				} else {
					value, _ = self.getData(parameterName, nil)
				}
				if numVal, err := strconv.ParseFloat(cast.ToString(value), 64); err == nil {
					value = numVal
				}

				(*data)[parameterName] = value
			}
			ret += parameterName
			pos2++
			pos = pyFind(expr, "${", pos2, 0)
		} else {
			pos2 = pos
			pos = -1
		}
	}
	ret += expr[pos2:]
	return ret
}

func (self *Context) incPageNumber() {
	self.RootData["page_number"] = self.RootData["page_number"].(int) + 1
}

func (self *Context) getPageNumber() int {
	return self.RootData["page_number"].(int)
}

func (self *Context) setPageCount(pageCount int) {
	self.RootData["page_count"] = pageCount
}

func NewContext(report report, parameters map[string]interface{}, data map[string]interface{}) Context {
	context := Context{}
	context.init(report, parameters, data)
	return context
}
