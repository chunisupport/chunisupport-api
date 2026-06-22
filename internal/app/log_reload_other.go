//go:build !linux

package app

import "os"

// NotifyLogReload は非Linux環境では何も登録しません。
func NotifyLogReload(ch chan<- os.Signal) {
}
