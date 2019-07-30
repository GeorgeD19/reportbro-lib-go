package reportbro

type reportBroError struct {
	error
	Error Error
}

func (self *reportBroError) init(err Error) {
	self.Error = err
}

func NewReportBroError(err Error) reportBroError {
	reportBroError := reportBroError{}
	reportBroError.init(err)
	return reportBroError
}

type Error struct {
	Message  string
	ObjectID int
	Field    string
	Info     interface{}
	context  string
}
