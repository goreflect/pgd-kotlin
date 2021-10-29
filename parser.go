package pgdkotlin

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sirupsen/logrus"
)

type Parser struct {
	s   *Scanner
	buf struct {
		tok   Token  // last read token
		lit   string // last read literal
		start int    // start position
		end   int    // end position
		n     int    // buffer size (max=1)
	}
	allDependencies []Depenedency
}

// NewParser returns a new instance of Parser.
func NewParser(r io.Reader) *Parser {
	return &Parser{s: NewScanner(r)}
}

func (p *Parser) scan() (tok Token, lit string, start, end int) {
	// If we have a token on the buffer, then return it.
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit, p.buf.start, p.buf.end
	}

	// Otherwise read the next token from the scanner.
	tok, lit, start, end = p.s.Scan()

	// Save it to the buffer in case we unscan later.
	p.buf.tok, p.buf.lit, p.buf.start, p.buf.end = tok, lit, start, end

	return
}

// unscan pushes the previously read token back onto the buffer.
func (p *Parser) unscan() { p.buf.n = 1 }

func (p *Parser) scanIgnoreWhitespace() (tok Token, lit string, start, end int) {
	for {
		tok, lit, start, end = p.scan()
		if tok == WS || tok == NEW_LINE {
		} else {
			break
		}
	}

	return
}

func (p *Parser) scanIgnoreLineWhiteSpace() (tok Token, lit string, start, end int) {
	for {
		tok, lit, start, end = p.scan()
		if tok != NAME && tok != NUMBER && tok != POINT && tok != COLON && tok != MINUS {
		} else {
			break
		}
	}

	return
}

func (p *Parser) scanIgnoreWhitespaceAndNotNewLine() (tok Token, lit string, start, end int) {
	for {
		tok, lit, start, end = p.scan()
		if tok == WS {
		} else {
			break
		}
	}

	return
}

func (p *Parser) scanIgnoreQuta() (tok Token, lit string, start, end int) {
	for {
		tok, lit, start, end = p.scan()
		if tok == QUOTION || tok == WS {
		} else {
			break
		}
	}

	return
}

func (p *Parser) skipQuotaAndMinus() {
	for {
		tok, _, _, _ := p.scan()
		if tok == QUOTION || tok == MINUS || tok == NEW_LINE {

		} else if tok == MULTY_NEW_LINE {
			p.unscan()
			break
		} else {
			break
		}
	}
}

func (p *Parser) scanIgnoreAllValuesBeforeProject() (tok Token, lit string, start, end int) {
	for {
		tok, lit, start, end = p.scan()
		if tok == NAME {
			if strings.ToLower(lit) == "project" {
				break
			}
		} else if tok == EOF {
			break
		}
	}

	return
}

func (p *Parser) skipBeforeStartDependencies() {
	for {
		tok, _, start, end := p.scan()
		if tok == MULTY_NEW_LINE {
			logrus.Debug("Found MultiyNewLine. Start: ", start, " End pos: ", end)
		} else {
			p.unscan()
			break
		}
	}
}

func (p *Parser) ParseGradleDependenciesPerProject() (*Project, error) {
	projectDependency := Project{}
	tok, lit, start, end := p.scanIgnoreAllValuesBeforeProject()
	if tok == EOF {
		logrus.Debug("Project not found")
		return nil, nil
	}
	if tok != NAME {
		return nil, fmt.Errorf("found %q, expected project name. Start position: %d; End Position: %d", lit, start, end)
	}

	tok, lit, _, _ = p.scanIgnoreQuta()
	if tok == COLON {
		tok, lit, start, end := p.scanIgnoreWhitespace()
		if tok == NAME {
			projectDependency.Sym = TerminalSymbol{
				Tok:            tok,
				Literal:        lit,
				StartPositioin: start,
				EndPosition:    end,
			}
		} else {
			return nil, errors.New("After : should be name project, but found: " + lit)
		}
		tok, lit, start, end = p.scanIgnoreWhitespace()
		if tok == MINUS {
			projectDependency.Sym.Literal += lit
			tok, lit, start, end = p.scanIgnoreWhitespace()
			if tok == NAME {
				projectDependency.Sym.Literal += lit
				projectDependency.Sym.EndPosition = end
			} else {
				projectDependency.Sym.Literal = projectDependency.Sym.Literal[:len(projectDependency.Sym.Literal)]
				p.unscan()
			}
		} else {
			p.unscan()
		}
	} else {
		return nil, errors.New("After Project keyword should be ':, but found: " + lit)
	}

	p.skipQuotaAndMinus()

	dependencies, err := p.startScanDependencies(projectDependency.Sym.Literal)
	if err != nil {
		return nil, err
	}
	projectDependency.ListDependencies = dependencies

	return &projectDependency, nil
}

