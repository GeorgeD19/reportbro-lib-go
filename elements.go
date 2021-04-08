package reportbro

import (
	"bytes"
	b64 "encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"mime"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jinzhu/now"
	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/barcode"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cast"
	"github.com/vincent-petithory/dataurl"
	"github.com/vjeantet/jodaTime"
)

type DocElementBaseProvider interface {
	base() *DocElementBase
	init(report *report, data map[string]interface{})
	renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB)
	cleanup()
	prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool)
	hasUncompletedPredecessor(completedElements map[int]bool) bool
	getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool)
	isPrinted(ctx Context) bool
	finishEmptyElement(offsetY float64)
	setHeight(height float64)
	addRows(rows []*TableRow, allowSplit bool, availableHeight float64, offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) int
}

type DocElementBase struct {
	Type                   string
	Report                 *report
	ID                     int
	Y                      float64
	RenderY                float64
	RenderBottom           float64
	Bottom                 float64
	Height                 float64
	PrintIf                string
	RemoveEmptyElement     bool
	SpreadsheetHide        bool
	SpreadsheetColumn      *string
	SpreadsheetAddEmptyRow bool
	FirstRenderElement     bool
	RenderingComplete      bool
	Predecessors           []DocElementBaseProvider
	Successors             []DocElementBaseProvider
	SortOrder              float64
	X                      float64
	Width                  float64
	RenderingBottom        bool
	Border                 Border
	BorderStyle            BorderStyle
	BackgroundColor        Color
	TotalHeight            float64
	UsedStyle              *textStyle
	Complete               bool
	TableElement           bool
	BorderColor            Color
	ZIndex                 int
}

func (self *DocElementBase) base() *DocElementBase {
	return self
}

func (self *DocElementBase) init(report *report, data map[string]interface{}) {
	self.Report = report
	self.ID = 0
	self.Y = float64(GetIntValue(data, "y"))
	self.RenderY = 0
	self.RenderBottom = 0
	self.Bottom = self.Y
	self.Height = 0
	self.PrintIf = ""
	self.RemoveEmptyElement = false
	self.SpreadsheetHide = true
	self.SpreadsheetColumn = nil
	self.SpreadsheetAddEmptyRow = false
	self.FirstRenderElement = true
	self.RenderingComplete = false
	self.Predecessors = make([]DocElementBaseProvider, 0)
	self.Successors = make([]DocElementBaseProvider, 0)
	self.Border = BorderNone
	self.SortOrder = 1 // sort order for elements with same "y"-value
	self.ZIndex = 0
}

func (self *DocElementBase) isPredecessor(elem DocElementBaseProvider) bool {
	// if bottom of element is above y-coord of first predecessor we do not need to store
	// the predecessor here because the element is already a predecessor of the first predecessor
	return (self.Y >= elem.base().Bottom && (len(self.Predecessors) == 0 || len(self.Predecessors) > 0 && elem.base().Bottom > self.Predecessors[0].base().Y))
}

func (self *DocElementBase) addPredecessor(predecessor DocElementBaseProvider) {
	self.Predecessors = append(self.Predecessors, predecessor)
	predecessor.base().Successors = append(predecessor.base().Successors, self.base())
}

// returns true in case there is at least one predecessor which is not completely rendered yet
func (self *DocElementBase) hasUncompletedPredecessor(completedElements map[int]bool) bool {
	for _, predecessor := range self.Predecessors {
		if notInElement(predecessor.base().ID, completedElements) || !predecessor.base().RenderingComplete {
			return true
		}
	}
	return false
}

func (self *DocElementBase) GetOffsetY() float64 {
	maxOffsetY := 0.0
	for _, predecessor := range self.Predecessors {
		offsetY := predecessor.base().RenderBottom + (self.Y - predecessor.base().Bottom)
		if offsetY > maxOffsetY {
			maxOffsetY = offsetY
		}
	}
	return maxOffsetY
}

func (self *DocElementBase) clearPredecessors() {
	self.Predecessors = make([]DocElementBaseProvider, 0)
}

func (self *DocElementBase) prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool) {
	return
}

func (self *DocElementBase) isPrinted(ctx Context) bool {
	if self.PrintIf != "" {
		return cast.ToBool(ctx.evaluateExpression(self.PrintIf, self.ID, "printIf"))
	}
	return true
}

func (self *DocElementBase) finishEmptyElement(offsetY float64) {
	if self.RemoveEmptyElement {
		self.RenderBottom = offsetY
	} else {
		self.RenderBottom = offsetY + self.Height
	}
	self.RenderingComplete = true
}

func (self *DocElementBase) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	self.RenderingComplete = true
	return nil, true
}

func (self *DocElementBase) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB) {
	return
}

func (self *DocElementBase) renderSpreadsheet(row int, col int, ctx Context, renderer Renderer) {
	return
}

func (self DocElementBase) cleanup() {
	return
}

func (self *DocElementBase) setHeight(height float64) {}

func (self *DocElementBase) addRows(rows []*TableRow, allowSplit bool, availableHeight float64, offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) int {
	return 0
}

func NewDocElementBase(report *report, data map[string]interface{}) *DocElementBase {
	docElementBase := DocElementBase{}
	docElementBase.init(report, data)
	return &docElementBase
}

type DocElement struct {
	DocElementBase
}

func (self *DocElement) init(report *report, data map[string]interface{}) {
	self.DocElementBase.init(report, data)
	self.ID = GetIntValue(data, "id")
	self.ZIndex = GetIntValue(data, "zIndex")
	self.X = float64(GetIntValue(data, "x"))
	self.Width = float64(GetIntValue(data, "width"))
	self.Height = float64(GetIntValue(data, "height"))
	self.Bottom = self.Y + self.Height
}

func (self *DocElement) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	if offsetY+self.Height <= containerHeight {
		self.RenderY = offsetY
		self.RenderBottom = offsetY + self.Height
		self.RenderingComplete = true
		return self, true
	}
	return nil, false
}

// @staticmethod
func drawBorder(x float64, y float64, width float64, height float64, renderElementType RenderElementType, borderStyle Style, pdfDoc *FPDFRB) {
	pdfDoc.Fpdf.SetDrawColor(borderStyle.base().BorderStyle.BorderColor.R, borderStyle.base().BorderStyle.BorderColor.G, borderStyle.base().BorderStyle.BorderColor.B)
	pdfDoc.Fpdf.SetLineWidth(borderStyle.base().BorderStyle.BorderWidth)
	borderOffset := borderStyle.base().BorderStyle.BorderWidth / 2
	borderX := x + borderOffset
	borderY := y + borderOffset
	borderWidth := width - borderStyle.base().BorderStyle.BorderWidth
	borderHeight := height - borderStyle.base().BorderStyle.BorderWidth
	if borderStyle.base().BorderStyle.BorderAll && renderElementType == RenderElementTypeComplete {
		pdfDoc.Fpdf.Rect(borderX, borderY, borderWidth, borderHeight, "D")
	} else {
		if borderStyle.base().BorderStyle.BorderLeft && (renderElementType == RenderElementTypeComplete || renderElementType == RenderElementTypeFirst) {
			pdfDoc.Fpdf.Line(borderX, borderY, borderX+borderWidth, borderY)
		}
		if borderStyle.base().BorderStyle.BorderTop && (renderElementType == RenderElementTypeComplete || renderElementType == RenderElementTypeFirst) {
			pdfDoc.Fpdf.Line(borderX, borderY, borderX+borderWidth, borderY)
		}
		if borderStyle.base().BorderStyle.BorderRight {
			pdfDoc.Fpdf.Line(borderX+borderWidth, borderY, borderX+borderWidth, borderY+borderHeight)
		}
		if borderStyle.base().BorderStyle.BorderBottom && (renderElementType == RenderElementTypeComplete || renderElementType == RenderElementTypeLast) {
			pdfDoc.Fpdf.Line(borderX, borderY+borderHeight, borderX+borderWidth, borderY+borderHeight)
		}
	}
}

func NewDocElement(report *report, data map[string]interface{}) *DocElement {
	docElement := DocElement{}
	docElement.init(report, data)
	return &docElement
}

type ImageElement struct {
	DocElement
	Eval                   bool
	Source                 string
	Content                string
	IsContent              bool
	Image                  string
	ImageFile              string
	ImageFilename          string
	HorizontalAlignment    HorizontalAlignment
	VerticalAlignment      VerticalAlignment
	BackgroundColor        Color
	RemoveEmptyElement     bool
	Link                   string
	SpreadsheetHide        bool
	SpreadsheetColumn      int
	SpreadsheetAddEmptyRow bool
	ImageKey               string
	ImageType              string
	Image64                string
	ImageFP                []byte
	ImageHeight            float64
	UsedStyle              textStyle
	SpaceTop               float64
	SpaceBottom            float64
}

func (self *ImageElement) init(report *report, data map[string]interface{}) {
	self.DocElement.init(report, data)
	self.Eval = GetBoolValue(data, "eval")
	self.Source = GetStringValue(data, "source")
	self.Content = GetStringValue(data, "content")
	self.IsContent = GetBoolValue(data, "isContent")
	if self.Source == "" {
		self.IsContent = true
	}

	self.Image = GetStringValue(data, "image")
	self.ImageFilename = GetStringValue(data, "imageFilename")
	self.HorizontalAlignment = GetHorizontalAlignment(GetStringValue(data, "horizontalAlignment"))
	self.VerticalAlignment = GetVerticalAlignment(GetStringValue(data, "verticalAlignment"))
	self.BackgroundColor = NewColor(GetStringValue(data, "backgroundColor"))
	self.PrintIf = GetStringValue(data, "printIf")
	self.RemoveEmptyElement = GetBoolValue(data, "removeEmptyElement")
	self.Link = GetStringValue(data, "link")
	self.SpreadsheetHide = GetBoolValue(data, "spreadsheet_hide")
	self.SpreadsheetColumn = GetIntValue(data, "spreadsheet_column")
	self.SpreadsheetAddEmptyRow = GetBoolValue(data, "spreadsheet_addEmptyRow")
	self.ImageKey = ""
	self.ImageType = ""
	self.ImageFP = nil
	self.TotalHeight = 0
	self.ImageHeight = 0
	self.UsedStyle = NewTextStyle(data, "")
}

func (self *ImageElement) setHeight(height float64) {
	self.Height = height
	self.SpaceTop = 0.0
	self.SpaceBottom = 0.0
	totalHeight := 0.0
	if self.ImageHeight > 0 {
		totalHeight = self.ImageHeight
	}
	self.TotalHeight = totalHeight
}

