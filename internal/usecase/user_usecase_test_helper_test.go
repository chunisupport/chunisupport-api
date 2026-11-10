package usecase

import "github.com/chunisupport/chunisupport-api/internal/domain/repository"

// newUserUsecaseForTest はコース統合と物理削除の依存を持たない UserUsecase を作ります。
// それらの依存が必要なテストは NewUserUsecase へ courseRepo、transactionManager、goalRepo を渡してください。
func newUserUsecaseForTest(
	db repository.Executor,
	userRepo repository.UserRepository,
	playerRepo repository.PlayerRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	worldsendRecordRepo repository.WorldsendRecordRepository,
	songRepo repository.SongRepository,
	worldsendChartRepo repository.WorldsendChartRepository,
	masterProvider userMasterProvider,
	friendshipRepo repository.FriendshipRepository,
) UserUsecase {
	return NewUserUsecase(db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, songRepo, worldsendChartRepo, masterProvider, friendshipRepo, nil, nil, nil)
}

// newUserUsecaseWithOverpowerDenominatorForTest はOVER POWER随時計算だけを有効にした UserUsecase を作ります。
// コース統合と物理削除は対象外のため、それらの依存は渡しません。
func newUserUsecaseWithOverpowerDenominatorForTest(
	db repository.Executor,
	userRepo repository.UserRepository,
	playerRepo repository.PlayerRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	worldsendRecordRepo repository.WorldsendRecordRepository,
	songRepo repository.SongRepository,
	worldsendChartRepo repository.WorldsendChartRepository,
	masterProvider userMasterProvider,
	friendshipRepo repository.FriendshipRepository,
	playerLockedSongRepo repository.PlayerLockedSongRepository,
	overpowerDenominatorProvider repository.OverpowerDenominatorProvider,
) UserUsecase {
	return NewUserUsecaseWithOverpowerDenominator(
		db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo,
		songRepo, worldsendChartRepo, masterProvider, friendshipRepo,
		playerLockedSongRepo, overpowerDenominatorProvider,
		nil, nil, nil,
	)
}

// newPlayerDataUsecaseForTest はコース記録を扱わない PlayerDataUsecase を作ります。
// コース記録が必要なテストは NewPlayerDataUsecase へ courseRepo を渡してください。
func newPlayerDataUsecaseForTest(
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRepo repository.PlayerRepository,
	playerRecRepo repository.PlayerRecordRepository,
	worldsendRecRepo repository.WorldsendRecordRepository,
	honorRepo repository.HonorRepository,
	playerDataRepo repository.PlayerDataRepository,
	lockedRepo repository.PlayerLockedSongRepository,
	masterCache repository.PlayerDataMasterProvider,
) PlayerDataUsecase {
	return NewPlayerDataUsecase(tm, userRepo, playerRepo, playerRecRepo, worldsendRecRepo, honorRepo, playerDataRepo, lockedRepo, masterCache, nil)
}
