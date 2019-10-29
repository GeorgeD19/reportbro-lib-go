//
// Copyright (C) 2019 jobsta
//
// This file is part of ReportBro, a library to generate PDF and Excel reports.
// Demos can be found at https://www.reportbro.com
//
// Dual licensed under AGPLv3 and ReportBro commercial license:
// https://www.reportbro.com/license
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see https://www.gnu.org/licenses/
//
// Details for ReportBro commercial license can be found at
// https://www.reportbro.com/license/agreement
//

package reportbro

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/jung-kurt/gofpdf"
	"github.com/spf13/cast"
	"github.com/vincent-petithory/dataurl"
)

var regexValidIdentifier, _ = regexp.Compile(`^[^\d\W]\w*$`)

type Renderer interface {
	render()
	insertImage(row int, col int, imageFilename string, width float64)
	write(row int, col int, colspan int, text string, cellFormat string, width float64)
}

type documentPDFRenderer struct {
	headerBand         containerProvider
	contentBand        containerProvider
	footerBand         containerProvider
	documentProperties documentProperties
	pdfDoc             FPDFRB
	context            Context
	addWatermark       bool
}

func (self *documentPDFRenderer) init(headerBand containerProvider, contentBand containerProvider, footerBand containerProvider, report *report, context Context, additionalFonts string, addWatermark bool) {
	self.headerBand = headerBand
	self.contentBand = contentBand
	self.footerBand = footerBand
	self.documentProperties = report.documentProperties
	self.pdfDoc = newFPDFRB(report.documentProperties, additionalFonts)
	self.pdfDoc.Fpdf.SetMargins(0.0, 0.0, 0.0)
	self.pdfDoc.CMargin = 0 // interior cell margin
	self.context = context
	self.addWatermark = addWatermark
}

func (self *documentPDFRenderer) addPage() {
	self.pdfDoc.Fpdf.AddPage()
	self.context.incPageNumber()
}

func (self *documentPDFRenderer) isFinished() bool {
	return self.contentBand.base().isFinished()
}

func (self *documentPDFRenderer) render() ([]byte, error) {
	watermarkHeight := 0.0
	watermarkWidth := watermarkHeight
	watermarkFilename := "reportbro"
	if self.addWatermark {
		watermarkWidth = self.documentProperties.pageWidth / 3
		watermarkHeight = watermarkWidth * (108.0 / 461.0)
	}

	self.contentBand.prepare(self.context, &self.pdfDoc, false)
	pageCount := 1
	for true {
		height := self.documentProperties.pageHeight - self.documentProperties.marginTop - self.documentProperties.marginBottom
		if self.documentProperties.headerDisplay == BandDisplayAlways || (self.documentProperties.headerDisplay == BandDisplayNotOnFirstPage && pageCount != 1) {
			height -= self.documentProperties.headerSize
		}
		if self.documentProperties.footerDisplay == BandDisplayAlways || (self.documentProperties.footerDisplay == BandDisplayNotOnFirstPage && pageCount != 1) {
			height -= self.documentProperties.footerSize
		}
		complete := self.contentBand.createRenderElements(height, self.context, &self.pdfDoc)
		if complete {
			break
		}
		pageCount++
		if pageCount >= 10000 {
			return nil, fmt.Errorf("Too many pages (probably an endless loop)")
		}
	}
	self.context.setPageCount(pageCount)

	footerOffsetY := self.documentProperties.pageHeight - self.documentProperties.footerSize - self.documentProperties.marginBottom
	// render at least one page to show header/footer even if content is empty
	for self.contentBand.isFinished() == false || self.context.getPageNumber() == 0 {
		self.addPage()
		if self.addWatermark {
			if watermarkHeight < self.documentProperties.pageHeight {
				dataURL, err := dataurl.DecodeString(watermark)
				if err != nil {
					fmt.Println(err)
				} else {
					options := gofpdf.ImageOptions{
						ReadDpi:               false,
						AllowNegativePosition: true,
					}
					switch dataURL.MediaType.ContentType() {
					case "image/png":
						options.ImageType = "png"
						break
					case "image/jpg":
						options.ImageType = "jpg"
						break
					}
					x := (self.documentProperties.pageWidth / 2) - (watermarkWidth / 2)
					y := (self.documentProperties.pageHeight) - watermarkHeight
					self.pdfDoc.Fpdf.RegisterImageOptionsReader(watermarkFilename, options, bytes.NewReader(dataURL.Data))
					self.pdfDoc.Fpdf.ImageOptions(watermarkFilename, x, y, watermarkWidth, watermarkHeight, false, options, 0, "")
				}
			}
		}

		contentOffsetY := self.documentProperties.marginTop
		pageNumber := self.context.getPageNumber()
		if self.documentProperties.headerDisplay == BandDisplayAlways || (self.documentProperties.headerDisplay == BandDisplayNotOnFirstPage && pageNumber != 1) {
			contentOffsetY += self.documentProperties.headerSize
			self.headerBand.prepare(self.context, &self.pdfDoc, false)
			self.headerBand.createRenderElements(self.documentProperties.headerSize, self.context, &self.pdfDoc)
			self.headerBand.renderPDF(self.documentProperties.marginLeft, self.documentProperties.marginTop, &self.pdfDoc, false)
		}
		if self.documentProperties.footerDisplay == BandDisplayAlways || (self.documentProperties.footerDisplay == BandDisplayNotOnFirstPage && pageNumber != 1) {
			self.footerBand.prepare(self.context, &self.pdfDoc, false)
			self.footerBand.createRenderElements(self.documentProperties.footerSize, self.context, &self.pdfDoc)
			self.footerBand.renderPDF(self.documentProperties.marginLeft, footerOffsetY, &self.pdfDoc, false)
		}

		self.contentBand.renderPDF(self.documentProperties.marginLeft, contentOffsetY, &self.pdfDoc, true)
	}

	self.headerBand.cleanup()
	self.footerBand.cleanup()

	writer := &bytes.Buffer{}
	err := self.pdfDoc.Fpdf.Output(writer)
	return writer.Bytes(), err
}