func (self *ImageElement) prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool) {
	if self.ImageKey != "" {
		return
	}
	imgDataB64 := ""
	isURL := false
	if self.Source != "" {
		parameterName := stripParameterName(self.Source)
		parameter := ctx.getParameter(stripParameterName(self.Source), nil)
		if parameter != nil {
			sourceparameter := parameter.(map[string]interface{})[parameterName].(Parameter)
			if sourceparameter.Type == ParameterTypeString {
				imageKey, _ := ctx.getData(sourceparameter.Name, nil)
				if imageKey != nil {
					self.ImageKey = imageKey.(string)
				}
				isURL = true
			} else if sourceparameter.Type == ParameterTypeImage {
				// image is available as base64 encoded or
				// file object (only possible if report data is passed directly from python code
				// and not via web request)
				imgData, _ := ctx.getData(sourceparameter.Name, nil)
				if reflect.TypeOf(imgData) == reflect.TypeOf("") {
					imgDataB64 = imgData.(string)
				}
			} else {
				log.Println(Error{Message: "errorMsgInvalidImageSourceparameter", ObjectID: self.base().ID, Field: "source"})
			}
		} else {
			source := pyStrip(self.Source, "")
			if source[0:2] == "${" && source[len(source)-1:len(source)] == "}" {
				log.Println(Error{Message: "errorMsgMissingparameter", ObjectID: self.base().ID, Field: "source"})
			}
			self.ImageKey = self.Source
			isURL = true
		}
	}

	if self.IsContent {
		parameterName := stripParameterName(self.Content)
		parameter := ctx.getParameter(parameterName, nil)
		if parameter != nil {
			sourceparameter := parameter.(map[string]interface{})[parameterName].(Parameter)
			imgData, parameterExists := ctx.getData(sourceparameter.Name, nil)
			if parameterExists {
				imgDataB64 = imgData.(string)
			}
		}
	}

	if imgDataB64 == "" && !isURL && self.ImageFP == nil {
		if self.ImageFilename != "" && self.Image != "" {
			// static image base64 encoded within image element
			imgDataB64 = self.Image
			self.ImageKey = self.ImageFilename
		}
	}

	if imgDataB64 != "" {
		self.Image64 = imgDataB64
		re, _ := regexp.Compile(`^data:image/(.+);base64,`)
		m := re.MatchString(imgDataB64)
		if !m {
			log.Println(Error{Message: "errorMsgInvalidImage", ObjectID: self.base().ID, Field: "source"})
		}
		dataURL, _ := dataurl.DecodeString(imgDataB64)
		extensions, _ := mime.ExtensionsByType(dataURL.MediaType.ContentType()) // [.jfif, .jpe, .jpeg, .jpg]

		invalid := map[string]string{
			".":    "",
			".jpe": ".jpg",
			"jpeg": "jpg",
			"jpe":  "jpg",
			"jfif": "jpg",
		}
		imageType := extensions[0]
		for s, r := range invalid {
			imageType = strings.Replace(imageType, s, r, -1)
		}
		self.ImageType = imageType
		imgData, _ := b64.StdEncoding.DecodeString(strings.Replace(imgDataB64, "^data:image/.+;base64,", "", -1))
		self.ImageFP = imgData
	} else if isURL {
		// if !(self.ImageKey && (self.ImageKey.startswith("http://") || strings.HasPrefix(self.imageKey, "https://"))) {
		// 	log.Println(Error{Message: "errorMsgInvalidImageSource", ObjectID: self.base().ID, Field: "source"})
		// }
		// pos = self.imageKey.rfind('.')
		// if pos != -1 {
		// 	self.ImageType = self.ImageKey[pos+1:]
		// } else {
		// 	self.ImageType = ""
		// }
	}

	if self.ImageType != "" {
		if !inArray(self.ImageType, []string{"png", "jpg", "jpeg"}) {
			log.Println(Error{Message: "errorMsgUnsupportedImageType", ObjectID: self.base().ID, Field: "source"})
		}
		if self.ImageKey == "" {
			self.ImageKey = "image_" + strings.ToUpper(fmt.Sprint(uuid.NewV4())) + "." + self.ImageType
			// self.ImageKey = "image_" + string(self.base().ID) + "." + self.ImageType
		}
	}
	self.Image = ""

	if self.Link != "" {
		self.Link = ctx.fillParameters(self.Link, self.base().ID, "link", "")
	}
}

func (self *ImageElement) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB) {
	x := self.X + containerOffsetX
	y := self.RenderY + containerOffsetY
	if self.BackgroundColor.Transparent == false {
		pdfDoc.Fpdf.SetFillColor(self.BackgroundColor.R, self.BackgroundColor.G, self.BackgroundColor.B)
		pdfDoc.Fpdf.Rect(x, y, self.Width, self.Height, "F")
	}

	if self.ImageKey != "" {
		options := gofpdf.ImageOptions{
			ReadDpi:               false,
			AllowNegativePosition: true,
		}
		dataURL, _ := dataurl.DecodeString(self.Image64)
		image, _, err := image.DecodeConfig(bytes.NewReader([]byte(dataURL.Data)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}

		// Detect if either width or height has been made smaller than original at all
		cwidth := false
		imageWidth := float64(image.Width)
		if float64(image.Width) >= self.Width {
			aW := (float64(image.Width) / float64(image.Height)) * self.Height
			if aW < self.Width {
				imageWidth = aW
				cwidth = true
			} else {
				imageWidth = self.Width
				cwidth = true
			}
		}

		// If has been made smaller on either axis, we must adjust the other axis to match the aspect ratio
		cheight := false
		imageHeight := float64(image.Height)
		if float64(image.Height) >= self.Height {
			aH := self.Width / (float64(image.Width) / float64(image.Height))
			if aH < self.Height {
				imageHeight = aH
				cheight = true
			} else {
				imageHeight = self.Height
				cheight = true
			}
		}

		if cwidth {
			imageHeight = imageWidth / (float64(image.Width) / float64(image.Height))
		}

		if cheight {
			imageWidth = (float64(image.Width) / float64(image.Height)) * imageHeight
		}

		imageX := x
		switch self.HorizontalAlignment {
		case HorizontalAlignmentCenter:
			imageX += ((self.Width - imageWidth) / 2)
			break
		case HorizontalAlignmentRight:
			imageX += (self.Width - imageWidth)
			break
		}

		imageY := y
		switch self.VerticalAlignment {
		case VerticalAlignmentMiddle:
			imageY += ((self.Height - imageHeight) / 2)
			break
		case VerticalAlignmentBottom:
			imageY += (self.Height - imageHeight)
			break
		}

		options.ImageType = self.ImageType

		if options.ImageType != "" {
			pdfDoc.Fpdf.RegisterImageOptionsReader(self.ImageKey, options, bytes.NewReader(dataURL.Data))
			pdfDoc.Fpdf.ImageOptions(self.ImageKey, imageX, imageY, imageWidth, imageHeight, false, options, 0, "")
		}

	}

	if self.Link != "" {
		// horizontal and vertical alignment of image within given width and height
		// by keeping original image aspect ratio
		offsetY := 0.0
		offsetX := offsetY
		imageDisplayWidth, imageDisplayHeight := 0.0, 0.0
		imageWidth, imageHeight := self.Width, self.Height
		if imageWidth <= self.Width && imageHeight <= self.Height {
			imageDisplayWidth, imageDisplayHeight = imageWidth, imageHeight
		} else {
			sizeRatio := imageWidth / imageHeight
			tmp := self.Width / sizeRatio
			if tmp <= self.Height {
				imageDisplayWidth = self.Width
				imageDisplayHeight = tmp
			} else {
				imageDisplayWidth = self.Height * sizeRatio
				imageDisplayHeight = self.Height
			}
		}
		if self.HorizontalAlignment == HorizontalAlignmentCenter {
			offsetX = ((self.Width - imageDisplayWidth) / 2)
		} else if self.HorizontalAlignment == HorizontalAlignmentRight {
			offsetX = self.Width - imageDisplayWidth
		}
		if self.VerticalAlignment == VerticalAlignmentMiddle {
			offsetY = ((self.Height - imageDisplayHeight) / 2)
		} else if self.VerticalAlignment == VerticalAlignmentBottom {
			offsetY = self.Height - imageDisplayHeight
		}

		linkID := pdfDoc.Fpdf.AddLink()
		pdfDoc.Fpdf.Link(x+offsetX, y+offsetY, imageDisplayWidth, imageDisplayHeight, linkID)
	}
}

func (self *ImageElement) renderSpreadsheet(row int, col int, ctx Context, renderer Renderer) (int, int) {
	if self.ImageKey != "" {
		if self.SpreadsheetColumn != 0 {
			col = cast.ToInt(self.SpreadsheetAddEmptyRow) - 1
		}
		renderer.insertImage(row, col, self.ImageKey, self.Width)
		if self.SpreadsheetAddEmptyRow {
			row += 2
		} else {
			row++
		}
	}
	return row, col
}

func (self *ImageElement) cleanup() {
	if self.ImageKey != "" {
		self.ImageKey = ""
	}
}

func (self *ImageElement) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	self.RenderingComplete = true // Hacky
	if offsetY+self.Height <= containerHeight {
		self.RenderY = offsetY
		self.RenderBottom = offsetY + self.Height
		self.RenderingBottom = true
		return self, true
	}
	return nil, false
}

func NewImageElement(report *report, data map[string]interface{}) *ImageElement {
	imageElement := ImageElement{}
	imageElement.init(report, data)
	return &imageElement
}

type BarCodeElement struct {
	DocElement
	Content                string
	Format                 string
	DisplayValue           bool
	RemoveEmptyElement     bool
	SpreadsheetHide        bool
	SpreadsheetColumn      int
	SpreadsheetColspan     int
	SpreadsheetAddEmptyRow bool
	ImageKey               *string
	ImageHeight            float64
}

func (self *BarCodeElement) init(report *report, data map[string]interface{}) {
	self.DocElement.init(report, data)
	self.Content = GetStringValue(data, "content")
	self.Format = strings.ToLower(GetStringValue(data, "format"))
	if self.Format != "code128" {
		panic(self.Format)
	}
	self.DisplayValue = GetBoolValue(data, "displayValue")
	self.PrintIf = GetStringValue(data, "printIf")
	self.RemoveEmptyElement = GetBoolValue(data, "removeEmptyElement")
	self.SpreadsheetHide = GetBoolValue(data, "spreadsheet_hide")
	self.SpreadsheetColumn = GetIntValue(data, "spreadsheet_column")
	self.SpreadsheetColspan = GetIntValue(data, "spreadsheet_colspan")
	self.SpreadsheetAddEmptyRow = GetBoolValue(data, "spreadsheet_addEmptyRow")
	self.ImageKey = nil
	if self.DisplayValue == true {
		self.ImageHeight = self.Height - 22.0
	} else {
		self.ImageHeight = self.Height
	}
}

func (self *BarCodeElement) isPrinted(ctx Context) bool {
	if self.Content == "" {
		return false
	}
	return self.DocElementBase.isPrinted(ctx)
}

func (self *BarCodeElement) prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool) {
	if self.ImageKey != nil {
		return
	}
	self.Content = ctx.fillParameters(self.Content, self.ID, "content", "")
	if self.Content != "" {
		self.Width = 136
		imageKey := barcode.RegisterCode128(pdfDoc.Fpdf, self.Content)
		self.ImageKey = &imageKey
	}

}

func (self *BarCodeElement) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	if offsetY+self.Height <= containerHeight {
		self.RenderY = offsetY
		self.RenderBottom = offsetY + self.Height
		self.RenderingComplete = true
		return self, true
	}
	return nil, false
}

func (self *BarCodeElement) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB) {
	x := self.X + containerOffsetX
	y := self.RenderY + containerOffsetY
	if self.ImageKey != nil {
		barcode.Barcode(pdfDoc.Fpdf, *self.ImageKey, x, y, self.Width, self.ImageHeight, false)
		if self.DisplayValue != false {
			pdfDoc.Fpdf.SetFont("courier", "B", 18)
			pdfDoc.Fpdf.SetTextColor(0, 0, 0)
			contentWidth := pdfDoc.Fpdf.GetStringWidth(self.Content)
			OffsetX := (self.Width - contentWidth) / 2
			pdfDoc.Fpdf.Text(x+OffsetX, y+self.ImageHeight+20, self.Content)
		}
	}

}

func (self *BarCodeElement) renderSpreadsheet(row int, col int, ctx Context, renderer Renderer) (int, int) {
	if self.Content != "" {
		cellFormat := ""
		if self.SpreadsheetColumn != 0 {
			col = self.SpreadsheetColumn - 1
		}
		renderer.write(row, col, self.SpreadsheetColspan, self.Content, cellFormat, self.Width)
		if self.SpreadsheetAddEmptyRow {
			row += 2
		} else {
			row++
		}
		col++
	}
	return row, col
}

