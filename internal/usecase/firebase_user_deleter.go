package usecase

import "context"

// FirebaseUserDeleter は Firebase ユーザーの物理削除を抽象化します。
type FirebaseUserDeleter interface {
	DeleteUser(ctx context.Context, uid string) error
}

type noopFirebaseUserDeleter struct{}

func (noopFirebaseUserDeleter) DeleteUser(_ context.Context, _ string) error {
	return nil
}