func newDocumentPDFRenderer(headerBand containerProvider, contentBand containerProvider, footerBand containerProvider, report *report, context Context, additionalFonts string, addWatermark bool) documentPDFRenderer {
	documentPDFRenderer := documentPDFRenderer{}
	documentPDFRenderer.init(headerBand, contentBand, footerBand, report, context, additionalFonts, addWatermark)
	return documentPDFRenderer
}

func (self *report) GeneratePDF(addWatermark bool) ([]byte, error) {
	renderer := newDocumentPDFRenderer(self.header, self.content, self.footer, self, self.context, self.additionalFonts, addWatermark)
	return renderer.render()
}

type DocumentXLSXRenderer struct {
	headerBand         containerProvider
	contentBand        containerProvider
	footerBand         containerProvider
	context            Context
	row                int
	columnWidths       []float64
	additionalFonts    string
	filename           string
	addWatermark       bool
	documentProperties documentProperties
	pdfDoc             FPDFRB
}

func (self *DocumentXLSXRenderer) init(headerBand containerProvider, contentBand containerProvider, footerBand containerProvider, report *report, context Context, filename string) {
	self.headerBand = headerBand
	self.contentBand = contentBand
	self.footerBand = footerBand
	self.documentProperties = report.documentProperties
	// self.workbook_mem = BytesIO()
	if filename != "" {
		// self.workbook = xlsxwriter.Workbook(filename if filename else self.workbook_mem)
	}
	// self.Worksheet = self.workbook.add_worksheet()
	self.context = context
	self.filename = filename
	self.row = 0
	self.columnWidths = make([]float64, 0)
}

func (self *DocumentXLSXRenderer) render() {
	if self.documentProperties.headerDisplay != BandDisplayNever {
		self.renderBand(&self.headerBand)
	}
	self.renderBand(&self.contentBand)
	if self.documentProperties.headerDisplay != BandDisplayNever {
		self.renderBand(&self.footerBand)
	}

	// for i, columnWidth := range self.columnWidths {
	// 	if columnWidth > 0 {
	// 		// setting the column width is just an approximation, in Excel the width
	// 		// is the number of characters in the default font
	// 		// self.worksheet.setColumn(i, i, columnWidth/7)
	// 	}
	// }
	// self.workbook.close()
	if self.filename == "" {
		// if no filename is given the spreadsheet data will be returned
		// self.workbook_mem.seek(0)
		// return self.workbook_mem.read()
	}
	return
}

