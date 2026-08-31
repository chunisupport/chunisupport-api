package constants

const (
	GoalMaxPerUser                   = 100
	GoalGroupMaxPerUser              = 20
	RecordFilterMaxPerUser           = 100
	RecordFilterNameMaxLength        = 30
	RecordFilterMaxPayloadBytes      = 8 * 1024
	MaxScoreHistoryEntriesPerChart   = 50
	MaxMetricHistoryEntriesPerPlayer = 500
	MaxOfficialRating                = 99.99
	MaxOfficialOverpower             = 999999.99
	MaxOfficialOverpowerPercent      = 100.00
	OfficialMetricDecimalScale       = 100
	OfficialMetricDecimalTolerance   = 1e-7
	PlayerFavoriteSongMaxCount       = 100
	BestSlotMaxCount                 = 30
	NewSlotMaxCount                  = 20
	CandidateSlotMaxCount            = 10
)
