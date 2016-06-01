package combos

import (
	"fmt"
	"strings"
)

import (
	lex "github.com/timtadh/lexmachine"
	"github.com/timtadh/lexmachine/frontend"
)

type ParseError struct {
	Reason string
	At *SourceLocation
	Chained []error
}

func ErrorOn(reason string, on *lex.Token) *ParseError {
	reason = fmt.Sprintf("'%v' : %v", string(on.Lexeme), reason)
	return &ParseError{Reason: reason, At: TokenLocation(on)}
}

func Unconsumed(s *lex.Scanner) *ParseError {
	return Error(s, "Unconsumed Input")
}

func EOS(s *lex.Scanner, token interface{}) *ParseError {
	return Error(s, "Unexpected end of string (EOS) expected : %v", token)
}

func Error(s *lex.Scanner, fmtString string, args ...interface{}) *ParseError {
	start := s.TC
	if start >= len(s.Text) {
		start = len(s.Text) - 1
	}
	end := start + 10
	if end > len(s.Text) {
		end = len(s.Text)
	}
	sline, scol := frontend.LineCol(s.Text, start)
	eline, ecol := frontend.LineCol(s.Text, end)
	loc := &SourceLocation{
		StartLine: sline,
		StartColumn: scol,
		EndLine: eline,
		EndColumn: ecol,
	}
	text := string(s.Text[start:end])
	reason := fmt.Sprintf(fmtString, args...)
	reason = fmt.Sprintf("'%v' : %v", reason, text)
	return &ParseError{Reason: reason, At: loc}
}

func (p *ParseError) Chain(e error) *ParseError {
	p.Chained = append(p.Chained, e)
	return p
}

func (p *ParseError) Error() string {
	errs := make([]string, 0, len(p.Chained)+1)
	for i := len(p.Chained) - 1; i >= 0; i-- {
		errs = append(errs, p.Chained[i].Error())
	}
	errs = append(errs, p.error())
	return strings.Join(errs, "\n")
}

func (p *ParseError) error() string {
	if p.At == nil {
		return fmt.Sprintf("Parse Error @ EOS : %v", p.Reason)
	} else {
		return fmt.Sprintf("Parse Error @ %v:%v-%v:%v : %v",
			p.At.StartLine,
			p.At.StartColumn,
			p.At.EndLine,
			p.At.EndColumn,
			p.Reason)
	}
}

func (p *ParseError) Less(o *ParseError) bool {
	if p == nil || o == nil {
		return false
	}
	if p.At == nil || o.At == nil {
		return false
	}
	if p.At.StartLine < o.At.StartLine {
		return true
	} else if p.At.StartLine > o.At.StartLine {
		return false
	}
	if p.At.StartColumn < o.At.StartColumn {
		return true
	} else if p.At.StartColumn > o.At.StartColumn {
		return false
	}
	if p.At.EndLine > o.At.EndLine {
		return true
	} else if p.At.EndLine < o.At.EndLine {
		return false
	}
	if p.At.EndColumn > o.At.EndColumn {
		return true
	} else if p.At.EndColumn < o.At.EndColumn {
		return false
	}
	return false
}
