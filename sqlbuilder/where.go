package sqlbuilder

func (s *SQL) Where(conditions string) *SQL {
	if err := s.errorCast(); err != nil {
		return s
	}
	s.Conditions = conditions
	return s
}