func (self BarCodeElement) cleanup() {
	if self.ImageKey != nil {
		// os.unlink(self.imageKey)
		self.ImageKey = nil
	}
}

func NewBarCodeElement(report *report, data map[string]interface{}) *BarCodeElement {
	barCodeElement := BarCodeElement{}
	barCodeElement.init(report, data)
	return &barCodeElement
}

type LineElement struct {
	DocElement
	Color Color
}

func (self *LineElement) init(report *report, data map[string]interface{}) {
	self.DocElement.init(report, data)
	self.Color = NewColor(GetStringValue(data, "color"))
	self.PrintIf = GetStringValue(data, "printIf")
}

func (self *LineElement) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB) {
	pdfDoc.Fpdf.SetDrawColor(self.Color.R, self.Color.G, self.Color.B)
	pdfDoc.Fpdf.SetLineWidth(self.Height)
	x := self.X + containerOffsetX
	y := self.Y + containerOffsetY + (self.Height / 2)
	pdfDoc.Fpdf.Line(x, y, x+self.Width, y)
}

func (self *LineElement) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	if offsetY+self.Height <= containerHeight {
		self.RenderY = offsetY
		self.RenderBottom = offsetY + self.Height
		self.RenderingBottom = true
		self.RenderingComplete = true
		return self, true
	}
	return nil, false
}

func NewLineElement(report *report, data map[string]interface{}) *LineElement {
	lineElement := LineElement{}
	lineElement.init(report, data)
	return &lineElement
}

type PageBreakElement struct {
	DocElementBase
}

func (self *PageBreakElement) init(report *report, data map[string]interface{}) {
	self.DocElementBase.init(report, data)
	self.ID = GetIntValue(data, "id")
	self.X = 0.0
	self.Width = 0.0
	self.SortOrder = 0 // sort order for elements with same 'y'-value, render page break before other elements
}

func (self *PageBreakElement) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	return self, true
}

func NewPageBreakElement(report *report, data map[string]interface{}) *PageBreakElement {
	pageBreakElement := PageBreakElement{}
	pageBreakElement.init(report, data)
	return &pageBreakElement
}

type TextElement struct {
	DocElement
	Content                          string
	Eval                             bool
	Style                            textStyle
	Pattern                          string
	Link                             string
	CsCondition                      string
	ConditionalStyle                 *textStyle
	TextHeight                       float64
	LineIndex                        int
	LineHeight                       float64
	LinesCount                       int
	TextLines                        []TextLine
	SpaceTop                         float64
	SpaceBottom                      float64
	SpreadsheetCellFormat            *int
	SpreadsheetCellFormatinitialized bool
	SpreadsheetColspan               int
	AlwaysPrintOnSamePage            bool
}

func (self *TextElement) init(report *report, data map[string]interface{}) {
	self.Type = DocElementTypeText.String()
	self.DocElement.init(report, data)

	self.Content = GetStringValue(data, "content")
	self.Eval = GetBoolValue(data, "eval")
	if GetIntValue(data, "styleId") != 0 {
		if style, ok := report.Styles[cast.ToString(GetIntValue(data, "styleId"))]; ok {
			self.Style = style
		} else {
			log.Println(Error{Message: fmt.Sprintf("Style for text element %s not found", self.ID)})
		}
	} else {
		self.Style = NewTextStyle(data, "")
	}
	self.PrintIf = GetStringValue(data, "printIf")
	self.Pattern = GetStringValue(data, "pattern")
	self.Link = GetStringValue(data, "link")
	self.CsCondition = GetStringValue(data, "cs_condition")
	if self.CsCondition != "" {
		if GetStringValue(data, "cs_styleId") != "" {
			if val, ok := report.Styles[cast.ToString(GetIntValue(data, "cs_styleId"))]; ok {
				self.ConditionalStyle = &val
			}
			if self.ConditionalStyle == nil {
				log.Println(Error{Message: fmt.Sprintf("Conditional style for text element %s not found", self.ID)})
			}
		} else {
			style := NewTextStyle(data, "cs_")
			self.ConditionalStyle = &style
		}
	} else {
		self.ConditionalStyle = nil
	}

	if getDataType(self) == DataTypeTableTextElement {
		self.RemoveEmptyElement = false
		self.AlwaysPrintOnSamePage = false
	} else {
		self.RemoveEmptyElement = GetBoolValue(data, "removeEmptyElement")
		self.AlwaysPrintOnSamePage = GetBoolValue(data, "alwaysPrintOnSamePage")
	}
	self.Height = float64(GetIntValue(data, "height"))
	self.SpreadsheetHide = GetBoolValue(data, "spreadsheet_hide")
	self.SpreadsheetColumn = StringPointer(cast.ToString(GetIntValue(data, "spreadsheet_column")))
	self.SpreadsheetColspan = GetIntValue(data, "spreadsheet_colspan")
	self.SpreadsheetAddEmptyRow = GetBoolValue(data, "spreadsheet_addEmptyRow")
	self.TextHeight = 0.0
	self.LineIndex = -1
	self.LineHeight = 0
	self.LinesCount = 0
	self.TextLines = nil
	self.UsedStyle = nil
	self.SpaceTop = 0
	self.SpaceBottom = 0
	self.TotalHeight = 0
	self.SpreadsheetCellFormat = nil
	self.SpreadsheetCellFormatinitialized = false
}

func (self *TextElement) isPrinted(ctx Context) bool {
	if self.RemoveEmptyElement && len(self.TextLines) == 0 {
		return false
	}
	return self.DocElementBase.isPrinted(ctx)
}

func (self *TextElement) prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool) {
	var content interface{}

	if self.Eval {
		content = ctx.evaluateExpression(self.Content, self.ID, "content")

		if self.Pattern != "" {
			if (reflect.TypeOf(content) == reflect.TypeOf(0)) || (reflect.TypeOf(content) == reflect.TypeOf(0.0)) {
				usedPattern := self.Pattern
				patternHasCurrency := strings.Contains(usedPattern, "$")
				if usedPattern != "" {
					usedPattern = strings.Replace(usedPattern, "0", "#", -1)
					usedPattern = strings.Replace(usedPattern, " ", "", -1)
					if patternHasCurrency {
						usedPattern = strings.Replace(usedPattern, "$", "", -1)
						content = humanize.FormatFloat(usedPattern, cast.ToFloat64(content))
						content = ctx.PatternCurrencySymbol + cast.ToString(content)
					} else {
						content = humanize.FormatFloat(usedPattern, cast.ToFloat64(content))
					}
				}
			} else if reflect.TypeOf(content) == reflect.TypeOf(time.Time{}) {
				usedPattern := self.Pattern
				if usedPattern != "" {
					t, err := now.Parse(cast.ToString(content))
					if err == nil {
						date := jodaTime.Format(usedPattern, t)
						content = date
					}
				}
			}
		}
		content = cast.ToString(content)
	} else {
		content = ctx.fillParameters(self.Content, self.ID, "content", self.Pattern)
	}

	if self.Link != "" {
		self.Link = ctx.fillParameters(self.Link, self.ID, "link", "")
	}

	if self.CsCondition != "" {
		if cast.ToBool(ctx.evaluateExpression(self.CsCondition, self.ID, "cs_condition")) {
			self.UsedStyle = self.ConditionalStyle
		} else {
			self.UsedStyle = &self.Style
		}
	} else {
		self.UsedStyle = &self.Style
	}

	if getDataType(self) == DataTypeTableTextElement {
		if self.UsedStyle.VerticalAlignment != VerticalAlignmentTop && self.AlwaysPrintOnSamePage == false {
			self.AlwaysPrintOnSamePage = true
		}
	}

	availableWidth := self.Width - self.UsedStyle.PaddingLeft - self.UsedStyle.PaddingRight

	self.TextLines = make([]TextLine, 0)
	if pdfDoc.Fpdf != nil {
		pdfDoc.Fpdf.SetFont(self.UsedStyle.Font, self.UsedStyle.FontStyle, self.UsedStyle.FontSize)
		lines := make([]TextLine, 0)
		if content != "" {
			tmpLines := pdfDoc.Fpdf.SplitLines([]byte(cast.ToString(content)), availableWidth)
			for split := 0; split < len(tmpLines); split++ {
				lines = append(lines, NewTextLine(string(tmpLines[split]), availableWidth, self.UsedStyle, self.Link))
			}
			content = strings.Replace(cast.ToString(content), "\n", " \n ", -1)
		}

		self.LineHeight = self.UsedStyle.base().FontSize * self.UsedStyle.base().LineSpacing
		self.LinesCount = len(lines)
		if self.LinesCount > 0 {
			self.TextHeight = float64((len(lines)-1))*self.LineHeight + self.UsedStyle.base().FontSize
		}
		self.LineIndex = 0
		for _, line := range lines {
			self.TextLines = append(self.TextLines, line)
		}
		if self.TableElement {
			self.TotalHeight = max(self.TextHeight+self.UsedStyle.PaddingTop+self.UsedStyle.PaddingBottom, self.Height)
		} else {
			self.setHeight(self.Height)
		}
	} else {
		self.Content = cast.ToString(content)
		// set textLines so isPrinted can check for empty element when rendering spreadsheet
		if content != "" {
			self.TextLines = make([]TextLine, 0)
			self.TextLines = append(self.TextLines, NewTextLine(cast.ToString(content), availableWidth, self.UsedStyle, self.Link))
		}
	}
}

func (self *TextElement) setHeight(height float64) {
	self.Height = height
	self.SpaceTop = 0.0
	self.SpaceBottom = 0.0
	totalHeight := 0.0
	if self.TextHeight > 0 {
		totalHeight = self.TextHeight + self.UsedStyle.PaddingTop + self.UsedStyle.PaddingBottom
	}
	if totalHeight < height {
		remainingSpace := height - totalHeight
		if self.UsedStyle.VerticalAlignment == VerticalAlignmentTop {
			self.SpaceBottom = remainingSpace
		} else if self.UsedStyle.VerticalAlignment == VerticalAlignmentMiddle {
			self.SpaceTop = remainingSpace / 2
			self.SpaceBottom = remainingSpace / 2
		} else if self.UsedStyle.VerticalAlignment == VerticalAlignmentBottom {
			self.SpaceTop = remainingSpace
		}
	}
	self.TotalHeight = totalHeight + self.SpaceTop + self.SpaceBottom
}

