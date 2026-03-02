package model

// Parse parses the integer Online status into boolean flags
func (s *OnlineStatus) Parse() {
	val := int(s.Online)
	s.IsManagementOnline = ((val >> 0) & 1) == 1
	s.IsBusinessOnline = ((val >> 1) & 1) == 1
	s.IsControlBoardOnline = ((val >> 3) & 1) == 1
	s.IsUSBMode = ((val >> 6) & 1) == 1
	s.IsADBEnabled = ((val >> 8) & 1) == 1
}
