package config

const (
	ServersCollectionName    string = "game-servers"
	ChallengesCollectionName string = "challenges"
	ServerListMaxCount       int64  = 25
)

// DatabaseConfig is the configuration data structure for the database
type DatabaseConfig struct {
	URL  string
	Name string
}

// DashboardConfig is the configuration data structure for the dashboard
type DashboardConfig struct {
	Port uint16
}

// Config is the main configuration data structure
type Config struct {
	Port                uint16
	Domain              string
	HeartbeatExpiration int32
	ChallengeExpiration int32
	Dashboard           DashboardConfig
	Database            DatabaseConfig
}

// IsValid checks if the configuration instance has all values defined with valid data
func (cfg *Config) IsValid() bool {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return false
	}
	if len(cfg.Domain) == 0 {
		return false
	}
	if cfg.HeartbeatExpiration <= 0 {
		return false
	}
	if cfg.ChallengeExpiration <= 0 {
		return false
	}
	if len(cfg.Database.URL) == 0 {
		return false
	}
	if len(cfg.Database.Name) == 0 {
		return false
	}

	return true
}