func (self *TextElement) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	availableHeight := containerHeight - offsetY
	if self.AlwaysPrintOnSamePage && self.FirstRenderElement && self.TotalHeight > availableHeight && offsetY != 0 {
		return nil, false
	}
	lines := make([]TextLine, 0)
	remainingHeight := availableHeight
	blockHeight := 0.0
	textHeight := 0.0
	textOffsetY := 0.0
	if self.SpaceTop > 0 {
		spaceTop := pyMin(self.SpaceTop, remainingHeight)
		self.SpaceTop -= spaceTop
		blockHeight += spaceTop
		remainingHeight -= spaceTop
		textOffsetY = spaceTop
	}
	if self.SpaceTop == 0 {
		firstLine := true
		for self.LineIndex < self.LinesCount {
			lastLine := (self.LineIndex >= self.LinesCount-1)
			lineHeight := self.LineHeight
			if firstLine && self.UsedStyle != nil {
				lineHeight = self.UsedStyle.FontSize
			}
			tmpHeight := lineHeight
			if self.LineIndex == 0 {
				tmpHeight += self.UsedStyle.PaddingTop
			}
			if lastLine && self.UsedStyle != nil {
				tmpHeight += self.UsedStyle.PaddingBottom
			}
			if tmpHeight > remainingHeight {
				break
			}
			lines = append(lines, self.TextLines[self.LineIndex])
			remainingHeight -= tmpHeight
			blockHeight += tmpHeight
			textHeight += lineHeight
			self.LineIndex++
			firstLine = false
		}
	}
	if self.LineIndex >= self.LinesCount && self.SpaceBottom > 0 {
		spaceBottom := pyMin(self.SpaceBottom, remainingHeight)
		self.SpaceBottom -= spaceBottom
		blockHeight += spaceBottom
		remainingHeight -= spaceBottom
	}
	if self.SpaceTop == 0 && self.LineIndex == 0 && self.LinesCount > 0 {
		// even first line does not fit
		if offsetY != 0 {
			// try on next container
			return nil, false
		} else {
			// already on top of container -> raise error
			log.Println("errorMsgInvalidSize", self.ID, "size")
		}
	}
	renderingComplete := (self.LineIndex >= self.LinesCount && self.SpaceTop == 0 && self.SpaceBottom == 0)
	if renderingComplete == false && remainingHeight > 0 {
		// draw text block until end of container
		blockHeight += remainingHeight
		remainingHeight = 0
	}
	var renderElementType RenderElementType
	if self.FirstRenderElement && renderingComplete {
		renderElementType = RenderElementTypeComplete
	} else {
		if self.FirstRenderElement {
			renderElementType = RenderElementTypeFirst
		} else if renderingComplete {
			renderElementType = RenderElementTypeLast
			if self.UsedStyle.VerticalAlignment == VerticalAlignmentBottom {
				// make sure text is exactly aligned to bottom
				tmpOffsetY := blockHeight - self.UsedStyle.PaddingBottom - textHeight
				if tmpOffsetY > 0 {
					textOffsetY = tmpOffsetY
				}
			}
		} else {
			renderElementType = RenderElementTypeBetween
		}
	}

	textBlockElem := NewTextBlockElement(self.Report, self.X, self.Y, offsetY, self.Width, blockHeight, textOffsetY, lines, self.LineHeight, renderElementType, self.UsedStyle)
	self.FirstRenderElement = false
	self.RenderBottom = textBlockElem.RenderBottom
	self.RenderingComplete = renderingComplete
	return textBlockElem, renderingComplete
}

func (self *TextElement) isFirstRenderElement() bool {
	return self.FirstRenderElement
}

func (self *TextElement) renderSpreadsheet(row int, col int, ctx Context, renderer Renderer) {
	return
}

func NewTextElement(report *report, data map[string]interface{}) *TextElement {
	textElement := TextElement{}
	textElement.init(report, data)
	return &textElement
}

type TextBlockElement struct {
	DocElementBase
	TextOffsetY       float64
	Lines             []TextLine
	LineHeight        float64
	RenderElementType RenderElementType
	Style             Style
}

func (self *TextBlockElement) initTextBlockElement(report *report, x float64, y float64, renderY float64, width float64, height float64, textOffsetY float64, lines []TextLine, lineHeight float64, renderElementType RenderElementType, style Style) {
	self.DocElementBase.init(report, map[string]interface{}{"y": cast.ToString(y)})
	self.X = x
	self.RenderY = renderY
	self.RenderBottom = renderY + height
	self.Width = width
	self.Height = height
	self.TextOffsetY = textOffsetY
	self.Lines = lines
	self.LineHeight = lineHeight
	self.RenderElementType = renderElementType
	defaultStyle := NewTextStyle(map[string]interface{}{}, "")
	self.Style = &defaultStyle
	if !reflect.ValueOf(style).IsNil() {
		self.Style = style
	}
}

func (self *TextBlockElement) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB) {
	y := containerOffsetY + self.RenderY
	if self.Style.base().BackgroundColor.Transparent != true {
		pdfDoc.Fpdf.SetFillColor(self.Style.base().BackgroundColor.R, self.Style.base().BackgroundColor.G, self.Style.base().BackgroundColor.B)
		pdfDoc.Fpdf.Rect(self.X+containerOffsetX, y, self.Width, self.Height, "F")
	}
	if self.Style.base().BorderStyle.BorderLeft || self.Style.base().BorderStyle.BorderTop || self.Style.base().BorderStyle.BorderRight || self.Style.base().BorderStyle.BorderBottom {
		drawBorder(self.X+containerOffsetX, y, self.Width, self.Height, self.RenderElementType, self.Style, pdfDoc)
	}

	if self.RenderElementType == RenderElementTypeComplete || self.RenderElementType == RenderElementTypeFirst {
		y += self.Style.base().PaddingTop
	}
	y += self.TextOffsetY

	underline := self.Style.base().Underline
	lastLineIndex := len(self.Lines) - 1
	// underline for justified text is drawn manually to have a single line for the
	// whole text. each word is rendered individually,
	// therefor we can"t use the underline style of the rendered text
	if self.Style.base().HorizontalAlignment == HorizontalAlignmentJustify && lastLineIndex > 0 {
		underline = false
		pdfDoc.Fpdf.SetDrawColor(self.Style.base().TextColor.R, self.Style.base().TextColor.G, self.Style.base().TextColor.B)
	}
	if underline {
		self.Style.base().FontStyle += "U"
	}
	pdfDoc.Fpdf.SetFont(self.Style.base().Font, self.Style.base().FontStyle, self.Style.base().FontSize)
	pdfDoc.Fpdf.SetTextColor(self.Style.base().TextColor.R, self.Style.base().TextColor.G, self.Style.base().TextColor.B)

	for i, line := range self.Lines {
		lastLine := (i == lastLineIndex)
		line.renderPDF(self.X+containerOffsetX+self.Style.base().PaddingLeft, y, lastLine, pdfDoc)
		y += self.LineHeight
	}
}

func NewTextBlockElement(report *report, x float64, y float64, renderY float64, width float64, height float64, textOffsetY float64, lines []TextLine, lineHeight float64, renderElementType RenderElementType, style Style) *TextBlockElement {
	textBlockElement := TextBlockElement{}
	textBlockElement.initTextBlockElement(report, x, y, renderY, width, height, textOffsetY, lines, lineHeight, renderElementType, style)
	return &textBlockElement
}

type TextLine struct {
	Text  string
	Width float64
	Style Style
	Link  string
}

func (self *TextLine) init(text string, width float64, style Style, link string) {
	self.Text = text
	self.Width = width
	self.Style = style
	self.Link = link
}

func (self *TextLine) renderPDF(x float64, y float64, lastLine bool, pdfDoc *FPDFRB) {
	renderY := y + self.Style.base().FontSize*0.8
	lineWidth := 0.0
	offsetX := 0.0
	if self.Style.base().HorizontalAlignment == HorizontalAlignmentJustify {
		if lastLine {
			pdfDoc.Fpdf.SetFont(self.Style.base().Font, self.Style.base().FontStyle, self.Style.base().FontSize)
			pdfDoc.Fpdf.Text(x, renderY, pdfDoc.Tr(self.Text))
		} else {
			words := strings.Split(self.Text, "")
			wordWidth := make([]float64, 0)
			totalWordWidth := 0.0
			for _, word := range words {
				tmpWidth := pdfDoc.Fpdf.GetStringWidth(word)
				wordWidth = append(wordWidth, tmpWidth)
				totalWordWidth += tmpWidth
			}
			countSpaces := len(words) - 1
			wordSpacing := 0.0
			if countSpaces > 0 {
				wordSpacing = ((self.Width - totalWordWidth) / float64(countSpaces))
			}
			wordX := x
			pdfDoc.Fpdf.SetFont(self.Style.base().Font, self.Style.base().FontStyle, self.Style.base().FontSize)
			for i, word := range words {
				pdfDoc.Fpdf.Text(wordX, renderY, pdfDoc.Tr(word))
				wordX += wordWidth[i] + wordSpacing
			}

			textWidth := 0.0
			if self.Style.base().Underline {
				if len(words) == 1 {
					textWidth = wordWidth[0]
				} else {
					textWidth = self.Width
				}

				underlinePosition := x    // pdfDoc.Fpdf.CurrentFont["up"]
				underlineThickness := 1.0 // pdfDoc.Fpdf.CurrentFont["ut"]
				renderY += -underlinePosition / 1000.0 * self.Style.base().FontSize
				underlineWidth := underlineThickness / 1000.0 * self.Style.base().FontSize
				pdfDoc.Fpdf.SetLineWidth(underlineWidth)
				pdfDoc.Fpdf.Line(x, renderY, x+textWidth, renderY)
			}

			if len(words) > 1 {
				lineWidth = self.Width
			} else if len(words) > 0 {
				lineWidth = wordWidth[0]
			}
		}
	} else {
		if self.Style.base().HorizontalAlignment != HorizontalAlignmentLeft {
			lineWidth = pdfDoc.Fpdf.GetStringWidth(self.Text)
			space := self.Width - lineWidth
			if self.Style.base().HorizontalAlignment == HorizontalAlignmentCenter {
				offsetX = (space / 2)
			} else if self.Style.base().HorizontalAlignment == HorizontalAlignmentRight {
				offsetX = space
			}
		}
		self.Text = strings.Replace(self.Text, `\n`, "\n", -1)
		pdfDoc.Fpdf.Text(x+offsetX, renderY, pdfDoc.Tr(self.Text))
	}

	if self.Style.base().Strikethrough {
		if lineWidth == 0.0 {
			lineWidth = pdfDoc.Fpdf.GetStringWidth(self.Text)
		}
		// use underline thickness
		strikethroughThickness := 1.0 // pdfDoc.Fpdf.CurrentFont["ut"]
		renderY = y + self.Style.base().FontSize*0.5
		strikethroughWidth := strikethroughThickness / 1000.0 * self.Style.base().FontSize
		pdfDoc.Fpdf.SetLineWidth(strikethroughWidth)
		pdfDoc.Fpdf.Line(x+offsetX, renderY, x+offsetX+lineWidth, renderY)
	}

	if self.Link != "" {
		if lineWidth == 0.0 {
			lineWidth = pdfDoc.Fpdf.GetStringWidth(self.Text)
		}
		linkID := pdfDoc.Fpdf.AddLink()
		pdfDoc.Fpdf.Link(x+offsetX, y, lineWidth, self.Style.base().FontSize, linkID)
	}
}

func NewTextLine(text string, width float64, style Style, link string) TextLine {
	textLine := TextLine{}
	textLine.init(text, width, style, link)
	return textLine
}

type TableTextElement struct {
	TextElement
}

func (self *TableTextElement) init(report *report, data map[string]interface{}) {
	self.TextElement.init(report, data)
}

func NewTableTextElement(report *report, data map[string]interface{}) *TableTextElement {
	tableTextElement := TableTextElement{}
	tableTextElement.init(report, data)
	return &tableTextElement
}

type TableImageElement struct {
	ImageElement
}

func (self *TableImageElement) init(report *report, data map[string]interface{}) {
	self.ImageElement.init(report, data)
}

func NewTableImageElement(report *report, data map[string]interface{}) *TableImageElement {
	tableImageElement := TableImageElement{}
	tableImageElement.init(report, data)
	return &tableImageElement
}

type TableRow struct {
	ColumnData               []DocElementBaseProvider
	Height                   float64
	AlwaysPrintOnSamePage    bool
	TableBand                *TableBandElement
	RenderElements           []DocElementBaseProvider
	BackgroundColor          Color
	AlternateBackgroundColor *Color
	GroupExpression          string
	PrintIfResult            bool
	PrevRow                  *TableRow
	NextRow                  *TableRow
}