func (self *DocumentXLSXRenderer) renderBand(band *containerProvider) {
	// band.prepare(self.context)
	// self.row, _ = band.render_spreadsheet(self.row, 0, self.context, self)
}

func (self *DocumentXLSXRenderer) update_columnWidth(col int, width float64) {
	// if col >= len(self.columnWidths):
	// 	// make sure columnWidth list contains entries for each column
	// 	self.columnWidths.extend([-1] * (col + 1 - len(self.columnWidths)))
	// if width > self.columnWidths[col]:
	// 	self.columnWidths[col] = width
}

func (self *DocumentXLSXRenderer) write(row int, col int, colspan int, text string, cellFormat string, width float64) {
	// if colspan > 1 {
	// 	self.worksheet.mergeRange(row, col, row, col+colspan-1, text, cellFormat)
	// } else {
	// 	self.worksheet.write(row, col, text, cellFormat)
	// 	self.updateColumnWidth(col, width)
	// }
}

func (self *DocumentXLSXRenderer) insertImage(row int, col int, imageFilename string, width float64) {
	// self.worksheet.insertImage(row, col, imageFilename)
	// self.update_columnWidth(col, width)
}

func (self *DocumentXLSXRenderer) add_format(formatProps string) {
	// return self.workbook.addFormat(formatProps)
}

func (self *DocumentXLSXRenderer) GenerateXlsx(filename string) {
	// renderer := DocumentXLSXRenderer{}
}

type documentProperties struct {
	ID                    int
	PageFormat            PageFormat
	Orientation           Orientation
	pageWidth             float64
	pageHeight            float64
	contentHeight         float64
	marginLeft            float64
	marginTop             float64
	marginRight           float64
	marginBottom          float64
	PatternLocale         string
	PatternCurrencySymbol string
	header                bool
	headerDisplay         BandDisplay
	headerSize            float64
	footer                bool
	footerDisplay         BandDisplay
	footerSize            float64
	report                *report
}

func (self *documentProperties) init(report *report, data map[string]interface{}) {
	// self.ID = "0_document_properties"
	self.ID = 0
	self.PageFormat = getPageFormat(strings.ToLower(GetStringValue(data, "pageFormat")))
	self.Orientation = getOrientation(GetStringValue(data, "orientation"))
	self.report = report

	var unit Unit
	if self.PageFormat == PageFormatA4 {
		if self.Orientation == OrientationPortrait {
			self.pageWidth = 210
			self.pageHeight = 297
		} else {
			self.pageWidth = 297
			self.pageHeight = 210
		}
		unit = UnitMm
	} else if self.PageFormat == PageFormatA5 {
		if self.Orientation == OrientationPortrait {
			self.pageWidth = 148
			self.pageHeight = 210
		} else {
			self.pageWidth = 210
			self.pageHeight = 148
		}
		unit = UnitMm
	} else if self.PageFormat == PageFormatLetter {
		if self.Orientation == OrientationPortrait {
			self.pageWidth = 8.5
			self.pageHeight = 11
		} else {
			self.pageWidth = 11
			self.pageHeight = 8.5
		}
		unit = UnitMm
	} else {
		self.pageWidth = float64(GetIntValue(data, "pageWidth"))
		self.pageHeight = float64(GetIntValue(data, "pageHeight"))
		unit = GetUnit(GetStringValue(data, "unit"))
		if unit == UnitMm {
			if self.pageWidth < 100 || self.pageWidth >= 100000 {
				self.report.errors = append(self.report.errors, Error{Message: "errorMsgInvalidPageSize", ObjectID: self.ID, Field: "page"})
			} else if self.pageHeight < 100 || self.pageWidth >= 100000 {
				self.report.errors = append(self.report.errors, Error{Message: "errorMsgInvalidPageSize", ObjectID: self.ID, Field: "page"})
			}
		} else if unit == UnitInch {
			if self.pageWidth < 1 || self.pageWidth >= 1000 {
				self.report.errors = append(self.report.errors, Error{Message: "errorMsgInvalidPageSize", ObjectID: self.ID, Field: "page"})
			} else if self.pageHeight < 1 || self.pageWidth >= 1000 {
				self.report.errors = append(self.report.errors, Error{Message: "errorMsgInvalidPageSize", ObjectID: self.ID, Field: "page"})
			}
		}
	}
	dpi := 72.0
	if unit == UnitMm {
		self.pageWidth = math.Round((dpi * self.pageWidth) / 25.4)
		self.pageHeight = math.Round((dpi * self.pageHeight) / 25.4)
	} else {
		self.pageWidth = math.Round((dpi * self.pageWidth))
		self.pageHeight = math.Round((dpi * self.pageHeight))
	}

	self.contentHeight = float64(GetIntValue(data, "contentHeight"))
	self.marginLeft = float64(GetIntValue(data, "marginLeft"))
	self.marginTop = float64(GetIntValue(data, "marginTop"))
	self.marginRight = float64(GetIntValue(data, "marginRight"))
	self.marginBottom = float64(GetIntValue(data, "marginBottom"))
	self.PatternLocale = GetStringValue(data, "patternLocale")
	self.PatternCurrencySymbol = GetStringValue(data, "patternCurrencySymbol")
	if notIn(self.PatternLocale, []string{"de", "en", "es", "fr", "it"}) {
		log.Println("invalid pattern_locale")
	}

	self.header = GetBoolValue(data, "header") // This isn"t working, it should return true but is returning false
	if self.header {
		self.headerDisplay = GetBandDisplay(GetStringValue(data, "headerDisplay"))
		self.headerSize = GetFloatValue(data, "headerSize") // This also doesn"t work
	} else {
		self.headerDisplay = BandDisplayNever
		self.headerSize = 0
	}
	self.footer = GetBoolValue(data, "footer")
	if self.footer {
		self.footerDisplay = GetBandDisplay(GetStringValue(data, "footerDisplay"))
		self.footerSize = float64(GetIntValue(data, "footerSize"))
	} else {
		self.footerDisplay = BandDisplayNever
		self.footerSize = float64(GetIntValue(data, "footerSize"))
	}
	if self.contentHeight == 0 {
		self.contentHeight = self.pageHeight - self.headerSize - self.footerSize - self.marginTop - self.marginBottom
	}
}

