package service

func (s *Service) Check() string {
	status := s.repo.Check()
	return status
}
