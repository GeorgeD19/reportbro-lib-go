package reportbro

type Border int

const (
	BorderGrid     Border = 1
	BorderFrameRow Border = 2
	BorderFrame    Border = 3
	BorderRow      Border = 4
	BorderNone     Border = 5
)

var borders = [...]string{
	"grid",
	"frame_row",
	"frame",
	"row",
	"none",
}

func (border Border) String() string {
	return borders[border-1]
}

func GetBorder(border string) Border {
	switch border {
	case BorderGrid.String():
		return BorderGrid
	case BorderFrameRow.String():
		return BorderFrameRow
	case BorderFrame.String():
		return BorderFrame
	case BorderRow.String():
		return BorderRow
	case BorderNone.String():
		return BorderNone
	}
	return BorderNone
}

type RenderElementType int

const (
	RenderElementTypeNone     RenderElementType = 0
	RenderElementTypeComplete RenderElementType = 1
	RenderElementTypeBetween  RenderElementType = 2
	RenderElementTypeFirst    RenderElementType = 3
	RenderElementTypeLast     RenderElementType = 4
)

var renderElementTypes = [...]string{
	"none",
	"complete",
	"between",
	"first",
	"last",
}

func (renderElementType RenderElementType) String() string {
	return renderElementTypes[renderElementType]
}

func GetRenderElementType(renderElementType string) RenderElementType {
	switch renderElementType {
	case RenderElementTypeNone.String():
		return RenderElementTypeNone
	case RenderElementTypeComplete.String():
		return RenderElementTypeComplete
	case RenderElementTypeBetween.String():
		return RenderElementTypeBetween
	case RenderElementTypeFirst.String():
		return RenderElementTypeFirst
	case RenderElementTypeLast.String():
		return RenderElementTypeLast
	}
	return RenderElementTypeNone
}

type DocElementType int

const (
	DocElementTypeText        DocElementType = 1
	DocElementTypeImage       DocElementType = 2
	DocElementTypeLine        DocElementType = 3
	DocElementTypePageBreak   DocElementType = 4
	DocElementTypeTableBand   DocElementType = 5
	DocElementTypeTable       DocElementType = 6
	DocElementTypeTableText   DocElementType = 7
	DocElementTypeBarCode     DocElementType = 8
	DocElementTypeFrame       DocElementType = 9
	DocElementTypeSection     DocElementType = 10
	DocElementTypeSectionBand DocElementType = 11
)

var docElementTypes = [...]string{
	"text",
	"image",
	"line",
	"page_break",
	"table_band",
	"table",
	"table_text",
	"bar_code",
	"frame",
	"section",
	"section_band",
}

func (docElementType DocElementType) String() string {
	return docElementTypes[docElementType-1]
}

func GetDocElementType(docElementType string) DocElementType {
	switch docElementType {
	case DocElementTypeText.String():
		return DocElementTypeText
	case DocElementTypeImage.String():
		return DocElementTypeImage
	case DocElementTypeLine.String():
		return DocElementTypeLine
	case DocElementTypePageBreak.String():
		return DocElementTypePageBreak
	case DocElementTypeTableBand.String():
		return DocElementTypeTableBand
	case DocElementTypeTable.String():
		return DocElementTypeTable
	case DocElementTypeTableText.String():
		return DocElementTypeTableText
	case DocElementTypeBarCode.String():
		return DocElementTypeBarCode
	case DocElementTypeFrame.String():
		return DocElementTypeFrame
	case DocElementTypeSection.String():
		return DocElementTypeSection
	case DocElementTypeSectionBand.String():
		return DocElementTypeSectionBand
	}
	return DocElementTypeText
}

type ParameterType int

const (
	ParameterTypeNone        ParameterType = 0
	ParameterTypeString      ParameterType = 1
	ParameterTypeNumber      ParameterType = 2
	ParameterTypeBoolean     ParameterType = 3
	ParameterTypeDate        ParameterType = 4
	ParameterTypeArray       ParameterType = 5
	ParameterTypeSimpleArray ParameterType = 6
	ParameterTypeMap         ParameterType = 7
	ParameterTypeSum         ParameterType = 8
	ParameterTypeAverage     ParameterType = 9
	ParameterTypeImage       ParameterType = 10
)

var ParameterTypes = [...]string{
	"none",
	"string",
	"number",
	"boolean",
	"date",
	"array",
	"simple_array",
	"map",
	"sum",
	"average",
	"image",
}

func (ParameterType ParameterType) String() string {
	return ParameterTypes[ParameterType]
}

func getParameterType(ParameterType string) ParameterType {
	switch ParameterType {
	case ParameterTypeNone.String():
		return ParameterTypeNone
	case ParameterTypeString.String():
		return ParameterTypeString
	case ParameterTypeNumber.String():
		return ParameterTypeNumber
	case ParameterTypeBoolean.String():
		return ParameterTypeBoolean
	case ParameterTypeDate.String():
		return ParameterTypeDate
	case ParameterTypeArray.String():
		return ParameterTypeArray
	case ParameterTypeSimpleArray.String():
		return ParameterTypeSimpleArray
	case ParameterTypeMap.String():
		return ParameterTypeMap
	case ParameterTypeSum.String():
		return ParameterTypeSum
	case ParameterTypeAverage.String():
		return ParameterTypeAverage
	case ParameterTypeImage.String():
		return ParameterTypeImage
	}
	return ParameterTypeNone
}

