package sqlbuilder

func (s *SQL) Where(conditions string) *SQL {
	s.Conditions = conditions
	return s
}