func newDocumentProperties(report *report, data map[string]interface{}) documentProperties {
	documentProperties := documentProperties{}
	documentProperties.init(report, data)
	return documentProperties
}

type FPDFRB struct {
	Fpdf            *gofpdf.Fpdf
	CMargin         int
	AdditionalFonts string
	X               float64
	Y               float64
	LoadedImages    map[string]string
	AvailableFonts  map[string]string
	Tr              func(string) string
}

func (self *FPDFRB) init(documentProperties documentProperties, additionalFonts string) {
	var orientation string
	var dimension gofpdf.SizeType
	if documentProperties.Orientation == OrientationPortrait {
		orientation = "P"
		dimension = gofpdf.SizeType{Wd: documentProperties.pageWidth, Ht: documentProperties.pageHeight}
	} else {
		orientation = "L"
		dimension = gofpdf.SizeType{Wd: documentProperties.pageHeight, Ht: documentProperties.pageWidth}
	}
	if documentProperties.PageFormat == PageFormatUserDefined {
		self.Fpdf = gofpdf.NewCustom(&gofpdf.InitType{
			UnitStr:    "pt",
			Size:       dimension,
			FontDirStr: additionalFonts,
		})
	} else {
		self.Fpdf = gofpdf.New(orientation, "pt", documentProperties.PageFormat.String(), additionalFonts)
	}
	self.Fpdf.SetAutoPageBreak(false, 0)
	self.X = 0.0
	self.Y = 0.0
	// "" defaults to "cp1252" | This removes unwanted Â from special characters e.g. £
	self.Tr = self.Fpdf.UnicodeTranslatorFromDescriptor("")
}

func (self *FPDFRB) addImage(img string, imageKey string) {
	self.LoadedImages[imageKey] = img
}

func (self *FPDFRB) getImage(imageKey string) string {
	if val, ok := self.LoadedImages[imageKey]; ok {
		return val
	}
	return ""
}

func newFPDFRB(documentProperties documentProperties, additionalFonts string) FPDFRB {
	fpdfrb := FPDFRB{}
	fpdfrb.init(documentProperties, additionalFonts)
	return fpdfrb
}

