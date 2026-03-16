package wikireader

// WikiPage represents a MediaWiki page
type WikiPage struct {
	ID         int
	Namespace  int
	Title      string
	Content    string
	Timestamp  string
	IsRedirect bool
}

type WikiReader interface {
	FetchPages() ([]WikiPage, error)
	Close() error
}
