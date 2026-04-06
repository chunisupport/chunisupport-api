package usecase

import "context"

// FirebaseUserDeleter は Firebase ユーザーの物理削除を抽象化します。
type FirebaseUserDeleter interface {
	DeleteUser(ctx context.Context, uid string) error
}

// FirebaseUserEmailLookup は Firebase UID からメールアドレスを取得するための抽象です。
type FirebaseUserEmailLookup interface {
	LookupEmailsByUIDs(ctx context.Context, uids []string) (map[string]string, error)
}

type noopFirebaseUserDeleter struct{}

type noopFirebaseUserEmailLookup struct{}

func (noopFirebaseUserDeleter) DeleteUser(_ context.Context, _ string) error {
	return nil
}

func (noopFirebaseUserEmailLookup) LookupEmailsByUIDs(_ context.Context, _ []string) (map[string]string, error) {
	return map[string]string{}, nil
}