func (self *TableRow) init(report *report, tableBand *TableBandElement, columns []int, ctx Context, prevRow *TableRow) {
	if len(columns) > len(tableBand.ColumnData) {
		return
	}
	self.ColumnData = make([]DocElementBaseProvider, 0)
	for _, column := range columns {
		columnContent := GetStringValue(tableBand.ColumnData[column].(map[string]interface{}), "content")

		parameterName := stripParameterName(columnContent)
		columnDataParameter := ctx.getParameter(parameterName, nil)

		if columnDataParameter != nil && columnDataParameter.(map[string]interface{})[parameterName].(Parameter).Type == ParameterTypeImage {
			columnElement := NewTableImageElement(report, tableBand.ColumnData[column].(map[string]interface{}))
			columnElement.TableElement = true
			self.ColumnData = append(self.ColumnData, columnElement)
		} else {
			columnElement := NewTableTextElement(report, tableBand.ColumnData[column].(map[string]interface{}))
			columnElement.TableElement = true
			self.ColumnData = append(self.ColumnData, columnElement)

			if tableBand.ColumnData[column].(map[string]interface{})["simple_array"] != false {
				// in case value of column is a simple array parameter we create multiple columns,
				// one for each array entry of parameter data
				isSimpleArray := false
				if columnElement.Content != "" && !columnElement.Eval && isParameterName(columnElement.Content) {

					if columnDataParameter != nil && columnDataParameter.(map[string]interface{})[parameterName].(Parameter).Type == ParameterTypeSimpleArray {
						isSimpleArray = true
						columns, _ := ctx.getData(columnDataParameter.(map[string]interface{})[parameterName].(Parameter).Name, nil)
						columnValues := columns.(map[string]interface{})
						for idx, columnValue := range columnValues {
							formattedVal := ctx.getFormattedValue(columnValue, columnDataParameter.(map[string]interface{})[parameterName].(Parameter), 0, "", true)
							if idx == "0" {
								columnElement.Content = formattedVal
							} else {
								columnElement = NewTableTextElement(report, tableBand.ColumnData[column].(map[string]interface{}))
								columnElement.Content = formattedVal
								self.ColumnData = append(self.ColumnData, columnElement)
							}
						}
					}
				}
				// store info if column content is a simple array parameter to
				// avoid checks for the next rows
				tableBand.ColumnData[column].(map[string]interface{})["simple_array"] = isSimpleArray
			}
		}
	}

	self.Height = 0
	self.AlwaysPrintOnSamePage = true
	self.TableBand = tableBand
	self.RenderElements = make([]DocElementBaseProvider, 0)
	self.BackgroundColor = tableBand.BackgroundColor
	self.AlternateBackgroundColor = &tableBand.BackgroundColor
	if tableBand.BandType == BandTypeContent && !tableBand.AlternateBackgroundColor.Transparent {
		self.AlternateBackgroundColor = tableBand.AlternateBackgroundColor
	}
	self.GroupExpression = ""
	self.PrintIfResult = true
	self.PrevRow = prevRow
	self.NextRow = nil
	if prevRow != nil {
		prevRow.NextRow = self
	}

}

func (self *TableRow) isPrinted(ctx Context) bool {
	printed := self.PrintIfResult
	if printed && self.TableBand.GroupExpression != "" {
		if self.TableBand.BeforeGroup {
			printed = (self.PrevRow == nil || self.GroupExpression != self.PrevRow.GroupExpression)
		} else {
			printed = (self.NextRow == nil || self.GroupExpression != self.NextRow.GroupExpression)
		}
	}
	return printed
}

func (self *TableRow) prepare(ctx Context, pdfDoc *FPDFRB, rowIndex int, onlyVerify bool) {
	if onlyVerify {
		for _, columnElement := range self.ColumnData {
			columnElement.prepare(ctx, pdfDoc, true)
		}
	} else {
		if self.TableBand != nil {
			if self.TableBand.GroupExpression != "" {
				self.GroupExpression = cast.ToString(ctx.evaluateExpression(self.TableBand.GroupExpression, self.TableBand.ID, "group_expression"))
			}

			if self.TableBand.PrintIf != "" {
				self.PrintIfResult = cast.ToBool(ctx.evaluateExpression(self.TableBand.PrintIf, self.TableBand.ID, "print_if"))
			}
		}

		if self.TableBand != nil {
			heights := []float64{self.TableBand.Height}
			for _, columnElement := range self.ColumnData {
				columnElement.prepare(ctx, pdfDoc, false)
				heights = append(heights, columnElement.base().TotalHeight)
				backgroundColor := self.BackgroundColor
				if rowIndex != -1 && rowIndex%2 == 1 {
					backgroundColor = *self.AlternateBackgroundColor
				}
				if backgroundColor.Transparent == false {
					if element, ok := columnElement.(*TableTextElement); ok {
						if element.UsedStyle.BackgroundColor.Transparent {
							element.UsedStyle.BackgroundColor = backgroundColor
						}
					}
				}
			}

			self.Height = maxFloatSlice(heights)
		}

		for _, columnElement := range self.ColumnData {
			columnElement.setHeight(self.Height)
		}
	}
}

func (self *TableRow) createRenderElements(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) {
	for _, columnElement := range self.ColumnData {
		renderElement, _ := columnElement.getNextRenderElement(offsetY, containerHeight, ctx, pdfDoc)
		if renderElement == nil {
			log.Println(Error{Message: fmt.Sprintf("TableRow.create_renderElements failed - failed to create column renderElement %v", columnElement.base().ID)})
			continue
		}
		self.RenderElements = append(self.RenderElements, renderElement)
	}
}

func (self *TableRow) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB) {
	x := containerOffsetX
	for _, renderElement := range self.RenderElements {
		renderElement.renderPDF(x, containerOffsetY, pdfDoc)
		x += renderElement.base().Width
	}
}

func (self *TableRow) renderSpreadsheet(row int, col int, ctx Context, renderer Renderer) int {
	for _, columnElement := range self.ColumnData {
		columnElement.base().renderSpreadsheet(row, col, ctx, renderer)
		col++
	}
	return row + 1
}

func (self *TableRow) verify(ctx Context) {
	// for _, columnElement := range self.ColumnData {
	// 	columnElement.base().verify(ctx)
	// }
}

func (self *TableRow) getWidth() float64 {
	width := 0.0
	for _, columnElement := range self.ColumnData {
		width += columnElement.base().Width
	}
	return width
}

func (self *TableRow) getRenderY() float64 {
	if len(self.RenderElements) > 0 {
		return self.RenderElements[0].base().RenderY
	}
	return 0
}

func NewTableRow(report *report, tableBand *TableBandElement, columns []int, ctx Context, prevRow *TableRow) *TableRow {
	tableRow := TableRow{}
	tableRow.init(report, tableBand, columns, ctx, prevRow)
	return &tableRow
}

type TableBlockElement struct {
	DocElementBase
	Table    DocElementBaseProvider
	Rows     []*TableRow
	Complete bool
}

func (self *TableBlockElement) initTableBlockElement(report *report, x float64, width float64, renderY float64, table DocElementBaseProvider) {
	self.DocElementBase.init(report, map[string]interface{}{"y": 0})
	self.Report = report
	self.X = x
	self.Width = width
	self.RenderY = renderY
	self.RenderBottom = renderY
	self.Table = table
	self.Rows = make([]*TableRow, 0)
	self.Complete = false
}

func (self *TableBlockElement) addRows(rows []*TableRow, allowSplit bool, availableHeight float64, offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) int {
	rowsAdded := 0
	if !self.Complete {
		if !allowSplit {
			height := 0.0
			for _, row := range rows {
				height += row.Height
			}
			if height <= availableHeight {
				for _, row := range rows {
					row.createRenderElements(offsetY, containerHeight, ctx, pdfDoc)
					self.Rows = append(self.Rows, row)
				}
				rowsAdded = len(rows)
				availableHeight -= height
				self.Height += height
				self.RenderBottom += height
			} else {
				self.Complete = true
			}
		} else {
			for _, row := range rows {
				if row.Height <= availableHeight {
					row.createRenderElements(offsetY, containerHeight, ctx, pdfDoc)
					self.Rows = append(self.Rows, row)
					rowsAdded += 1
					availableHeight -= row.Height
					self.Height += row.Height
					self.RenderBottom += row.Height
				} else {
					self.Complete = true
					break
				}
			}
		}
	}
	return rowsAdded
}

func (self *TableBlockElement) isEmpty() bool {
	return (len(self.Rows) == 0)
}

func (self *TableBlockElement) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB) {
	y := containerOffsetY
	for _, row := range self.Rows {
		row.renderPDF(containerOffsetX+self.X, y, pdfDoc)
		y += row.Height
	}

	if len(self.Rows) > 0 && self.Table.base().Border != BorderNone {
		pdfDoc.Fpdf.SetDrawColor(self.Table.base().BorderColor.R, self.Table.base().BorderColor.G, self.Table.base().BorderColor.B)
		pdfDoc.Fpdf.SetLineWidth(self.Table.base().BorderStyle.BorderWidth)
		halfBorderWidth := self.Table.base().BorderStyle.BorderWidth / 2
		x1 := containerOffsetX + self.X
		x2 := x1 + self.Rows[0].getWidth()
		x1 += halfBorderWidth
		x2 -= halfBorderWidth
		y1 := self.Rows[0].getRenderY() + containerOffsetY
		y2 := y1 + (y - containerOffsetY)
		if self.Table.base().Border == BorderGrid || self.Table.base().Border == BorderFrameRow || self.Table.base().Border == BorderFrame {
			pdfDoc.Fpdf.Line(x1, y1, x1, y2)
			pdfDoc.Fpdf.Line(x2, y1, x2, y2)
		}
		y = y1
		pdfDoc.Fpdf.Line(x1, y1, x2, y1)
		if self.Table.base().Border != BorderFrame {

			for _, row := range self.Rows[:len(self.Rows)-1] {
				y += row.Height
				pdfDoc.Fpdf.Line(x1, y, x2, y)
			}
		}
		pdfDoc.Fpdf.Line(x1, y2, x2, y2)
		if self.Table.base().Border == BorderGrid {
			columns := self.Rows[0].ColumnData
			// add half borderWidth so border is drawn inside right column and can be aligned with
			// borders of other elements outside the table
			x := x1
			for _, column := range columns[:len(columns)-1] {
				x += column.base().Width
				pdfDoc.Fpdf.Line(x, y1, x, y2)
			}
		}
	}
}

func NewTableBlockElement(report *report, x float64, width float64, renderY float64, table DocElementBaseProvider) *TableBlockElement {
	tableBlockElement := TableBlockElement{}
	tableBlockElement.initTableBlockElement(report, x, width, renderY, table)
	return &tableBlockElement
}

type TableElement struct {
	DocElement
	DataSource             string
	Columns                []int
	header                 *TableBandElement
	PrintHeader            bool
	Footer                 *TableBandElement
	PrintFooter            bool
	ContentRows            []*TableBandElement
	BorderWidth            float64
	RemoveEmptyElement     bool
	SpreadsheetAddEmptyRow bool
	DataSourceparameter    *Parameter
	RowParameters          map[string]interface{}
	Rows                   []interface{}
	PreparedRows           []*TableRow
	RowCount               int
	RowIndex               int
	PrevContentRows        []*TableRow
}

