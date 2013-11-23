package opml

import (
	. "launchpad.net/gocheck"

	"bytes"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ParserTest struct{}

var _ = Suite(&ParserTest{})

const testSanityOpml = `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
	<head>
		<title>testSanityOpml</title>
	</head>
	<body>
		<outline title="dir" text="dir">
			<outline text="a" title="a" type="rss"
				xmlUrl="http://www.a.com/rss"
				htmlUrl="https://www.a.com/index.html"/>
			</outline>
		</outline>
	</body>
</opml>`

func (test *ParserTest) TestSanity(t *C) {
	p := NewParser(bytes.NewReader([]byte(testSanityOpml)))
	t.Assert(p.Parse(), IsNil)

	node := p.Result.Body.Children[0]
	t.Assert(node.Attr["title"], Equals, "dir")
	t.Assert(node.Attr["text"], Equals, "dir")

	node = node.Children[0]
	t.Assert(node.Attr["title"], Equals, "a")
	t.Assert(node.Attr["text"], Equals, "a")
	t.Assert(node.Attr["type"], Equals, "rss")
	t.Assert(node.Attr["xmlUrl"], Equals, "http://www.a.com/rss")
	t.Assert(node.Attr["htmlUrl"], Equals, "https://www.a.com/index.html")
}
