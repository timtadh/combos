package combos

import (
	"fmt"
)

import (
	lex "github.com/timtadh/lexmachine"
)

type Action func(ctx interface{}, nodes ...*Node) (*Node, *ParseError)

type Consumer interface {
	Consume(p *Parser) (*Node, *ParseError)
}

type FnConsumer func(p *Parser) (*Node, *ParseError)

func (self FnConsumer) Consume(p *Parser) (*Node, *ParseError) {
	return self(p)
}

type LazyConsumer struct {
	G *Grammar
	ProductionName string
}

func (l *LazyConsumer) Consume(p *Parser) (*Node, *ParseError) {
	if p.G.Debug {
		fmt.Printf("start lazy %v\n", l.ProductionName)
	}
	n, e := l.G.productions[l.ProductionName].Consume(p)
	if p.G.Debug {
		name := ""
		if n != nil {
			name = n.Label
		}
		if e == nil {
			fmt.Printf("end lazy %v %v\n", l.ProductionName, name)
		} else {
			fmt.Printf("fail lazy %v\n", l.ProductionName)
		}
	}
	return n, e
}

type Parser struct {
	Ctx interface{} // User supplied ctx type passed into Concat Action functions. Optional.
	G *Grammar // The Grammar driving the parsing
	S *lex.Scanner // The Scanner giving the tokens to be parsed
	LastError *ParseError // The last error
	UserError *ParseError // The last error from a Action
}

type Grammar struct {
	Tokens []string
	TokenIds map[string]int
	productions map[string]Consumer
	startProduction string
	Debug bool // Set this flag to print out debug information during parsing
}

func NewGrammar(tokens []string, tokenIds map[string]int) *Grammar {
	g := &Grammar{
		Tokens: tokens,
		TokenIds: tokenIds,
		productions: make(map[string]Consumer),
	}
	for _, token := range tokens {
		g.AddRule(token, g.Consume(token))
	}
	return g
}

// Using the *Grammar parse the string being scanned by the lexer with the given
// parser context.
func (g *Grammar) Parse(s *lex.Scanner, parserCtx interface{}) (*Node, *ParseError) {
	p := &Parser{
		Ctx: parserCtx,
		G: g,
		S: s,
	}
	n, err := g.productions[g.startProduction].Consume(p)
	if err != nil {
		return nil, err
	}

	if p.UserError != nil {
		return nil, p.UserError
	}
	
	t, serr, eof := s.Next()
	if eof {
		return n, nil
	} else if p.LastError != nil {
		return nil, p.LastError
	} else if serr != nil {
		return nil, Error("Unconsumed Input", nil).Chain(err)
	} else {
		return nil, Error("Unconsumed Input", t.(*lex.Token))
	}
}

func (g *Grammar) Start(name string) *Grammar {
	g.startProduction = name
	return g
}

func (g *Grammar) AddRule(name string, c Consumer) *Grammar {
	g.productions[name] = c
	return g
}

func (g *Grammar) P(productionName string) Consumer {
	return &LazyConsumer{G: g, ProductionName: productionName}
}

func (g *Grammar) Effect(consumers ...Consumer) func(do func(interface{}, ...*Node) error) Consumer {
	return func(do func(interface{}, ...*Node) error) Consumer {
		return FnConsumer(func(p *Parser) (n *Node, err *ParseError) {
			tc := p.S.TC
			nodes, err := g.concat(consumers, p)
			if err != nil {
				p.S.TC = tc
				return nil, err
			}
			doerr := do(p.Ctx, nodes...)
			if doerr != nil {
				p.S.TC = tc
				t, _, _ := p.S.Next()
				if t == nil {
					err := Error("Side Effect Error", nil).Chain(doerr)
					p.UserError = err
					return nil, err
				}
				tok := t.(*lex.Token)
				err := Error("Side Effect Error", tok).Chain(doerr)
				p.UserError = err
				return nil, err
			}
			n = NewNode("Effect")
			n.Children = nodes
			return n, nil
		})
	}
}