func (self *TableElement) init(report *report, data map[string]interface{}) {
	self.DocElement.init(report, data)
	self.DataSource = GetStringValue(data, "dataSource")
	self.Columns = pyRange(0, GetIntValue(data, "columns"), 1)
	header := GetBoolValue(data, "header")
	footer := GetBoolValue(data, "footer")
	self.header = nil
	if header {
		self.header = NewTableBandElement(data["headerData"].(map[string]interface{}), BandTypeHeader, false)
	}
	self.ContentRows = make([]*TableBandElement, 0)
	contentDataRows := data["contentDataRows"].([]interface{})

	mainContentCreated := false
	for _, contentDataRow := range contentDataRows {
		bandElement := NewTableBandElement(contentDataRow.(map[string]interface{}), BandTypeContent, !mainContentCreated)
		if !mainContentCreated && bandElement.GroupExpression == "" {
			mainContentCreated = true
		}
		self.ContentRows = append(self.ContentRows, bandElement)
	}
	self.Footer = nil
	if footer {
		self.Footer = NewTableBandElement(data["footerData"].(map[string]interface{}), BandTypeFooter, false)
	}
	if self.header != nil {
		self.PrintHeader = true
	}
	if self.Footer != nil {
		self.PrintFooter = true
	}
	self.Border = GetBorder(GetStringValue(data, "border"))
	self.BorderColor = NewColor(GetStringValue(data, "borderColor"))
	self.BorderWidth = GetFloatValue(data, "borderWidth")
	self.PrintIf = GetStringValue(data, "printIf")
	self.RemoveEmptyElement = GetBoolValue(data, "removeEmptyElement")
	self.SpreadsheetHide = GetBoolValue(data, "spreadsheet_hide")
	self.SpreadsheetColumn = StringPointer(cast.ToString(GetIntValue(data, "spreadsheet_column")))
	self.SpreadsheetAddEmptyRow = GetBoolValue(data, "spreadsheet_addEmptyRow")
	self.DataSourceparameter = nil
	self.RowParameters = map[string]interface{}{}
	self.Rows = make([]interface{}, 0)
	self.RowCount = 0
	self.RowIndex = -1
	self.PreparedRows = make([]*TableRow, 0)
	self.PrevContentRows = make([]*TableRow, len(self.ContentRows))
	self.Width = 0
	if self.header != nil {
		self.Height += self.header.Height
	}
	if self.Footer != nil {
		self.Height += self.Footer.Height
	}
	if len(self.ContentRows) > 0 {
		for _, contentRow := range self.ContentRows {
			self.Height += contentRow.Height
		}
		for _, column := range self.ContentRows[0].ColumnData {
			self.Width += GetFloatValue(column.(map[string]interface{}), "width")
		}
	}
	self.Bottom = self.Y + self.Height
	self.FirstRenderElement = true
}

func (self *TableElement) prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool) {
	if self.header != nil {
		for columnIdx, column := range self.header.ColumnData {
			if GetStringValue(column.(map[string]interface{}), "printIf") != "" {
				printed := cast.ToBool(ctx.evaluateExpression(GetStringValue(column.(map[string]interface{}), "printIf"), GetIntValue(column.(map[string]interface{}), "id"), "printIf"))
				if !printed {
					self.Columns = append(self.Columns[:columnIdx], self.Columns[columnIdx+1:]...)
				}
			}
		}
	}
	parameterName := stripParameterName(self.DataSource)
	self.DataSourceparameter = nil
	if parameterName != "" {
		parameters := ctx.getParameter(parameterName, nil)
		param := parameters.(map[string]interface{})
		parameter := param[parameterName].(Parameter)
		self.DataSourceparameter = &parameter
		if self.DataSourceparameter == nil {
			log.Println(Error{Message: "errorMsgMissingparameter", ObjectID: self.ID, Field: "data_source"})
		}
		if self.DataSourceparameter.Type != ParameterTypeArray {
			log.Println(Error{Message: "errorMsgInvalidDataSourceparameter", ObjectID: self.ID, Field: "data_source"})
		}
		for _, rowparameter := range self.DataSourceparameter.Children {
			self.RowParameters[rowparameter.(Parameter).Name] = rowparameter.(Parameter)
		}

		rows, parameterExists := ctx.getData(self.DataSourceparameter.Name, nil)
		if parameterExists == false {
			log.Println(Error{Message: "errorMsgMissingData", ObjectID: self.ID, Field: "data_source"})
		}

		if rows, ok := rows.([]interface{}); ok {
			self.Rows = rows
		} else {
			log.Println(Error{Message: "errorMsgInvalidDataSource", ObjectID: self.ID, Field: "data_source"})
		}
	} else {
		// there is no data source parameter so we create a static table (faked by one empty data row)
		self.Rows = make([]interface{}, 1)
		self.Rows[0] = map[string]interface{}{}
	}

	self.RowCount = len(self.Rows)
	self.RowIndex = 0

	if onlyVerify {
		if self.PrintHeader {
			tableRow := NewTableRow(self.Report, self.header, self.Columns, ctx, nil)
			tableRow.prepare(ctx, nil, 0, true)
		}
		for self.RowIndex < self.RowCount {
			// push data context of current row so values of current row can be accessed
			ctx.pushContext(self.RowParameters, self.Rows[self.RowIndex].(map[string]interface{}))
			for _, contentRow := range self.ContentRows {
				tableRow := NewTableRow(self.Report, contentRow, self.Columns, ctx, nil)
				tableRow.prepare(ctx, nil, self.RowIndex, true)
			}
			ctx.popContext()
			self.RowIndex++
		}
		if self.PrintFooter {
			tableRow := NewTableRow(self.Report, self.Footer, self.Columns, ctx, nil)
			tableRow.prepare(ctx, nil, 0, true)
		}
	}

}

func (self *TableElement) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	self.RenderY = offsetY
	self.RenderBottom = self.RenderY
	if self.isRenderingComplete() {
		self.RenderingComplete = true
		return nil, true
	}
	renderElement := NewTableBlockElement(self.Report, self.X, self.Width, offsetY, self)

	// batch size can be anything >= 3 because each row needs previous and next row to evaluate
	// group expression (in case it is set), the batch size defines the number of table rows
	// which will be prepared before they are rendered
	batchSize := 10
	remainingBatchSize := batchSize

	// add header in case it is not already available in prepared rows (from previous page)
	if self.PrintHeader && (len(self.PreparedRows) == 0 || self.PreparedRows[0].TableBand.BandType != BandTypeHeader) {
		tableRow := NewTableRow(self.Report, self.header, self.Columns, ctx, nil)
		tableRow.prepare(ctx, pdfDoc, 0, false)
		self.PreparedRows = append([]*TableRow{tableRow}, self.PreparedRows...)
		if !self.header.RepeatHeader {
			self.PrintHeader = false
		}
	}

	for self.RowIndex < self.RowCount {
		// push data context of current row so values of current row can be accessed
		ctx.pushContext(self.RowParameters, self.Rows[self.RowIndex].(map[string]interface{}))
		for i, contentRow := range self.ContentRows {
			var prevRow *TableRow
			if self.PrevContentRows[i] != nil {
				prevRow = self.PrevContentRows[i]
			}
			tableRow := NewTableRow(self.Report, contentRow, self.Columns, ctx, prevRow)
			tableRow.prepare(ctx, pdfDoc, self.RowIndex, false)
			self.PreparedRows = append(self.PreparedRows, tableRow)
			self.PrevContentRows[i] = tableRow
		}
		ctx.popContext()
		remainingBatchSize--
		self.RowIndex++
		if remainingBatchSize == 0 {
			remainingBatchSize = batchSize
			if self.RowIndex < self.RowCount || self.PrintFooter {
				self.updateRenderElement(renderElement, offsetY, containerHeight, ctx, pdfDoc)
				if renderElement.Complete {
					break
				}
			}
		}
	}

	if self.RowIndex >= self.RowCount && self.PrintFooter {
		tableRow := NewTableRow(self.Report, self.Footer, self.Columns, ctx, nil)
		tableRow.prepare(ctx, pdfDoc, 0, false)
		self.PreparedRows = append(self.PreparedRows, tableRow)
		self.PrintFooter = false
	}

	self.updateRenderElement(renderElement, offsetY, containerHeight, ctx, pdfDoc)

	if self.isRenderingComplete() {
		self.RenderingComplete = true
	}

	if renderElement.isEmpty() {
		return nil, self.RenderingComplete
	}

	self.RenderBottom = renderElement.RenderBottom
	self.FirstRenderElement = false
	return renderElement, self.RenderingComplete
}

func (self *TableElement) updateRenderElement(renderElement DocElementBaseProvider, offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) {
	availableHeight := containerHeight - offsetY
	filteredRows := make([]*TableRow, 0)
	rowsForNextUpdate := make([]*TableRow, 0)
	allRowsProcessed := (self.RowIndex >= self.RowCount)
	for _, preparedRow := range self.PreparedRows {
		if preparedRow.TableBand != nil && preparedRow.TableBand.BandType == BandTypeContent {
			if preparedRow.NextRow != nil || allRowsProcessed {
				if preparedRow.isPrinted(ctx) {
					filteredRows = append(filteredRows, preparedRow)
				}
			} else {
				rowsForNextUpdate = append(rowsForNextUpdate, preparedRow)
			}
		} else {
			filteredRows = append(filteredRows, preparedRow)
		}
	}

	for renderElement.base().Complete == false && len(filteredRows) > 0 {
		addRowCount := 1
		if len(filteredRows) >= 2 && (filteredRows[0].TableBand != nil && filteredRows[0].TableBand.BandType == BandTypeHeader) || (filteredRows[len(filteredRows)-1].TableBand != nil && filteredRows[len(filteredRows)-1].TableBand.BandType == BandTypeFooter) {
			// make sure header row is not printed alone on a page
			addRowCount = 2
		}
		// allow splitting multiple rows (header + content or footer) in case we are already at top
		// of the container and there is not enough space for both rows
		allowSplit := (offsetY == 0)
		height := availableHeight - renderElement.base().Height

		if addRowCount > len(filteredRows) {
			addRowCount = len(filteredRows)
		}
		rowsAdded := renderElement.addRows(filteredRows[:addRowCount], allowSplit, height, offsetY, containerHeight, ctx, pdfDoc)
		if rowsAdded == 0 {
			break
		}
		filteredRows = filteredRows[rowsAdded:]
		self.FirstRenderElement = false
	}

	self.PreparedRows = filteredRows
	for _, row := range rowsForNextUpdate {
		self.PreparedRows = append(self.PreparedRows, row)
	}
}

func (self *TableElement) isRenderingComplete() bool {
	return ((!self.PrintHeader || (self.header != nil && self.header.RepeatHeader)) && !self.PrintFooter && self.RowIndex >= self.RowCount && len(self.PreparedRows) == 0)
}

func (self *TableElement) renderSpreadsheet() {
	// if self.spreadsheet_column:
	// 	col = self.spreadsheet_column - 1

	// if self.printHeader:
	// 	tableRow = TableRow(self.report, self.header, self.columns, ctx=ctx)
	// 	tableRow.prepare(ctx, pdfDoc=nil)
	// 	if tableRow.isPrinted(ctx):
	// 		row = tableRow.render_spreadsheet(row, col, ctx, renderer)

	// data_context_added = false
	// while self.rowIndex < self.rowCount:
	// 	// push data context of current row so values of current row can be accessed
	// 	if data_context_added:
	// 		ctx.popContext()
	// 	else:
	// 		data_context_added = true
	// 	ctx.pushcontext(self.rowparameters, self.rows[self.rowIndex])

	// 	for i, contentRow in enumerate(self.contentRows):
	// 		tableRow = TableRow(
	// self.report, contentRow, self.columns, ctx=ctx, PrevRow=self.P[i])
	// 		tableRow.prepare(ctx, pdfDoc=nil, rowIndex=self.rowIndex)
	// 		// render rows from previous preparation because we need next row set (used for group_expression)
	// if self.P[i] is not nil and self.P[i].isPrinted(ctx):
	// 	row = self.P[i].render_spreadsheet(row, col, ctx, renderer)

	// self.P[i] = tableRow
	// 	self.rowIndex += 1
	// if data_context_added:
	// 	ctx.popContext()

	// for i, prev_contentRow in enumerate(self.P):
	// 	if self.P[i] is not nil and self.P[i].isPrinted(ctx):
	// 		row = self.P[i].render_spreadsheet(row, col, ctx, renderer)

	// if self.printFooter:
	// 	tableRow = TableRow(self.report, self.footer, self.columns, ctx=ctx)
	// 	tableRow.prepare(ctx, pdfDoc=nil)
	// 	if tableRow.isPrinted(ctx):
	// 		row = tableRow.render_spreadsheet(row, col, ctx, renderer)

	// if self.spreadsheet_add_empty_row:
	// 	row += 1
	// return row, col + self.get_column_count()
}

