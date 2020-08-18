package config

// NewDefaultConfig creates an instance of Config with default values
func NewDefaultConfig() Config {
	return Config{
		Port:                27010,
		Domain:              "localhost",
		HeartbeatExpiration: 300,
		ChallengeExpiration: 30,
		Database: DatabaseConfig{
			URL:  "",
			Name: "",
		},
		Dashboard: DashboardConfig{
			Port: 3000,
		},
	}
}
