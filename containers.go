package reportbro

import (
	"sort"
)

type containerProvider interface {
	base() *Container
	prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool)
	createRenderElements(contentHeight float64, ctx Context, pdfDoc *FPDFRB) bool
	isFinished() bool
	renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB, cleanup bool)
	cleanup()
}

type containers struct {
	containers []*Container
}

func (self *containers) addContainer(container *Container) {
	self.containers = append(self.containers, container)
}

func (self *containers) GetContainer(ID string) *Container {
	for _, container := range self.containers {
		if container.ID == ID {
			return container
		}
	}
	return nil
}

type Container struct {
	ID                    string
	Report                *report
	DocElements           []DocElementBaseProvider
	Width                 float64
	Height                float64
	AllowPageBreak        bool
	ContainerOffsetY      float64
	SortedElements        []DocElementBaseProvider
	RenderElements        []DocElementBaseProvider
	RenderElementsCreated bool
	ExplicitPageBreak     bool
	PageY                 float64
	FirstElementOffsetY   float64
	UsedBandHeight        float64
}

func (self *Container) base() *Container {
	return self
}

func (self *Container) init(containerID string, containers *containers, report *report) {
	self.ID = containerID
	self.Report = report
	self.DocElements = make([]DocElementBaseProvider, 0) // type: List[DocElementBase]
	self.Width = 0.0
	self.Height = 0.0
	containers.addContainer(self)

	self.AllowPageBreak = true
	self.ContainerOffsetY = 0.0
	self.SortedElements = nil // type: List[DocElementBase]
	self.RenderElements = nil // type: List[DocElementBase]
	self.RenderElementsCreated = false
	self.ExplicitPageBreak = true
	self.PageY = 0.0
	self.FirstElementOffsetY = 0.0
	self.UsedBandHeight = 0.0
}

func (self *Container) add(docElement DocElementBaseProvider) {
	self.DocElements = append(self.DocElements, docElement)
}

func (self *Container) isVisible() bool {
	return true
}

func (self *Container) prepare(ctx Context, pdfDoc *FPDFRB, onlyVerify bool) {
	self.SortedElements = make([]DocElementBaseProvider, 0)
	for _, elem := range self.DocElements {
		if pdfDoc != nil || elem.base().SpreadsheetHide == false || onlyVerify {
			elem.prepare(ctx, pdfDoc, onlyVerify)
			if self.AllowPageBreak == false {
				// make sure element can be rendered multiple times (for header/footer)
				elem.base().FirstRenderElement = true
				elem.base().RenderingComplete = false
			}
			self.SortedElements = append(self.SortedElements, elem)
		}
	}

	if pdfDoc != nil {
		sort.Slice(self.SortedElements, func(i, j int) bool {
			return (self.SortedElements[i].base().Y < self.SortedElements[j].base().Y)
		})
		// predecessors are only needed for rendering pdf document
		for i, elem := range self.SortedElements {
			for _, j := range pyRange(i-1, -1, -1) {
				elem2 := self.SortedElements[j]
				if _, ok := elem2.(*PageBreakElement); ok {
					// new page so all elements before are not direct predecessors
					break
				}
				if elem.base().isPredecessor(elem2) {
					elem.base().addPredecessor(elem2)
				}
			}
		}
		self.RenderElements = make([]DocElementBaseProvider, 0)
		self.UsedBandHeight = 0
		self.FirstElementOffsetY = 0
	} else {
		sort.Slice(self.SortedElements, func(i, j int) bool {
			return self.SortedElements[i].base().Y < self.SortedElements[j].base().X
		})
	}
}

func (self *Container) ClearRenderedElements() {
	self.RenderElements = make([]DocElementBaseProvider, 0)
	self.UsedBandHeight = 0.0
}

func (self *Container) GetRenderElementsBottom() float64 {
	bottom := 0.0
	for _, renderElement := range self.RenderElements {
		if renderElement.base().RenderBottom > bottom {
			bottom = renderElement.base().RenderBottom
		}
	}
	return bottom
}

