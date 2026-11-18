package usecase

import "github.com/chunisupport/chunisupport-api/internal/domain/repository"

// newUserUsecaseForTest はコース統合と目標削除の依存を持たない UserUsecase を作ります。
// 物理削除は常にトランザクションで行うため、記録用のトランザクションマネージャーだけを渡します。
// それらの依存が必要なテストは NewUserUsecase へ直接渡してください。
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
	return NewUserUsecase(db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, songRepo, worldsendChartRepo, masterProvider, friendshipRepo, nil, nil, nil, nil, nil, nil, &userDeletionTransactionManagerStub{}, nil)
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
	return NewUserUsecase(
		db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo,
		songRepo, worldsendChartRepo, masterProvider, friendshipRepo,
		nil, playerLockedSongRepo, overpowerDenominatorProvider, nil, nil,
		nil, nil, nil,
	)
}

// newPlayerDataUsecaseForTest はコース記録を扱わない PlayerDataUsecase を作ります。
// 履歴やコース記録が必要なテストは NewPlayerDataUsecase へ直接渡してください。
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
	return NewPlayerDataUsecase(tm, userRepo, playerRepo, playerRecRepo, worldsendRecRepo, honorRepo, playerDataRepo, lockedRepo, masterCache, nil, nil)
}
