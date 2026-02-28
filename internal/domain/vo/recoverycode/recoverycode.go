package recoverycode

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/chunisupport/chunisupport-api/internal/info"
)

var recoveryCodePattern = regexp.MustCompile(
	fmt.Sprintf("^[A-Za-z0-9]{%d}(-[A-Za-z0-9]{%d}){%d}$", info.RecoveryCodeSegmentLength, info.RecoveryCodeSegmentLength, info.RecoveryCodeSegmentCount-1),
)

type RecoveryCode struct {
	value string
}

func New(value string) (RecoveryCode, error) {
	if !recoveryCodePattern.MatchString(value) {
		return RecoveryCode{}, errors.New("recovery code format is invalid")
	}
	return RecoveryCode{value: value}, nil
}

func (r RecoveryCode) String() string {
	return r.value
}
