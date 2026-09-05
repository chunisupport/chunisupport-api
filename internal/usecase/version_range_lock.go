package usecase

import "sync"

// 単一プロセス内の楽曲書込とバージョン削除を直列化し、削除可否確認後の区間への楽曲混入を防ぎます。
var versionRangeMutationMu sync.Mutex
