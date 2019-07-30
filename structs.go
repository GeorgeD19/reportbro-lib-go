package reportbro

import (
	"strings"

	colorful "github.com/lucasb-eyer/go-colorful"
)

// Color
type Color struct {
	ColorCode   string `json:"color_code"`
	R           int    `json:"r"`
	G           int    `json:"g"`
	B           int    `json:"b"`
	Transparent bool   `json:"transparent"`
}

func (self *Color) init(color string) {
	self.ColorCode = ""
	if len(color) == 7 && string(color[0]) == "#" {
		c, err := colorful.Hex(color)
		if err == nil {
			r, g, b := c.RGB255()
			self.R = int(r)
			self.G = int(g)
			self.B = int(b)
		}
		self.Transparent = false
		self.ColorCode = color
	} else {
		self.Transparent = true
	}
}

// IsBlack returns true or false on whether or not the color is black and not transparent
func IsBlack(self *Color) bool {
	return self.R == 0 && self.G == 0 && self.B == 0 && self.Transparent == false
}

// NewColor creates a new Color instance with defaults
func NewColor(color string) Color {
	colorType := Color{}
	colorType.init(color)
	return colorType
}

type Parameter struct {
	report             *report
	ID                 int
	Name               string
	Type               ParameterType
	ArrayItemType      ParameterType
	Eval               bool
	Nullable           bool
	Expression         string
	Pattern            string
	PatternHasCurrency bool
	IsInternal         bool
	Children           []interface{}
	Fields             map[string]interface{}
}

func (self *Parameter) init(report *report, data map[string]interface{}) {
	self.report = report
	self.ID = GetIntValue(data, "id")
	self.Name = GetStringValue(data, "name")
	if self.Name == "" {
		self.Name = "<unnamed>"
	}
	self.Type = getParameterType(GetStringValue(data, "type"))
	if self.Type == ParameterTypeSimpleArray {
		self.ArrayItemType = getParameterType(GetStringValue(data, "arrayItemType"))
	} else {
		self.ArrayItemType = ParameterTypeNone
	}
	self.Eval = GetBoolValue(data, "eval")
	self.Nullable = GetBoolValue(data, "nullable")
	self.Expression = GetStringValue(data, "expression")
	self.Pattern = GetStringValue(data, "pattern")
	self.PatternHasCurrency = strings.Contains(self.Pattern, "$")
	self.IsInternal = !notIn(self.Name, []string{"page_count", "page_number"})
	self.Children = make([]interface{}, 0)
	self.Fields = make(map[string]interface{}, 0)
	if self.Type == ParameterTypeArray || self.Type == ParameterTypeMap {
		for _, item := range data["children"].([]interface{}) {
			parameter := NewParameter(report, item.(map[string]interface{}))
			if _, ok := self.Fields[parameter.Name]; ok {
				self.report.errors = append(self.report.errors, Error{Message: "errorMsgDuplicateparameterField", ObjectID: parameter.ID, Field: "name"})
			} else {
				self.Children = append(self.Children, parameter)
				self.Fields[parameter.Name] = parameter
			}
		}
	}
}

func NewParameter(report *report, data map[string]interface{}) Parameter {
	parameter := Parameter{}
	parameter.init(report, data)
	return parameter
}

type Style interface {
	init(data map[string]interface{}, keyPrefix string)
	base() *textStyle
	border() *BorderStyle
}

type BorderStyle struct {
	BorderColor  Color
	BorderWidth  float64
	BorderAll    bool
	BorderLeft   bool
	BorderTop    bool
	BorderRight  bool
	BorderBottom bool
}

func (self *BorderStyle) base() *textStyle {
	if self != nil {
		return &textStyle{BorderStyle: *self}
	}
	return &textStyle{}
}

func (self *BorderStyle) border() *BorderStyle {
	return self
}

func (self *BorderStyle) init(data map[string]interface{}, keyPrefix string) {
	self.BorderColor = NewColor(GetStringValue(data, keyPrefix+"borderColor"))
	self.BorderWidth = float64(GetIntValue(data, keyPrefix+"borderWidth"))
	self.BorderAll = GetBoolValue(data, keyPrefix+"borderAll")
	self.BorderLeft = GetBoolValue(data, keyPrefix+"borderLeft")
	if self.BorderLeft || self.BorderAll {
		self.BorderLeft = true
	}
	self.BorderTop = GetBoolValue(data, keyPrefix+"borderTop")
	if self.BorderTop || self.BorderAll {
		self.BorderTop = true
	}
	self.BorderRight = GetBoolValue(data, keyPrefix+"borderRight")
	if self.BorderRight || self.BorderAll {
		self.BorderRight = true
	}
	self.BorderBottom = GetBoolValue(data, keyPrefix+"borderBottom")
	if self.BorderBottom || self.BorderAll {
		self.BorderBottom = true
	}
}