func (self *TableElement) getColumnCount() int {
	return len(self.Columns)
}

func NewTableElement(report *report, data map[string]interface{}) *TableElement {
	tableElement := TableElement{}
	tableElement.init(report, data)
	return &tableElement
}

type TableBandElement struct {
	ID                       int
	Height                   float64
	BandType                 BandType
	RepeatHeader             bool
	BackgroundColor          Color
	AlternateBackgroundColor *Color
	ColumnData               []interface{}
	GroupExpression          string
	PrintIf                  string
	BeforeGroup              bool
}

func (self *TableBandElement) init(data map[string]interface{}, bandType BandType, beforeGroup bool) {
	self.ID = GetIntValue(data, "id")
	self.Height = float64(GetIntValue(data, "height"))
	self.BandType = bandType
	if bandType == BandTypeHeader {
		self.RepeatHeader = GetBoolValue(data, "repeatHeader")
	}
	self.BackgroundColor = NewColor(GetStringValue(data, "backgroundColor"))
	if bandType == BandTypeContent {
		self.AlternateBackgroundColor = ColorPointer(NewColor(GetStringValue(data, "alternateBackgroundColor")))
	} else {
		self.AlternateBackgroundColor = nil
	}
	self.ColumnData = data["columnData"].([]interface{})
	self.GroupExpression = GetStringValue(data, "groupExpression")
	self.PrintIf = GetStringValue(data, "printIf")
	self.BeforeGroup = beforeGroup
}

func NewTableBandElement(data map[string]interface{}, bandType BandType, beforeGroup bool) *TableBandElement {
	tableBandElement := TableBandElement{}
	tableBandElement.init(data, bandType, beforeGroup)
	return &tableBandElement
}

type FrameBlockElement struct {
	DocElementBase
	RemoveEmptyElement float64
	Elements           []DocElementBaseProvider
	RenderElementType  RenderElementType
	Complete           bool
}

func (self *FrameBlockElement) initFrameBlock(report *report, frame DocElementBaseProvider, renderY float64) {
	self.DocElementBase.init(report, map[string]interface{}{"y": 0})
	self.Report = report
	self.X = frame.base().X
	self.Width = frame.base().Width
	self.BorderStyle = frame.base().BorderStyle
	self.BackgroundColor = frame.base().BackgroundColor
	self.RenderY = renderY
	self.RenderBottom = renderY
	self.Height = 0.0
	self.Elements = make([]DocElementBaseProvider, 0)
	self.RenderElementType = RenderElementTypeNone
	self.Complete = false
}

func (self *FrameBlockElement) addElements(container *Frame, renderElementType RenderElementType, height float64) {
	self.Elements = container.RenderElements
	self.RenderElementType = renderElementType
	self.RenderBottom += height
}

func (self *FrameBlockElement) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB) {
	x := self.X + containerOffsetX
	y := self.RenderY + containerOffsetY
	height := self.RenderBottom - self.RenderY

	contentX := x
	contentWidth := self.Width
	contentY := y
	contentHeight := height

	if self.BorderStyle.BorderLeft {
		contentX += self.BorderStyle.BorderWidth
		contentWidth -= self.BorderStyle.BorderWidth
	}
	if self.BorderStyle.BorderRight {
		contentWidth -= self.BorderStyle.BorderWidth
	}
	if self.BorderStyle.BorderTop && (self.RenderElementType == RenderElementTypeFirst || self.RenderElementType == RenderElementTypeComplete) {
		contentY += self.BorderStyle.BorderWidth
		contentHeight -= self.BorderStyle.BorderWidth
	}
	if self.BorderStyle.BorderBottom && (self.RenderElementType == RenderElementTypeLast || self.RenderElementType == RenderElementTypeComplete) {
		contentHeight -= self.BorderStyle.BorderWidth
	}

	if self.BackgroundColor.Transparent == false && self.BackgroundColor.ColorCode != "" {
		pdfDoc.Fpdf.SetFillColor(self.BackgroundColor.R, self.BackgroundColor.G, self.BackgroundColor.B)
		pdfDoc.Fpdf.Rect(contentX, contentY, contentWidth, contentHeight, "F")
	}

	renderY := y
	if self.BorderStyle.BorderTop && (self.RenderElementType == RenderElementTypeFirst || self.RenderElementType == RenderElementTypeComplete) {
		renderY += self.BorderStyle.BorderWidth
	}

	for _, element := range self.Elements {
		element.renderPDF(contentX, contentY, pdfDoc)
	}

	if self.BorderStyle.BorderLeft || self.BorderStyle.BorderTop || self.BorderStyle.BorderRight || self.BorderStyle.BorderBottom {
		drawBorder(self.X+containerOffsetX, y, self.Width, height, self.RenderElementType, &self.BorderStyle, pdfDoc)
	}
}

func NewFrameBlockElement(report *report, frame DocElementBaseProvider, renderY float64) *FrameBlockElement {
	frameBlockElement := FrameBlockElement{}
	frameBlockElement.initFrameBlock(report, frame, renderY)
	return &frameBlockElement
}

type FrameElement struct {
	DocElement
	ShrinkToContentHeight     bool
	SpreadsheetColumn         int
	NextPageRenderingComplete bool
	PrevPageContentHeight     float64
	RenderElementType         RenderElementType
	Container                 *Frame
}

func (self *FrameElement) initFrameElement(report *report, data map[string]interface{}, containers *containers) {
	self.DocElement.init(report, data)
	self.BackgroundColor = NewColor(GetStringValue(data, "backgroundColor"))
	self.BorderStyle = NewBorderStyle(data, "")
	self.PrintIf = GetStringValue(data, "printIf")
	self.RemoveEmptyElement = GetBoolValue(data, "removeEmptyElement")
	self.ShrinkToContentHeight = GetBoolValue(data, "shrinkToContentHeight")
	self.SpreadsheetHide = GetBoolValue(data, "spreadsheet_hide")
	self.SpreadsheetColumn = GetIntValue(data, "spreadsheet_column")
	self.SpreadsheetAddEmptyRow = GetBoolValue(data, "spreadsheet_addEmptyRow")

	// renderingComplete status for next page, in case rendering was not started on first page.
	self.NextPageRenderingComplete = false
	// container content height of previous page, in case rendering was not started on first page
	self.PrevPageContentHeight = 0

	self.RenderElementType = RenderElementTypeNone
	self.Container = NewFrame(self.Width, self.Height, cast.ToString(GetIntValue(data, "linkedContainerId")), containers, report)
}

func (self *FrameElement) getUsedHeight() float64 {
	height := self.Container.GetRenderElementsBottom()
	if self.BorderStyle.BorderTop && self.RenderElementType == RenderElementTypeNone {
		height += self.BorderStyle.BorderWidth
	}
	if self.BorderStyle.BorderBottom {
		height += self.BorderStyle.BorderWidth
	}
	if self.RenderElementType == RenderElementTypeNone && self.ShrinkToContentHeight == false {
		height = max(self.Height, height)
	}
	return height
}

func (self *FrameElement) prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool) {
	self.Container.prepare(ctx, pdfDoc, onlyVerify)
	self.NextPageRenderingComplete = false
	self.PrevPageContentHeight = 0
	self.RenderElementType = RenderElementTypeNone
}

func (self *FrameElement) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	self.RenderY = offsetY
	contentHeight := containerHeight
	renderElement := NewFrameBlockElement(self.Report, self, offsetY)

	if self.BorderStyle.BorderTop && self.RenderElementType == RenderElementTypeNone {
		contentHeight -= self.BorderStyle.BorderWidth
	}
	if self.BorderStyle.BorderBottom {
		// this is not 100% correct because bottom border is only applied if frame fits
		// on current page. this should be negligible because the border is usually only a few pixels
		// && most of the time the frame fits on one page.
		// to get the exact height in advance would be quite hard && is probably not worth the effort ...
		contentHeight -= self.BorderStyle.BorderWidth
	}

	if self.FirstRenderElement {
		availableHeight := containerHeight - offsetY
		self.FirstRenderElement = false
		renderingComplete := self.Container.createRenderElements(contentHeight, ctx, pdfDoc)

		neededHeight := self.getUsedHeight()

		if renderingComplete && neededHeight <= availableHeight {
			// rendering is complete && all elements of frame fit on current page
			self.RenderingComplete = true
			self.RenderBottom = offsetY + neededHeight
			self.RenderElementType = RenderElementTypeComplete
			renderElement.addElements(self.Container, self.RenderElementType, neededHeight)
			return renderElement, true
		} else {
			if offsetY == 0 {
				// rendering of frame elements does not fit on current page but
				// we are already at the top of the page -> start rendering && continue on next page
				self.RenderBottom = offsetY + availableHeight
				self.RenderElementType = RenderElementTypeFirst
				renderElement.addElements(self.Container, self.RenderElementType, availableHeight)
				return renderElement, false
			} else {
				// rendering of frame elements does not fit on current page -> start rendering on next page
				self.NextPageRenderingComplete = renderingComplete
				self.PrevPageContentHeight = contentHeight
				return nil, false
			}
		}
	}

	if self.RenderElementType == RenderElementTypeNone {
		// render elements were already created on first call to getNextRenderElement
		// but elements did not fit on first page

		if contentHeight == self.PrevPageContentHeight {
			// we don"t have to create any render elements here because we can use
			// the previously created elements

			self.RenderingComplete = self.NextPageRenderingComplete
		} else {
			// we cannot use previously created render elements because container height is different
			// on current page. this should be very unlikely but could happen when the frame should be
			// printed on the first page && header/footer are not shown on first page, i.e. the following
			// pages have a different content band size than the first page.

			self.Container.prepare(ctx, pdfDoc, false)
			self.RenderingComplete = self.Container.createRenderElements(contentHeight, ctx, pdfDoc)
		}
	} else {
		self.RenderingComplete = self.Container.createRenderElements(contentHeight, ctx, pdfDoc)
	}
	self.RenderBottom = offsetY + self.getUsedHeight()

	if self.RenderingComplete == false {
		// use whole size of container if frame is not rendered completely
		self.RenderBottom = offsetY + containerHeight

		if self.RenderElementType == RenderElementTypeNone {
			self.RenderElementType = RenderElementTypeFirst
		} else {
			self.RenderElementType = RenderElementTypeBetween
		}
	} else {
		if self.RenderElementType == RenderElementTypeNone {
			self.RenderElementType = RenderElementTypeComplete
		} else {
			self.RenderElementType = RenderElementTypeLast
		}
	}
	renderElement.addElements(self.Container, self.RenderElementType, self.getUsedHeight())
	return renderElement, self.RenderingComplete
}

func (self *FrameElement) renderSpreadsheet(row int, col int, ctx Context, renderer Renderer) (int, int) {
	if self.SpreadsheetColumn != 0 {
		col = self.SpreadsheetColumn - 1
	}
	row, col = self.Container.RenderSpreadsheet(row, col, ctx, renderer)
	if self.SpreadsheetAddEmptyRow {
		row++
	}
	return row, col
}

func (self FrameElement) cleanup() {
	self.Container.cleanup()
}

