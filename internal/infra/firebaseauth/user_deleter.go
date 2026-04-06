package firebaseauth

import (
	"context"
	"errors"
	"strings"

	firebaseauthsdk "firebase.google.com/go/v4/auth"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

type firebaseUserClient interface {
	DeleteUser(ctx context.Context, uid string) error
	GetUsers(ctx context.Context, identifiers []firebaseauthsdk.UserIdentifier) (*firebaseauthsdk.GetUsersResult, error)
}

type firebaseUserDeleter struct {
	client firebaseUserClient
}

const maxFirebaseGetUsersBatchSize = 100

// NewFirebaseUserDeleter は Firebase Admin SDK の auth.Client を使う FirebaseUserDeleter を生成します。
func NewFirebaseUserDeleter(client *firebaseauthsdk.Client) usecase.FirebaseUserDeleter {
	return &firebaseUserDeleter{client: client}
}

func (d *firebaseUserDeleter) DeleteUser(ctx context.Context, uid string) error {
	if d.client == nil {
		return errors.New("firebase auth client is nil")
	}
	trimmedUID := strings.TrimSpace(uid)
	if trimmedUID == "" {
		return errors.New("firebase uid is empty")
	}

	return d.client.DeleteUser(ctx, trimmedUID)
}

func (d *firebaseUserDeleter) LookupEmailsByUIDs(ctx context.Context, uids []string) (map[string]string, error) {
	if d.client == nil {
		return nil, errors.New("firebase auth client is nil")
	}

	normalizedUIDs := normalizeFirebaseUIDs(uids)
	emailsByUID := make(map[string]string, len(normalizedUIDs))
	if len(normalizedUIDs) == 0 {
		return emailsByUID, nil
	}

	for start := 0; start < len(normalizedUIDs); start += maxFirebaseGetUsersBatchSize {
		end := min(start+maxFirebaseGetUsersBatchSize, len(normalizedUIDs))
		identifiers := make([]firebaseauthsdk.UserIdentifier, 0, end-start)
		for _, uid := range normalizedUIDs[start:end] {
			identifiers = append(identifiers, firebaseauthsdk.UIDIdentifier{UID: uid})
		}

		result, err := d.client.GetUsers(ctx, identifiers)
		if err != nil {
			return nil, err
		}
		for _, user := range result.Users {
			if user == nil {
				continue
			}
			email := strings.TrimSpace(user.Email)
			if user.UID == "" || email == "" {
				continue
			}
			emailsByUID[user.UID] = email
		}
	}

	return emailsByUID, nil
}

func normalizeFirebaseUIDs(uids []string) []string {
	normalized := make([]string, 0, len(uids))
	seen := make(map[string]struct{}, len(uids))
	for _, uid := range uids {
		trimmedUID := strings.TrimSpace(uid)
		if trimmedUID == "" {
			continue
		}
		if _, exists := seen[trimmedUID]; exists {
			continue
		}
		seen[trimmedUID] = struct{}{}
		normalized = append(normalized, trimmedUID)
	}
	return normalized
}

var _ usecase.FirebaseUserEmailLookup = (*firebaseUserDeleter)(nil)
