package types


type WireguardMetadata struct {
	ID          uint64 `gorm:"primary_key"`
	NodeID      NodeID
	Country     string
	CountryCode string
	City        string
	CityCode    string
	Latitude    float64
	Longitude   float64
}
