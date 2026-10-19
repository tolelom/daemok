package check

func (c *checker) requirements() {
	for _, r := range c.m.Requirements {
		sats := c.satisfiers[r.ID]
		if r.Status == "active" {
			if len(sats) == 0 {
				c.warnf(r.Pos, "충족하는 설계 요소가 없는 요구사항: %s", r.ID)
			}
			continue
		}
		for _, s := range sats {
			c.warnf(s.ref.Pos, "%s 요구사항에 연결된 설계 요소: %s → %s", r.Status, s.id, r.ID)
		}
	}
}
