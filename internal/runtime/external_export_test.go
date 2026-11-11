package runtime

func MailboxLen(s *Session) int {
	s.mb.mu.Lock()
	defer s.mb.mu.Unlock()
	return len(s.mb.q) - s.mb.head
}

func OnActor(s *Session) bool { return s.onActor() }

func ExternalCount(s *Session) int { return len(s.externals) }
