package gitConnector

type GitConnector interface {
	Connect() string
	BuildStart()
	BuildFail()
	BuildSucc()
	BuildStop()
	CleanComment()
	Comment(string)
	GetToken() string
	PrintCloneURL()
	PrintExecutePath()
}
