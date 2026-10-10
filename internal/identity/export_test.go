package identity

import "time"

func (s *Service) SetNow(now func() time.Time) { s.now = now }

func (s *Service) SetNewCode(newCode func() (string, error)) {
	s.newCode = newCode
}