func NewFrameElement(report *report, data map[string]interface{}, containers *containers) *FrameElement {
	frameElement := FrameElement{}
	frameElement.initFrameElement(report, data, containers)
	return &frameElement
}

type SectionBandElement struct {
	ID                    string
	Width                 float64
	Height                float64
	BandType              BandType
	RepeatHeader          bool
	AlwaysPrintOnSamePage bool
	ShrinkToContentHeight bool
	Container             Container
	RenderingComplete     bool
	PrepareContainer      bool
	RenderedBandHeight    float64
}

func (self *SectionBandElement) init(report *report, data map[string]interface{}, bandType BandType, containers *containers) {
	self.ID = GetStringValue(data, "id")
	self.Width = report.documentProperties.pageWidth - report.documentProperties.marginBottom - report.documentProperties.marginRight
	self.Height = float64(GetIntValue(data, "height"))
	self.BandType = bandType
	if bandType == BandTypeHeader {
		self.RepeatHeader = GetBoolValue(data, "repeatHeader")
		self.AlwaysPrintOnSamePage = true
	} else {
		self.RepeatHeader = false
		self.AlwaysPrintOnSamePage = GetBoolValue(data, "alwaysPrintOnSamePage")
	}
	self.ShrinkToContentHeight = GetBoolValue(data, "shrinkToContentHeight")

	self.Container = NewContainer(GetStringValue(data, "linkedContainerId"), containers, report)
	self.Container.Width = self.Width
	self.Container.Height = self.Height
	self.Container.AllowPageBreak = false
	self.RenderingComplete = false
	self.PrepareContainer = true
	self.RenderedBandHeight = 0
}

func (self *SectionBandElement) prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool) {
}

func (self *SectionBandElement) createRenderElements(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) {
	availableHeight := containerHeight - offsetY
	if self.AlwaysPrintOnSamePage && self.ShrinkToContentHeight == false && (containerHeight-offsetY) < self.Height {
		// not enough space for whole band
		self.RenderingComplete = false
	} else {
		if self.PrepareContainer {
			// self.Container.prepare(ctx, pdfDoc)
			self.RenderedBandHeight = 0
		} else {
			self.RenderedBandHeight += self.Container.UsedBandHeight
			// clear render elements from previous page
			self.Container.ClearRenderedElements()
		}
		self.RenderingComplete = self.Container.createRenderElements(availableHeight, ctx, pdfDoc)
	}
	if self.RenderingComplete {
		remainingMinHeight := self.Height - self.RenderedBandHeight
		if self.ShrinkToContentHeight == false && self.Container.UsedBandHeight < remainingMinHeight {
			// rendering of band complete, make sure band is at least as large
			// as minimum height (even if it spans over more than 1 page)
			if remainingMinHeight <= availableHeight {
				self.PrepareContainer = true
				self.Container.UsedBandHeight = remainingMinHeight
			} else {
				// minimum height is larger than available space, continue on next page
				self.RenderingComplete = false
				self.PrepareContainer = false
				self.Container.UsedBandHeight = availableHeight
			}
		} else {
			self.PrepareContainer = true
		}
	} else {
		if self.AlwaysPrintOnSamePage {
			// band must be printed on same page but available space is not enough,
			// try to render it on top of next page
			self.PrepareContainer = true
			if offsetY == 0 {
				field := "alwaysPrintOnSamePage"
				if self.BandType == BandTypeHeader {
					field = "size"
				}
				log.Println("errorMsgSectionBandNotOnSamePage", self.ID, field)
				// raise ReportBroError(Error("errorMsgSectionBandNotOnSamePage", self.ID, field))
			}
		} else {
			self.PrepareContainer = false
			self.Container.FirstElementOffsetY = availableHeight
			self.Container.UsedBandHeight = availableHeight
		}
	}
}

func (self *SectionBandElement) getUsedBandHeight() float64 {
	return self.Container.UsedBandHeight
}

func (self *SectionBandElement) getRenderElements() []DocElementBaseProvider {
	return self.Container.RenderElements
}

func NewSectionBandElement(report *report, data map[string]interface{}, bandType BandType, container *containers) *SectionBandElement {
	sectionBandElement := SectionBandElement{}
	sectionBandElement.init(report, data, bandType, container)
	return &sectionBandElement
}

type SectionBand struct {
	Height   float64
	Elements []DocElementBaseProvider
}

type SectionBlockElement struct {
	DocElementBase
	Complete bool
	Bands    []SectionBand
}

func (self *SectionBlockElement) init(report *report, data map[string]interface{}) {
	return
}

func (self *SectionBlockElement) initSectionBlockElement(report *report, renderY float64) {
	self.DocElementBase.init(report, map[string]interface{}{"y": 0})
	self.Report = report
	self.RenderY = renderY
	self.RenderBottom = renderY
	self.Height = 0
	self.Bands = make([]SectionBand, 0)
	self.Complete = false
}

func (self *SectionBlockElement) isEmpty() bool {
	return (len(self.Bands) == 0)
}

func (self *SectionBlockElement) addSectionBand(sectionBand SectionBandElement) {
	if sectionBand.RenderingComplete || sectionBand.AlwaysPrintOnSamePage == false {
		bandHeight := sectionBand.getUsedBandHeight()
		self.Bands = append(self.Bands, SectionBand{Height: bandHeight, Elements: sectionBand.getRenderElements()})
		self.Height += bandHeight
		self.RenderBottom += bandHeight
	}
}

func (self *SectionBlockElement) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB) {
	y := self.RenderY + containerOffsetY
	for _, band := range self.Bands {
		for _, element := range band.Elements {
			element.renderPDF(containerOffsetX, containerOffsetY, pdfDoc)
		}
		y += band.Height
	}
}

func NewSectionBlockElement(report *report, renderY float64) *SectionBlockElement {
	sectionBlockElement := SectionBlockElement{}
	sectionBlockElement.initSectionBlockElement(report, renderY)
	return &sectionBlockElement
}

type SectionElement struct {
	DocElement
	DataSource          string
	Header              *SectionBandElement
	Content             *SectionBandElement
	Footer              *SectionBandElement
	PrintHeader         bool
	DataSourceparameter *Parameter
	RowParameters       map[string]interface{}
	RowCount            int
	RowIndex            int
	Rows                map[int]interface{}
}

func (self *SectionElement) initSectionElement(report *report, data map[string]interface{}, containers *containers) {
	self.DocElement.init(report, data)
	self.DataSource = GetStringValue(data, "dataSource")
	self.PrintIf = GetStringValue(data, "printIf")

	header := GetBoolValue(data, "header")
	footer := GetBoolValue(data, "footer")
	if header {
		self.Header = NewSectionBandElement(report, data["headerData"].(map[string]interface{}), BandTypeHeader, containers)
	} else {
		self.Header = nil
	}
	self.Content = NewSectionBandElement(report, data["contentData"].(map[string]interface{}), BandTypeContent, containers)
	if footer {
		self.Footer = NewSectionBandElement(report, data["footerData"].(map[string]interface{}), BandTypeFooter, containers)
	} else {
		self.Footer = nil
	}
	self.PrintHeader = (self.Header != nil)

	self.X = 0
	self.Width = 0
	self.Height = self.Content.Height
	if self.Header != nil {
		self.Height += self.Header.Height
	}
	if self.Footer != nil {
		self.Height += self.Footer.Height
	}
	self.Bottom = self.Y + self.Height

	self.DataSourceparameter = nil
	self.RowParameters = make(map[string]interface{}, 0)
	self.RowCount = 0
	self.RowIndex = -1
}

func (self *SectionElement) prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool) {
	parameterName := stripParameterName(string(self.DataSource))
	param := map[string]interface{}{}
	parameters := ctx.getParameter(parameterName, param).(map[string]interface{})
	parameter := parameters[parameterName].(Parameter)
	self.DataSourceparameter = &parameter
	if self.DataSourceparameter == nil {
		// raise ReportBroError(Error("errorMsgMissingDataSourceparameter", self.ID, "data_source"))
	}
	if self.DataSourceparameter.Type != ParameterTypeArray {
		// raise ReportBroError(Error("errorMsgInvalidDataSourceparameter", self.ID, "data_source"))
	}
	for _, rowparameter := range self.DataSourceparameter.Children {
		self.RowParameters[rowparameter.(Parameter).Name] = rowparameter.(Parameter)
	}

	rows, parameterExists := ctx.getData(self.DataSourceparameter.Name, nil)
	self.Rows = rows.(map[int]interface{})
	if parameterExists == false {
		// raise ReportBroError(Error("errorMsgMissingData", self.ID, "data_source"))
	}

	// if not isinstance(self.rows, list):
	//     raise ReportBroError(
	//         Error("errorMsgInvalidDataSource", self.ID, "data_source"))

	self.RowCount = len(self.Rows)
	self.RowIndex = 0

	if onlyVerify {
		if self.Header != nil {
			self.Header.prepare(ctx, nil, true)
		}
		for self.RowIndex < self.RowCount {
			// push data context of current row so values of current row can be accessed
			ctx.pushContext(self.RowParameters, self.Rows[self.RowIndex].(map[string]interface{}))
			self.Content.prepare(ctx, nil, true)
			ctx.popContext()
			self.RowIndex += 1
		}
		if self.Footer != nil {
			self.Footer.prepare(ctx, nil, true)
		}
	}
}

func (self *SectionElement) getNextRenderElement(offsetY float64, containerHeight float64, ctx Context, pdfDoc *FPDFRB) (DocElementBaseProvider, bool) {
	self.RenderY = offsetY
	self.RenderBottom = self.RenderY
	renderElement := NewSectionBlockElement(self.Report, offsetY)

	if self.PrintHeader {
		self.Header.createRenderElements(offsetY, containerHeight, ctx, pdfDoc)
		renderElement.addSectionBand(*self.Header)
		if !self.Header.RenderingComplete {
			return renderElement, false
		}
		if !self.Header.RepeatHeader {
			self.PrintHeader = false
		}
	}

	for self.RowIndex < self.RowCount {
		// push data context of current row so values of current row can be accessed
		ctx.pushContext(self.RowParameters, self.Rows[self.RowIndex].(map[string]interface{}))
		self.Content.createRenderElements(offsetY+renderElement.Height, containerHeight, ctx, pdfDoc)
		ctx.popContext()
		renderElement.addSectionBand(*self.Content)
		if !self.Content.RenderingComplete {
			return renderElement, false
		}
		self.RowIndex += 1
	}

	if self.Footer != nil {
		self.Footer.createRenderElements(offsetY+renderElement.Height, containerHeight, ctx, pdfDoc)
		renderElement.addSectionBand(*self.Footer)
		if !self.Footer.RenderingComplete {
			return renderElement, false
		}
	}

	// all bands finished
	self.RenderingComplete = true
	self.RenderBottom += renderElement.Height
	return renderElement, true
}

func (self *SectionElement) renderSpreadsheet(row int, col int, ctx Context, renderer Renderer) (int, int) {
	if self.Header != nil {
		row, _ = self.Header.Container.RenderSpreadsheet(row, col, ctx, renderer)
	}
	row, _ = self.Content.Container.RenderSpreadsheet(row, col, ctx, renderer)
	if self.Footer != nil {
		row, _ = self.Footer.Container.RenderSpreadsheet(row, col, ctx, renderer)
	}
	return row, col
}

func (self *SectionElement) cleanup() {
	if self.Header != nil {
		self.Header.Container.cleanup()
	}
	self.Content.Container.cleanup()
	if self.Footer != nil {
		self.Footer.Container.cleanup()
	}
}

func NewSectionElement(report *report, data map[string]interface{}, containers *containers) *SectionElement {
	sectionElement := SectionElement{}
	sectionElement.initSectionElement(report, data, containers)
	return &sectionElement
}
