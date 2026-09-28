package yaml

func (p *parser) parseFlow(ln line, rest string, col int) (*Node, error) {
	return nil, p.errAt(ln.num, col, "플로우 컬렉션은 아직 지원하지 않는다")
}