func (p *Parser) checkContinueScanDependency() bool {
	tok, lit, _, _ := p.scan()
	if tok == MULTY_NEW_LINE || tok == EOF {
		p.unscan()
		return false
	} else if tok == NEW_LINE {
		tok, _, _, _ := p.scan()
		logrus.Debug("Check New Line: ", tok)
		if tok == EOF {
			logrus.Debug("Can not continue, because eof!")
			return false
		}
	}
	logrus.Debug("Check continue scan dependency: ", rune(lit[0]))

	return true
}

func (p *Parser) checkContinueScan() bool {
	tok, _, _, _ := p.scan()
	if tok != EOF {
		p.unscan()
		return true
	}
	return false
}

func (p *Parser) startScanDependencies(projectName string) ([]Depenedency, error) {
	var resultDependencies []Depenedency
	for {
		typeDependencies, eof, err := p.scanTypeDependency()
		if eof {
			logrus.Debug("All dependecies analyed!")
			break
		}
		if err != nil {
			logrus.Debug("TypeDependency not have dependencies: ", err)
			continue
		}
		logrus.Debug("Success scan typeDependencies: ", typeDependencies)

		for {
			dependency, skip, err := p.scanDependency(projectName)

			if err != nil {
				return nil, err
			}

			logrus.Debug("Success scanned dependency: ", dependency)
			p.allDependencies = append(p.allDependencies, dependency)
			if skip {
				logrus.Debug("Skip Dependency!")
			} else {
				resultDependencies = append(resultDependencies, dependency)
			}

			if ok := p.checkContinueScanDependency(); !ok {
				logrus.Debug("Checked fail continue scan dependency:")
				break
			}
		}

		if ok := p.checkContinueScan(); !ok {
			logrus.Error("Can not continue scan dependencies ")
			break
		}
	}

	return resultDependencies, nil
}

func (p *Parser) resetCarettByCount(counter int) {
	for i := 0; i < counter; i++ {
		p.unscan()
	}
}

func (p *Parser) checkProjectNext() bool {
	found_minus := false
	found_new_line := false
	counter_back := 0
	for {
		tok, lit, _, _ := p.scan()
		if tok == NAME && found_minus && found_new_line {
			logrus.Warn("FOUND PROJECT")
			if strings.ToLower(lit) == "project" {
				p.resetCarettByCount(counter_back)
				return true
			} else {
				p.unscan()
			}
		} else if tok == MINUS {
			logrus.Warn("FOUND MINUS")
			found_minus = true
			counter_back++
			continue
		} else if tok == NEW_LINE && found_minus {
			logrus.Warn("FOUND NEW LINE")
			found_new_line = true
			counter_back++
			continue
		} else {
			break
		}
	}
	return false
}

func (p *Parser) scanTypeDependency() (TerminalSymbol, bool, error) {
	p.skipBeforeStartDependencies()
	tok, lit, start, end := p.scanIgnoreWhitespace()
	if tok == EOF {
		return TerminalSymbol{}, true, nil
	}
	if nextProject := p.checkProjectNext(); nextProject {
		logrus.Debug("Scanned next project")
		return TerminalSymbol{}, true, nil
	}
	if tok != NAME {
		return TerminalSymbol{}, false, fmt.Errorf("found %q, expected type dependencies, like allMain, runtime.... Start position: %d; End Position: %d", lit, start, end)
	}

	if ok := p.checkHaveDependencyAndMoveCarretToThat(); !ok {
		return TerminalSymbol{}, false, errors.New("Can not have dependencies")
	}
	return TerminalSymbol{
		Tok:            tok,
		Literal:        lit,
		StartPositioin: start,
		EndPosition:    end,
	}, false, nil
}

