package opml

import (
	"encoding/xml"
	"fmt"
	"io"
	//"log"
)

type Parser struct {
	Result  Opml
	decoder *xml.Decoder
}

func NewParser(reader io.Reader) *Parser {
	parser := &Parser{ decoder: xml.NewDecoder(reader) }
	parser.Result.Body = new(Node)
	return parser
}

func (p *Parser) Parse() error {
	for {
		token, err := p.decoder.Token()
		if token == nil {
			break
		}
		if err != nil {
			return err
		}
		child, err := p.handleToken(token)
		if err != nil {
			return err
		}
		if child != nil {
			p.Result.Body.Children = append(p.Result.Body.Children, child)
		}
	}
	return nil
}

func (p *Parser) handleToken(token xml.Token) (*Node, error) {
	if v, ok := token.(xml.StartElement); ok {
		if v.Name.Local != "outline" {
			return nil, nil
		}

		node := &Node { Attr : make(map[string]string) }
		for _, attr := range v.Attr {
			node.Attr[attr.Name.Local] = attr.Value
		}
		for {
			want, err := p.decoder.Token()
			switch {
			case want == nil:
				return nil, fmt.Errorf("Unexpected EndElement, Tag: %s",
					v.Name.Local)
			case err != nil:
				return nil, err
			default:
				if _, ok := want.(xml.EndElement); ok {
					return node, nil
				}
				child, err := p.handleToken(want)
				if err != nil {
					return nil, err
				}
				if child != nil {
					node.Children = append(node.Children, child)
				}
			}
		}
	}
	return nil, nil
}