func (g *Grammar) Memoize(c Consumer) Consumer {
	type result struct {
		n *Node
		e *ParseError
		tc int
	}
	var s *lex.Scanner
	var cache map[int]*result
	return FnConsumer(func(p *Parser) (*Node, *ParseError) {
		if cache == nil || s != p.S {
			cache = make(map[int]*result)
			s = p.S
		}
		tc := p.S.TC
		if res, in := cache[tc]; in {
			p.S.TC = res.tc
			return res.n, res.e
		}
		n, e := c.Consume(p)
		cache[tc] = &result{n, e, p.S.TC}
		return n, e
	})
}

func (g *Grammar) Epsilon(n *Node) Consumer {
	return FnConsumer(func(p *Parser) (*Node, *ParseError) {
		if g.Debug {
			fmt.Printf("epsilon %v\n", n)
		}
		return n, nil
	})
}

func (g *Grammar) Concat(consumers ...Consumer) func(Action) Consumer {
	return func(action Action) Consumer {
		return (FnConsumer(func(p *Parser) (*Node, *ParseError) {
			tc := p.S.TC
			nodes, err := g.concat(consumers, p)
			if err != nil {
				p.S.TC = tc
				return nil, err
			}
			an, aerr := action(p.Ctx, nodes...)
			if aerr != nil {
				p.S.TC = tc
				p.UserError = aerr
				return nil, aerr
			}
			return an, nil
		}))
	}
}

func (g *Grammar) concat(consumers []Consumer, p *Parser) ([]*Node, *ParseError) {
	var nodes []*Node
	var n *Node
	var err *ParseError
	tc := p.S.TC
	for _, c := range consumers {
		n, err = c.Consume(p)
		if err == nil {
			nodes = append(nodes, n)
		} else {
			p.S.TC = tc
			return nil, err
		}
	}
	return nodes, nil
}

func (g *Grammar) Alt(consumers ...Consumer) Consumer {
	return (FnConsumer(func(p *Parser) (*Node, *ParseError) {
		var err *ParseError = nil
		tc := p.S.TC
		always := false
		for _, c := range consumers {
			p.S.TC = tc
			n, e := c.Consume(p)
			if e == nil {
				return n, nil
			} else if err == nil {
				err = e
			} else if e.Less(err) {
				// err = err.Chain(e)
				err = err
			} else {
				// err = e.Chain(err)
				err = e
			}
			if p.LastError == nil || always {
				p.LastError = err
			} else if p.LastError.Less(err) {
				always = true
				p.LastError = err
			}
		}
		p.S.TC = tc
		return nil, err
	}))
}

func (g *Grammar) Consume(token string) Consumer {
	return FnConsumer(func(p *Parser) (*Node, *ParseError) {
		tc := p.S.TC
		t, err, eof := p.S.Next()
		if eof {
			p.S.TC = tc
			return nil, Error(
				fmt.Sprintf("Ran off the end of the input. expected '%v''", token), nil)
		}
		if err != nil {
			p.S.TC = tc
			return nil, Error("Lexer Error", nil).Chain(err)
		}
		tk := t.(*lex.Token)
		if tk.Type == g.TokenIds[token] {
			return NewTokenNode(g, tk), nil
		}
		p.S.TC = tc
		return nil, Error(fmt.Sprintf("Expected %v", token), tk)
	})
}

func (g *Grammar) Peek(tokens ...string) Consumer {
	return FnConsumer(func(p *Parser) (*Node, *ParseError) {
		tc := p.S.TC
		t, err, eof := p.S.Next()
		p.S.TC = tc
		if eof {
			return nil, Error(
				fmt.Sprintf("Ran off the end of the input. expected '%v''", tokens), nil)
		}
		if err != nil {
			return nil, Error("Lexer Error", nil).Chain(err)
		}
		tk := t.(*lex.Token)
		for _, token := range tokens {
			if tk.Type == g.TokenIds[token] {
				return nil, nil
			}
		}
		return nil, Error(fmt.Sprintf("Expected one of %v", tokens), tk)
	})
}
