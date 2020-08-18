package valve

// Region defines a part of the world where servers are located
type Region uint8

const (
	// USEastCoast : United States - East Coast
	USEastCoast  uint8 = 0x00
	USWestCoast  uint8 = 0x01
	SouthAmerica uint8 = 0x02
	Europe       uint8 = 0x03
	Asia         uint8 = 0x04
	Australia    uint8 = 0x05
	MiddleEast   uint8 = 0x06
	Africa       uint8 = 0x07
	AllRegions   uint8 = 0xFF
)

// OperatingSystem defines which platform the server is launched from
type OperatingSystem string

const (
	// Windows : Microsoft Windows
	Windows string = "w"
	Linux   string = "l"
	OSX     string = "o"
)

const (
	RequestServerListHeader byte = 0x31
	RequestJoinHeader       byte = 0x71
	RequestQuitHeader       byte = 0x62
	RequestChallengeHeader  byte = 0x30
)

var (
	ServerListHeader []byte = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x66, 0x0A}
	ChallengeHeader  []byte = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x73, 0x0A}
	QuitHeader       []byte = []byte{0x62, 0x0A, 0x00}
)
