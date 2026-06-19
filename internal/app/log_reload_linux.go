//go:build linux

package app

import (
	"os"
	"os/signal"
	"syscall"
)

// NotifyLogReload はLinuxでSIGHUPによるログ再オープンを登録します。
func NotifyLogReload(ch chan<- os.Signal) {
	signal.Notify(ch, syscall.SIGHUP)
}