type report struct {
	errors             []Error
	documentProperties documentProperties
	containers         containers
	header             containerProvider
	content            containerProvider
	footer             containerProvider
	parameters         map[string]interface{}
	Styles             map[string]textStyle
	Data               map[string]interface{}
	ImageData          map[string][]byte
	IsTestData         bool
	additionalFonts    string
	context            Context
	LogMode            bool
}

func (self *report) init(reportDefinition map[string]interface{}, data map[string]interface{}, isTestData bool, additionalFonts string, imageData map[string][]byte) {
	self.errors = make([]Error, 0)

	self.documentProperties = newDocumentProperties(self, reportDefinition["documentProperties"].(map[string]interface{}))

	self.containers = containers{}
	self.header = newReportBand(BandTypeHeader, "0_header", &self.containers, self)
	self.content = newReportBand(BandTypeContent, "0_content", &self.containers, self)
	self.footer = newReportBand(BandTypeFooter, "0_footer", &self.containers, self)

	self.parameters = make(map[string]interface{}, 0)
	self.Styles = map[string]textStyle{}
	self.Data = map[string]interface{}{}
	self.ImageData = imageData
	self.IsTestData = isTestData

	self.additionalFonts = additionalFonts

	version := GetIntValue(reportDefinition, "version")
	if version != 0 {
		// Convert old report definitions
		if version < 2 {
			for _, docElement := range reportDefinition["docElements"].(map[int]map[string]interface{}) {
				if GetDocElementType(GetStringValue(docElement, "elementType")) == DocElementTypeTable {
					docElement["contentDataRows"] = docElement["contentData"]
				}
			}
		}
	}

	// list is needed to compute parameters (parameters with expression) in given order
	parameterList := make([]interface{}, 0)
	reportParameters := reportDefinition["parameters"].([]interface{})
	for _, item := range reportParameters {
		parameter := NewParameter(self, item.(map[string]interface{}))
		if _, exists := self.parameters[parameter.Name]; exists {
			self.errors = append(self.errors, Error{Message: "errorMsgDuplicateparameter", ObjectID: parameter.ID, Field: "name"})
		}
		self.parameters[parameter.Name] = parameter
		parameterList = append(parameterList, parameter)
	}

	if style, ok := reportDefinition["styles"]; ok {
		for _, item := range style.([]interface{}) {
			style := NewTextStyle(item.(map[string]interface{}), "")
			styleID := GetIntValue(item.(map[string]interface{}), "id")
			self.Styles[cast.ToString(styleID)] = style
		}
	}

	for _, element := range reportDefinition["docElements"].([]interface{}) {
		docElement := element.(map[string]interface{})
		elementType := GetDocElementType(GetStringValue(docElement, "elementType"))
		containerID := GetStringValue(docElement, "containerId")
		if containerID == "" {
			containerID = cast.ToString(GetIntValue(docElement, "containerId"))
		}
		var container *Container
		if containerID != "" {
			if val := self.containers.GetContainer(containerID); val != nil {
				container = val
			}
		}
		var elem DocElementBaseProvider
		if elementType == DocElementTypeText {
			elem = NewTextElement(self, docElement)
		} else if elementType == DocElementTypeLine {
			elem = NewLineElement(self, docElement)
		} else if elementType == DocElementTypeImage {
			elem = NewImageElement(self, docElement)
		} else if elementType == DocElementTypeBarCode {
			elem = NewBarCodeElement(self, docElement)
		} else if elementType == DocElementTypeTable {
			elem = NewTableElement(self, docElement)
		} else if elementType == DocElementTypePageBreak {
			elem = NewPageBreakElement(self, docElement)
		} else if elementType == DocElementTypeFrame {
			elem = NewFrameElement(self, docElement, &self.containers)
		} else if elementType == DocElementTypeSection {
			elem = NewSectionElement(self, docElement, &self.containers)
		}

		if elem != nil && container != nil {
			if container.isVisible() {
				if elem.base().X < 0 {
					self.errors = append(self.errors, Error{Message: "errorMsgInvalidPosition", ObjectID: elem.base().ID, Field: "position"})
				} else if elem.base().X+elem.base().Width > container.Width {
					self.errors = append(self.errors, Error{Message: "errorMsgInvalidSize", ObjectID: elem.base().ID, Field: "position"})
				}
				if elem.base().Y < 0 {
					self.errors = append(self.errors, Error{Message: "errorMsgInvalidPosition", ObjectID: elem.base().ID, Field: "position"})
				} else if elem.base().Y+elem.base().Height > container.Height {
					self.errors = append(self.errors, Error{Message: "errorMsgInvalidSize", ObjectID: elem.base().ID, Field: "position"})
				}
			}
			container.add(elem)
		}
	}

	self.context = NewContext(*self, self.parameters, self.Data)

	computedparameters := map[int]computedParameter{}
	self.processData(&self.Data, data, parameterList, isTestData, &computedparameters, map[int]Parameter{})
	if len(self.errors) < 1 {
		self.computeParameters(computedparameters, self.Data)
	} else {
		for _, err := range self.errors {
			fmt.Println(err)
		}
	}
}