type BandType int

const (
	BandTypeHeader  BandType = 1
	BandTypeContent BandType = 2
	BandTypeFooter  BandType = 3
)

var bandTypes = [...]string{
	"header",
	"content",
	"footer",
}

func (bandType BandType) String() string {
	return bandTypes[bandType-1]
}

func GetBandType(bandType string) BandType {
	switch bandType {
	case BandTypeHeader.String():
		return BandTypeHeader
	case BandTypeContent.String():
		return BandTypeContent
	case BandTypeFooter.String():
		return BandTypeFooter
	}
	return BandTypeContent
}

type PageFormat int

const (
	PageFormatA4          PageFormat = 1
	PageFormatA5          PageFormat = 2
	PageFormatLetter      PageFormat = 3
	PageFormatUserDefined PageFormat = 4
)

var pageFormats = [...]string{
	"a4",
	"a5",
	"letter",
	"user_defined",
}

func (pageFormat PageFormat) String() string {
	if (pageFormat - 1) > 0 {
		return pageFormats[pageFormat-1]
	}
	return pageFormats[0]
}

func getPageFormat(pageFormat string) PageFormat {
	switch pageFormat {
	case PageFormatA4.String():
		return PageFormatA4
	case PageFormatA5.String():
		return PageFormatA5
	case PageFormatLetter.String():
		return PageFormatLetter
	case PageFormatUserDefined.String():
		return PageFormatUserDefined
	}
	return PageFormatA4
}

type Unit int

const (
	UnitPt   Unit = 1
	UnitMm   Unit = 2
	UnitInch Unit = 3
)

var units = [...]string{
	"pt",
	"mm",
	"inch",
}

func (unit Unit) String() string {
	return units[unit-1]
}

func GetUnit(unit string) Unit {
	switch unit {
	case UnitPt.String():
		return UnitPt
	case UnitMm.String():
		return UnitMm
	case UnitInch.String():
		return UnitInch
	}
	return UnitPt
}

type Orientation int

const (
	OrientationPortrait  Orientation = 1
	OrientationLandscape Orientation = 2
)

var orientations = [...]string{
	"portrait",
	"landscape",
}

func (orientation Orientation) String() string {
	return orientations[orientation-1]
}

func getOrientation(orientation string) Orientation {
	switch orientation {
	case OrientationPortrait.String():
		return OrientationPortrait
	case OrientationLandscape.String():
		return OrientationLandscape
	}
	return OrientationPortrait
}

type BandDisplay int

const (
	BandDisplayNever          BandDisplay = 1
	BandDisplayAlways         BandDisplay = 2
	BandDisplayNotOnFirstPage BandDisplay = 3
)

var bandDisplays = [...]string{
	"never",
	"always",
	"not_on_first_page",
}

func (bandDisplay BandDisplay) String() string {
	return bandDisplays[bandDisplay-1]
}

func GetBandDisplay(bandDisplay string) BandDisplay {
	switch bandDisplay {
	case BandDisplayNever.String():
		return BandDisplayNever
	case BandDisplayAlways.String():
		return BandDisplayAlways
	case BandDisplayNotOnFirstPage.String():
		return BandDisplayNotOnFirstPage
	}
	return BandDisplayNever
}

type HorizontalAlignment int

const (
	HorizontalAlignmentLeft    HorizontalAlignment = 1
	HorizontalAlignmentCenter  HorizontalAlignment = 2
	HorizontalAlignmentRight   HorizontalAlignment = 3
	HorizontalAlignmentJustify HorizontalAlignment = 4
)

var horizontalAlignments = [...]string{
	"left",
	"center",
	"right",
	"justify",
}

func (horizontalAlignment HorizontalAlignment) String() string {
	return horizontalAlignments[horizontalAlignment-1]
}

func GetHorizontalAlignment(horizontalAlignment string) HorizontalAlignment {
	switch horizontalAlignment {
	case HorizontalAlignmentLeft.String():
		return HorizontalAlignmentLeft
	case HorizontalAlignmentCenter.String():
		return HorizontalAlignmentCenter
	case HorizontalAlignmentRight.String():
		return HorizontalAlignmentRight
	case HorizontalAlignmentJustify.String():
		return HorizontalAlignmentJustify
	}
	return HorizontalAlignmentLeft
}

type VerticalAlignment int

const (
	VerticalAlignmentTop    VerticalAlignment = 1
	VerticalAlignmentMiddle VerticalAlignment = 2
	VerticalAlignmentBottom VerticalAlignment = 3
)

var verticalAlignments = [...]string{
	"top",
	"middle",
	"bottom",
}

func (verticalAlignment VerticalAlignment) String() string {
	return verticalAlignments[verticalAlignment-1]
}

func GetVerticalAlignment(verticalAlignment string) VerticalAlignment {
	switch verticalAlignment {
	case VerticalAlignmentTop.String():
		return VerticalAlignmentTop
	case VerticalAlignmentMiddle.String():
		return VerticalAlignmentMiddle
	case VerticalAlignmentBottom.String():
		return VerticalAlignmentBottom
	}
	return VerticalAlignmentTop
}

func getLocaleCurrency(locale string) string {
	switch locale {
	case "de", "es", "fr", "it":
		return "EUR"
	case "en-GB":
		return "GBP"
	}
	return "USD"
}