func NewBorderStyle(data map[string]interface{}, keyPrefix string) BorderStyle {
	borderStyle := BorderStyle{}
	borderStyle.init(data, keyPrefix)
	return borderStyle
}

type textStyle struct {
	BorderStyle
	Bold                bool
	Italic              bool
	Underline           bool
	Strikethrough       bool
	TextColor           Color
	BackgroundColor     Color
	Font                string
	FontSize            float64
	LineSpacing         float64
	FontStyle           string
	TextAlign           string
	HorizontalAlignment HorizontalAlignment
	VerticalAlignment   VerticalAlignment
	PaddingLeft         float64
	PaddingTop          float64
	PaddingRight        float64
	PaddingBottom       float64
}

func (self *textStyle) base() *textStyle {
	return self
}

func (self *textStyle) init(data map[string]interface{}, keyPrefix string) {
	self.BorderStyle.init(data, keyPrefix)
	self.Bold = GetBoolValue(data, keyPrefix+"bold")
	self.Italic = GetBoolValue(data, keyPrefix+"italic")
	self.Underline = GetBoolValue(data, keyPrefix+"underline")
	self.Strikethrough = GetBoolValue(data, keyPrefix+"strikethrough")
	self.HorizontalAlignment = GetHorizontalAlignment(GetStringValue(data, keyPrefix+"horizontalAlignment"))
	self.VerticalAlignment = GetVerticalAlignment(GetStringValue(data, keyPrefix+"verticalAlignment"))
	self.TextColor = NewColor(GetStringValue(data, keyPrefix+"textColor"))
	self.BackgroundColor = NewColor(GetStringValue(data, keyPrefix+"backgroundColor"))
	// self.Font = GetStringValue(data, keyPrefix+"font")
	if self.Font == "" {
		self.Font = "Helvetica"
	}
	self.FontSize = float64(GetIntValue(data, keyPrefix+"fontSize"))
	if self.FontSize == 0.0 {
		self.FontSize = 12.0
	}
	self.LineSpacing = GetFloatValue(data, keyPrefix+"lineSpacing")
	self.PaddingLeft = float64(GetIntValue(data, keyPrefix+"paddingLeft"))
	self.PaddingTop = float64(GetIntValue(data, keyPrefix+"paddingTop"))
	self.PaddingRight = float64(GetIntValue(data, keyPrefix+"paddingRight"))
	self.PaddingBottom = float64(GetIntValue(data, keyPrefix+"paddingBottom"))
	self.FontStyle = ""
	if self.Bold {
		self.FontStyle += "B"
	}
	if self.Italic {
		self.FontStyle += "I"
	}
	self.TextAlign = ""
	if self.HorizontalAlignment == HorizontalAlignmentLeft {
		self.TextAlign = "L"
	} else if self.HorizontalAlignment == HorizontalAlignmentCenter {
		self.TextAlign = "C"
	} else if self.HorizontalAlignment == HorizontalAlignmentRight {
		self.TextAlign = "R"
	} else if self.HorizontalAlignment == HorizontalAlignmentJustify {
		self.TextAlign = "J"
	}
	self.addBorderPadding()
}

func (self *textStyle) getFontStyle(ignoreUnderline bool) string {
	fontStyle := ""
	if self.Bold {
		fontStyle += "B"
	}
	if self.Italic {
		fontStyle += "I"
	}
	if self.Underline && ignoreUnderline == false {
		fontStyle += "U"
	}
	return fontStyle
}

func (self *textStyle) addBorderPadding() {
	if self.BorderLeft {
		self.PaddingLeft += self.BorderWidth
	}
	if self.BorderTop {
		self.PaddingTop += self.BorderWidth
	}
	if self.BorderRight {
		self.PaddingRight += self.BorderWidth
	}
	if self.BorderBottom {
		self.PaddingBottom += self.BorderWidth
	}
}

func NewTextStyle(data map[string]interface{}, keyPrefix string) textStyle {
	textStyle := textStyle{}
	textStyle.init(data, keyPrefix)
	return textStyle
}