// goes through all elements in header, content and footer and throws a ReportBroError in case
// an element is invalid
func (self *report) verify() {
	if self.documentProperties.headerDisplay != BandDisplayNever {
		self.header.prepare(self.context, nil, true)
	}
	self.content.prepare(self.context, nil, true)
	if self.documentProperties.headerDisplay != BandDisplayNever {
		self.footer.prepare(self.context, nil, true)
	}
}

func (self *report) parseParameterValue(parameter Parameter, parentID int, isTestData bool, ParameterType ParameterType, value interface{}) interface{} {
	errorField := "type"
	if isTestData {
		errorField = "test_data"
	}

	if ParameterType == ParameterTypeString {
		if value != nil {
			if getDataType(value) != DataTypeString {
				log.Println(fmt.Sprintf("value of parameter %s must be str type (unicode for Python 2.7.x)", parameter.Name))
			}
		} else if parameter.Nullable == false {
			value = ""
		}
	} else if ParameterType == ParameterTypeNumber {
		if value != nil && ((getDataType(value) == DataTypeString && cast.ToString(value) != "") || (getDataType(value) == DataTypeFloat && cast.ToFloat64(value) != 0.0) || (getDataType(value) == DataTypeInt && cast.ToInt(value) != 0)) {
			if getDataType(value) == DataTypeString {
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

				value = strings.Replace(string(out), ",", "", -1)
			}
			var err error
			value, err = strconv.ParseFloat(cast.ToString(value), 64)
			if err != nil {
				if parentID != 0 && isTestData {
					self.errors = append(self.errors, Error{Message: "errorMsgInvalidTestData", ObjectID: parentID, Field: "test_data"})
					self.errors = append(self.errors, Error{Message: "errorMsgInvalidNumber", ObjectID: parentID, Field: "type"})
				} else {
					self.errors = append(self.errors, Error{Message: "errorMsgInvalidNumber", Field: errorField, context: parameter.Name})
				}
			}
		} else if value != nil {
			if getDataType(value) == DataTypeInt || getDataType(value) == DataTypeFloat {
				value = 0.0
			} else if isTestData && getDataType(value) == DataTypeString {
				if parameter.Nullable {
					value = nil
				} else {
					value = 0.0
				}
			} else if getDataType(value) != DataTypeFloat {
				value = 0.0
				// if parentID != 0 && isTestData {
				// 	self.errors = append(self.errors, Error{Message: "errorMsgInvalidTestData", ObjectID: parentID, Field: "test_data"})
				// 	self.errors = append(self.errors, Error{Message: "errorMsgInvalidNumber", ObjectID: parentID, Field: "type"})
				// } else {
				// 	value = 0.0
				// 	self.errors = append(self.errors, Error{Message: "errorMsgInvalidNumber", Field: errorField, context: parameter.Name})
				// }
			}
		} else if parameter.Nullable == false {
			value = 0.0
		}

	} else if ParameterType == ParameterTypeBoolean {
		if value != nil {
			value = cast.ToBool(value)
		} else if parameter.Nullable == false {
			value = false
		}
	} else if ParameterType == ParameterTypeDate {
		if getDataType(value) == DataTypeString {
			if isTestData && value == nil {
				if parameter.Nullable {
					value = nil
				} else {
					value = time.Now().Format(time.RFC3339)
				}
			} else {
				format := "2006-01-02"
				colonCount := strings.Count(cast.ToString(value), ":")
				if colonCount == 1 {
					format = "2006-01-02 15:04"
				} else if colonCount == 2 {
					format = "2006-01-02 15:04:05"
				}
				time, err := time.Parse(format, cast.ToString(value))
				if err == nil {
					value = time.Format(format)
				} else {
					// One last attempt
					time, err := dateparse.ParseAny(cast.ToString(value))
					if err == nil {
						value = time.Format(format)
					} else {
						if parentID != 0 && isTestData {
							self.errors = append(self.errors, Error{Message: "errorMsgInvalidTestData", ObjectID: parentID, Field: "test_data", Info: value})
							self.errors = append(self.errors, Error{Message: "errorMsgInvalidDate", ObjectID: parameter.ID, Field: "type", Info: value})
						} else {
							self.errors = append(self.errors, Error{Message: "errorMsgInvalidDate", ObjectID: parameter.ID, Field: errorField, context: parameter.Name, Info: value})
						}
					}
				}
			}
		} else if value != nil {
			if parentID != 0 && isTestData {
				self.errors = append(self.errors, Error{Message: "errorMsgInvalidTestData", ObjectID: parentID, Field: "test_data", Info: value})
				self.errors = append(self.errors, Error{Message: "errorMsgInvalidDate", ObjectID: parameter.ID, Field: "type", Info: value})
			} else {
				self.errors = append(self.errors, Error{Message: "errorMsgInvalidDate", ObjectID: parameter.ID, Field: errorField, context: parameter.Name, Info: value})
			}
		} else if parameter.Nullable == false {
			value = time.Now()
		}
	}

	return value
}

