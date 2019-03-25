package gitConnector

type GitConnector interface {
	Initialize()
	Connect()
	BuildStart()
	BuildFail()
	BuildSucc()
	BuildStop()
	Comment(string)
	GetToken() string
}
