package opml

type Opml struct {
	Title string
	Body  *Node
}

type Node struct {
	Attr     map[string]string
	Children []*Node
}
