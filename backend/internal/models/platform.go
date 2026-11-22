package models

// JKSummary is a ЖК as the platform operator sees it: details plus counts.
type JKSummary struct {
	JK
	StoragesCount  int `json:"storages_count"`
	AdminsCount    int `json:"admins_count"`
	ResidentsCount int `json:"residents_count"`
}
