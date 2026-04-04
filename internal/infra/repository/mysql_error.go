package repository

import (
	"errors"
	"fmt"
	"strings"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/go-sql-driver/mysql"
)

const mysqlDuplicateEntryErrorNumber uint16 = 1062

func wrapFirebaseUIDDuplicateError(err error) error {
	if !isMySQLDuplicateEntryForKey(err, "uk_users_firebase_uid") && !isMySQLDuplicateEntryForKey(err, "firebase_uid") {
		return err
	}

	return fmt.Errorf("%w: %v", domainrepo.ErrFirebaseUIDAlreadyLinked, err)
}

func wrapUsernameDuplicateError(err error) error {
	if !isMySQLDuplicateEntryForKey(err, "username") {
		return err
	}

	return fmt.Errorf("%w: %v", domainrepo.ErrDuplicateUsername, err)
}

func isMySQLDuplicateEntryForKey(err error, key string) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}

	return mysqlErr.Number == mysqlDuplicateEntryErrorNumber && strings.Contains(mysqlErr.Message, key)
}