func (self *Container) createRenderElements(containerHeight float64, ctx Context, pdfDoc *FPDFRB) bool {
	i := 0
	newPage := false
	processedElements := make([]DocElementBaseProvider, 0)
	completedElements := map[int]bool{}

	self.RenderElementsCreated = false
	setExplicitPageBreak := false
	for newPage == false && i < len(self.SortedElements) {
		elem := self.SortedElements[i]
		if elem.hasUncompletedPredecessor(completedElements) {
			// a predecessor is not completed yet -> start new page
			newPage = true
		} else {
			elemDeleted := false
			if _, ok := elem.(*PageBreakElement); ok {
				if self.AllowPageBreak {
					self.SortedElements = removeElement(self.SortedElements, i)
					elemDeleted = true
					newPage = true
					setExplicitPageBreak = true
					self.PageY = elem.base().Y
				} else {
					self.SortedElements = make([]DocElementBaseProvider, 0)
					return true
				}
			} else {
				offsetY := 0.0
				complete := false
				if len(elem.base().Predecessors) > 0 {
					// element is on same page as predecessor element(s) so offset is relative to predecessors
					offsetY = elem.base().GetOffsetY()
				} else {
					if self.AllowPageBreak {
						if elem.base().FirstRenderElement && self.ExplicitPageBreak {
							offsetY = elem.base().Y - self.base().PageY
						} else {
							offsetY = 0
						}
					} else {
						offsetY = elem.base().Y - self.FirstElementOffsetY
						if offsetY < 0 {
							offsetY = 0
						}
					}
				}
				var renderElem DocElementBaseProvider

				if elem.isPrinted(ctx) {
					if offsetY >= containerHeight {
						newPage = true
					}
					if newPage == false {
						renderElem, complete = elem.getNextRenderElement(offsetY, containerHeight, ctx, pdfDoc)
						if renderElem != nil {
							if complete {
								processedElements = append(processedElements, elem)
							}
							self.RenderElements = append(self.RenderElements, renderElem)
							self.RenderElementsCreated = true
							if renderElem.base().RenderBottom > self.UsedBandHeight {
								self.UsedBandHeight = renderElem.base().RenderBottom
							}
						}
					}
				} else {
					processedElements = append(processedElements, elem)
					elem.finishEmptyElement(offsetY)
					complete = true
				}

				if complete {
					completedElements[elem.base().ID] = true
					self.SortedElements = removeElement(self.SortedElements, i)
					elemDeleted = true
				}
			}
			if elemDeleted == false {
				i += 1
			}
		}
	}

	// in case of manual page break the element on the next page is positioned relative
	// to page break position
	self.ExplicitPageBreak = true
	if self.AllowPageBreak {
		self.ExplicitPageBreak = setExplicitPageBreak
	}

	if len(self.SortedElements) > 0 {
		if self.AllowPageBreak {
			self.RenderElements = append(self.RenderElements, NewPageBreakElement(self.Report, map[string]interface{}{"y": -1}))
		}
		for _, processedElement := range processedElements {
			// remove dependency to predecessors because successor element is either already added
			// to renderElements or on new page
			for _, successor := range processedElement.base().Successors {
				succ := successor
				succ.base().clearPredecessors()
			}
		}
	}

	return (len(self.SortedElements) == 0)
}

func (self *Container) renderPDF(containerOffsetX float64, containerOffsetY float64, pdfDoc *FPDFRB, cleanup bool) {
	counter := 0
	for _, renderElem := range self.RenderElements {
		counter += 1
		if _, ok := renderElem.(*PageBreakElement); ok {
			break
		}

		renderElem.renderPDF(containerOffsetX, containerOffsetY, pdfDoc)
		if cleanup {
			renderElem.cleanup()
		}
	}
	self.RenderElements = self.RenderElements[counter:]
}

func (self *Container) RenderSpreadsheet(row int, col int, ctx Context, renderer Renderer) (int, int) {
	return row, col
}

func (self *Container) isFinished() bool {
	return len(self.RenderElements) == 0
}

func (self *Container) cleanup() {
	for _, elem := range self.RenderElements {
		elem.cleanup()
	}
}

func NewContainer(containerID string, containers *containers, report *report) Container {
	container := Container{}
	container.init(containerID, containers, report)
	return container
}

type Frame struct {
	Container
	X               float64
	Width           float64
	Height          float64
	BorderStyle     BorderStyle
	BackgroundColor Color
}

func (self *Frame) init(width float64, height float64, containerID string, containers *containers, report *report) {
	self.Container.init(containerID, containers, report)
	self.Container.Width = width
	self.Container.Height = height
	self.AllowPageBreak = false
}

func NewFrame(width float64, height float64, containerID string, containers *containers, report *report) *Frame {
	frame := Frame{}
	frame.init(width, height, containerID, containers, report)
	return &frame
}

type reportBand struct {
	Container
	Band   BandType
	Report *report
}

func (self *reportBand) init(band BandType, containerID string, containers *containers, report *report) {
	self.Container.init(containerID, containers, report)
	self.Band = band
	self.Width = report.documentProperties.pageWidth - report.documentProperties.marginLeft - report.documentProperties.marginRight
	if band == BandTypeContent {
		self.Height = report.documentProperties.contentHeight
	} else if band == BandTypeHeader {
		self.AllowPageBreak = false
		self.Height = report.documentProperties.headerSize
	} else if band == BandTypeFooter {
		self.AllowPageBreak = false
		self.Height = report.documentProperties.footerSize
	}
}

func (self *reportBand) isVisible() bool {
	if self.Band == BandTypeHeader {
		return self.Report.documentProperties.header
	}
	return self.Report.documentProperties.footer
}

func newReportBand(band BandType, containerID string, containers *containers, report *report) *reportBand {
	reportBand := reportBand{}
	reportBand.init(band, containerID, containers, report)
	return &reportBand
}