type computedParameter struct {
	parameter   Parameter
	parentNames []string
}

// Here is where the context data is resolved
func (self *report) processData(destData *map[string]interface{}, srcData map[string]interface{}, parameters []interface{}, isTestData bool, computedParameters *map[int]computedParameter, parents map[int]Parameter) {
	field := "type"
	if isTestData {
		field = "test_data"
	}
	parentID := 0
	if len(parents) > 0 {
		parentID = parents[len(parents)-1].ID
	}

	for _, v := range parameters {
		param := v.(Parameter)

		if param.IsInternal {
			continue
		}
		if regexValidIdentifier.MatchString(param.Name) == false {
			self.errors = append(self.errors, Error{Message: "errorMsgInvalidparameterName", ObjectID: param.ID, Field: "name", Info: param.Name})
		}
		parameterType := param.Type
		if param.Eval || (parameterType == ParameterTypeAverage || parameterType == ParameterTypeSum) {
			if param.Expression == "" {
				self.errors = append(self.errors, Error{Message: "errorMsgMissingExpression", ObjectID: param.ID, Field: "expression", context: param.Name})
			} else {
				parentNames := make([]string, 0)
				for _, parent := range parents {
					parentNames = append(parentNames, parent.Name)
				}
				drComputedParameters := *computedParameters
				drComputedParameters[len(drComputedParameters)] = computedParameter{
					parameter:   param,
					parentNames: parentNames,
				}
				computedParameters = &drComputedParameters
			}
		} else {
			value := srcData[param.Name]
			if parameterType == ParameterTypeString || parameterType == ParameterTypeNumber || parameterType == ParameterTypeBoolean || parameterType == ParameterTypeDate {
				value = self.parseParameterValue(param, parentID, isTestData, parameterType, value)
			} else if len(parents) < 1 {
				if parameterType == ParameterTypeArray {
					if value, ok := value.([]interface{}); ok {
						parents[len(parents)] = param
						parameterList := make([]interface{}, 0)
						for _, field := range param.Fields {
							parameterList = append(parameterList, field.(Parameter))
						}
						// create new list which will be assigned to destData to keep srcData unmodified
						destArray := make([]interface{}, 0)

						for _, row := range value {
							destArrayItem := make(map[string]interface{}, 0)
							self.processData(&destArrayItem, row.(map[string]interface{}), parameterList, isTestData, computedParameters, parents)
							destArray = append(destArray, destArrayItem)
						}
						delete(parents, len(parents)-1)
						value = destArray
					} else if value == nil {
						if !param.Nullable {
							value = make([]interface{}, 0)
						}
					} else {
						self.errors = append(self.errors, Error{Message: "errorMsgInvalidArray", ObjectID: param.ID, Field: field, context: param.Name})
					}
				} else if parameterType == ParameterTypeSimpleArray {
					if value, ok := value.([]interface{}); ok {
						listValues := make([]interface{}, 0)
						for _, listValue := range value {
							parsedValue := self.parseParameterValue(param, parentID, isTestData, param.ArrayItemType, listValue)
							listValues = append(listValues, parsedValue)
						}
						value = listValues
					} else if value == nil {
						if param.Nullable == false {
							value = make([]interface{}, 0)
						}
					} else {
						self.errors = append(self.errors, Error{Message: "errorMsgInvalidArray", ObjectID: param.ID, Field: field, context: param.Name})
					}
				} else if parameterType == ParameterTypeMap {
					if value == nil && param.Nullable == false {
						value = make(map[string]interface{}, 0)
					}
					if value, ok := value.(map[string]interface{}); ok {
						if len(param.Children) > 0 {
							parents[len(parents)] = param
							// create new dict which will be assigned to destData to keep srcData unmodified
							destMap := make(map[string]interface{}, 0)

							self.processData(&destMap, value, param.Children, isTestData, computedParameters, parents)
							delete(parents, len(parents)-1)
							value = destMap
						} else {
							self.errors = append(self.errors, Error{Message: "errorMsgInvalidMap", ObjectID: param.ID, Field: "type", context: param.Name})
						}
					} else {
						self.errors = append(self.errors, Error{Message: "errorMsgMissingData", ObjectID: param.ID, Field: "name", context: param.Name})
					}
				}
			}

			(*destData)[param.Name] = value
		}
	}
}