func (p *Parser) checkHaveDependencyAndMoveCarretToThat() bool {
	no_dependencies := false
	for {
		tok, lit, start, end := p.scanIgnoreWhitespace()
		if tok == NAME {
			if no_dependencies {
				if strings.ToLower(lit) == "dependencies" {
					return false
				}
				no_dependencies = false
			}
			if strings.ToLower(lit) == "no" {
				no_dependencies = true
			}
		}
		if tok == PLUS || tok == LINE || tok == SLASH {
			p.unscan()
			break
		}
		if tok == EOF {
			return false
		}
		logrus.Debug("Skip terminal sym: ", lit, " Start Position: ", start, " End posoition: ", end)
	}
	return true
}

func (p *Parser) skipBrackets() {
	tok, lit, start, end := p.scanIgnoreWhitespaceAndNotNewLine()
	if tok == LEFT_BRACKET {
		for {
			if tok != RIGHT_BRACKET {
				logrus.Debug("Skip terminal sym: ", lit, " Start Position: ", start, " End posoition: ", end)
				tok, lit, start, end = p.scanIgnoreWhitespaceAndNotNewLine()
				continue
			} else {
				logrus.Debug("Skip terminal sym: ", lit, " Start Position: ", start, " End posoition: ", end)
				return
			}

		}
	}
}

func (p *Parser) scanDependency(projectName string) (Depenedency, bool, error) {
	need_skip_dependency := false
	for {
		tok, _, _, _ := p.scanIgnoreWhitespace()
		if tok == PLUS || tok == MINUS || tok == SLASH {

		} else if tok == LINE {
			need_skip_dependency = true
		} else {
			p.unscan()
			break
		}
	}

	name := TerminalSymbol{
		Tok: DEPENDENCY_NAME,
	}
	version := TerminalSymbol{
		Tok: DEPENDENCY_VERSION,
	}
	changedVersion := &TerminalSymbol{
		Tok: DEPENDENCY_VERSION,
	}

	tok, lit, start, end := p.scanIgnoreLineWhiteSpace()
	name.StartPositioin = start
	name.EndPosition = end
	for {
		if tok == WS {
			break
		}
		if tok == COLON {
			_, beforeLit, _, beforeEnd := tok, lit, start, end
			tok, lit, start, end = p.scanIgnoreLineWhiteSpace()
			if tok == NUMBER {
				name.Literal += beforeLit
				name.EndPosition = beforeEnd
				break
			} else {
				name.Literal += beforeLit + lit
				name.EndPosition = end
				tok, lit, start, end = p.scanIgnoreLineWhiteSpace()
				continue
			}
		}
		name.Literal += lit
		name.EndPosition = end
		tok, lit, start, end = p.scanIgnoreLineWhiteSpace()
	}

	name.Literal = name.Literal[:len(name.Literal)-1]

	if tok == NUMBER {
		version.StartPositioin = start
		for {
			if tok == MINUS || tok == NEW_LINE {
				break
			} else if tok == MULTY_NEW_LINE {
				p.unscan()
				break
			} else if tok == LEFT_BRACKET {
				p.unscan()
				p.skipBrackets()
				changedVersion = nil
				goto skipChangeVersion
			}
			version.EndPosition = end
			version.Literal += lit
			tok, lit, start, end = p.scanIgnoreWhitespaceAndNotNewLine()
		}
	}

	if tok == MINUS {
		changedVersion.StartPositioin = start
		for {
			if tok == NEW_LINE {
				break
			} else if tok == LEFT_BRACKET {
				p.unscan()
				p.skipBrackets()
				tok, lit, _, end = p.scanIgnoreWhitespaceAndNotNewLine()
				p.unscan()
				continue
			} else if tok == MINUS || tok == ARROW {
				tok, lit, _, end = p.scanIgnoreWhitespaceAndNotNewLine()
				continue
			}
			changedVersion.EndPosition = end
			changedVersion.Literal += lit
			tok, lit, _, end = p.scanIgnoreWhitespaceAndNotNewLine()
		}
	}

skipChangeVersion:

	return Depenedency{
		Name:           name,
		Version:        version,
		ChangedVersion: changedVersion,
		Project:        projectName,
	}, need_skip_dependency, nil
}
