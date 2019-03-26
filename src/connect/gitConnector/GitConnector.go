package gitConnector

type GitConnector interface {
	Connect() string
	BuildStart()
	BuildFail()
	BuildSucc()
	BuildStop()
	Comment(string)
	GetToken() string
	PrintCloneURL()
	PrintExecutePath()
}