func (self *report) computeParameters(computedParameters map[int]computedParameter, data map[string]interface{}) {
	for _, computedParameter := range computedParameters {
		parameter := computedParameter.parameter
		var value interface{}
		if parameter.Type == ParameterTypeAverage || parameter.Type == ParameterTypeSum {
			expr := stripParameterName(parameter.Expression)
			pos := pyFind(expr, ".", 0, 0)
			if pos == -1 {
				self.errors = append(self.errors, Error{Message: "errorMsgInvalidAvgSumExpression", ObjectID: parameter.ID, Field: "expression", context: parameter.Name})
			} else {
				parameterName := expr[:pos]
				parameterField := expr[pos+1:]
				if items, ok := data[parameterName]; !ok {
					self.errors = append(self.errors, Error{Message: "errorMsgInvalidAvgSumExpression", ObjectID: parameter.ID, Field: "expression", context: parameter.Name})
				} else {
					total := 0.0
					if items != nil {
						for _, item := range items.([]interface{}) {
							if itemValue, ok := item.(map[string]interface{})[parameterField]; !ok {
								self.errors = append(self.errors, Error{Message: "errorMsgInvalidAvgSumExpression", ObjectID: parameter.ID, Field: "expression", context: parameter.Name})
								break
							} else {
								runes := cast.ToString(itemValue)
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

								total += cast.ToFloat64(strings.Replace(string(out), ",", "", -1))
							}
						}
					}
					if parameter.Type == ParameterTypeAverage {
						value = total / float64(len(items.([]interface{})))
					} else if parameter.Type == ParameterTypeSum {
						value = total
					}
				}
			}
		} else {
			value = self.context.evaluateExpression(parameter.Expression, parameter.ID, "expression")
		}

		dataEntry := data
		valid := true
		for _, parentName := range computedParameter.parentNames {
			ldata := dataEntry[parentName]
			if value, ok := ldata.(map[string]interface{}); !ok {
				self.errors = append(self.errors, Error{Message: "errorMsgInvalidparameterData", ObjectID: parameter.ID, Field: "name", context: parameter.Name})
				valid = false
			} else {
				dataEntry = value
			}
		}
		if valid {
			dataEntry[parameter.Name] = value
		}
	}

}

func NewReport(reportDefinition map[string]interface{}, data map[string]interface{}, isTestData bool, additionalFonts string, imageData map[string][]byte) report {
	report := report{}
	report.init(reportDefinition, data, isTestData, additionalFonts, imageData)
	return report
}
