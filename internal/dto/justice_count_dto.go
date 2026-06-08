package dto

import (
	"math"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
)

const comboLampAllJustice = 3

func calcJusticeCount(score uint32, comboLampID int, notesCount *int) *int {
	if score == constants.TheoreticalScore {
		justiceCount := 0
		return &justiceCount
	}
	if comboLampID != comboLampAllJustice || notesCount == nil {
		return nil
	}

	diff := constants.TheoreticalScore - int(score)
	justiceCount := int(math.Round(float64(*notesCount) * float64(diff) / 10000))
	return &justiceCount
}
