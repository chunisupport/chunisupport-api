package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

var (
	ErrInvalidFriendRequest        = errors.New("invalid friend request")
	ErrFriendshipAlreadyExists     = errors.New("friendship already exists")
	ErrFriendRequestNotFound       = errors.New("friend request not found")
	ErrFriendshipLimitExceeded     = errors.New("friendship limit exceeded")
	errFriendshipNilDB             = errors.New("database executor is nil")
	errFriendshipNilTM             = errors.New("transaction manager is nil")
	errFriendshipNilUserRepo       = errors.New("user repository is nil")
	errFriendshipNilFriendshipRepo = errors.New("friendship repository is nil")
)

type friendshipUsecase struct {
	db             repository.Executor
	tm             TransactionManager
	userRepo       repository.UserRepository
	friendshipRepo repository.FriendshipRepository
	now            func() time.Time
}

func NewFriendshipUsecase(
	db repository.Executor,
	tm TransactionManager,
	userRepo repository.UserRepository,
	friendshipRepo repository.FriendshipRepository,
) (FriendshipUsecase, error) {
	if db == nil {
		return nil, errFriendshipNilDB
	}
	if tm == nil {
		return nil, errFriendshipNilTM
	}
	if userRepo == nil {
		return nil, errFriendshipNilUserRepo
	}
	if friendshipRepo == nil {
		return nil, errFriendshipNilFriendshipRepo
	}
	return &friendshipUsecase{
		db:             db,
		tm:             tm,
		userRepo:       userRepo,
		friendshipRepo: friendshipRepo,
		now:            time.Now,
	}, nil
}

func (u *friendshipUsecase) SendRequest(ctx context.Context, userID int, username string) error {
	target, err := u.userRepo.FindByUsername(ctx, u.db, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		slog.Error("failed to find friend target by username", "username", username, "error", err)
		return err
	}
	if target == nil {
		return ErrUserNotFound
	}
	if target.ID == userID {
		return ErrInvalidFriendRequest
	}

	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := u.lockUsers(ctx, tx, userID, target.ID); err != nil {
			return err
		}

		outgoing, err := u.friendshipRepo.Find(ctx, tx, userID, target.ID)
		if err != nil {
			return err
		}
		if outgoing != nil {
			return ErrFriendshipAlreadyExists
		}

		if err := u.ensureOutgoingSlot(ctx, tx, userID); err != nil {
			return err
		}

		incoming, err := u.friendshipRepo.Find(ctx, tx, target.ID, userID)
		if err != nil {
			return err
		}
		now := u.now()
		if incoming != nil {
			if incoming.IsPending() {
				return u.acceptMutual(ctx, tx, incoming, userID, target.ID, now)
			}
			return ErrFriendshipAlreadyExists
		}

		request, err := entity.NewFriendRequest(userID, target.ID, now)
		if err != nil {
			return ErrInvalidFriendRequest
		}
		return u.friendshipRepo.Save(ctx, tx, request)
	})
}

func (u *friendshipUsecase) ListFriends(ctx context.Context, userID int) ([]*FriendshipUserOutput, error) {
	rows, err := u.friendshipRepo.ListFriends(ctx, u.db, userID)
	if err != nil {
		return nil, err
	}
	return toFriendshipUserOutputs(rows), nil
}

func (u *friendshipUsecase) ListReceivedRequests(ctx context.Context, userID int) ([]*FriendshipUserOutput, error) {
	rows, err := u.friendshipRepo.ListReceivedRequests(ctx, u.db, userID)
	if err != nil {
		return nil, err
	}
	return toFriendshipUserOutputs(rows), nil
}

func (u *friendshipUsecase) ListSentRequests(ctx context.Context, userID int) ([]*FriendshipUserOutput, error) {
	rows, err := u.friendshipRepo.ListSentRequests(ctx, u.db, userID)
	if err != nil {
		return nil, err
	}
	return toFriendshipUserOutputs(rows), nil
}

func (u *friendshipUsecase) AcceptRequest(ctx context.Context, userID int, requesterID int) error {
	if userID == requesterID {
		return ErrInvalidFriendRequest
	}

	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := u.lockUsers(ctx, tx, userID, requesterID); err != nil {
			return err
		}

		if err := u.ensureOutgoingSlot(ctx, tx, userID); err != nil {
			return err
		}

		incoming, err := u.friendshipRepo.Find(ctx, tx, requesterID, userID)
		if err != nil {
			return err
		}
		if incoming == nil || !incoming.IsPending() {
			return ErrFriendRequestNotFound
		}

		now := u.now()
		return u.acceptMutual(ctx, tx, incoming, userID, requesterID, now)
	})
}

func (u *friendshipUsecase) RejectRequest(ctx context.Context, userID int, requesterID int) error {
	if userID == requesterID {
		return ErrInvalidFriendRequest
	}

	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := u.lockUsers(ctx, tx, userID, requesterID); err != nil {
			return err
		}

		incoming, err := u.friendshipRepo.Find(ctx, tx, requesterID, userID)
		if err != nil {
			return err
		}
		if incoming == nil || !incoming.IsPending() {
			return ErrFriendRequestNotFound
		}
		return u.friendshipRepo.DeletePending(ctx, tx, requesterID, userID)
	})
}

func (u *friendshipUsecase) Remove(ctx context.Context, userID int, friendUserID int) error {
	if userID == friendUserID {
		return ErrInvalidFriendRequest
	}
	return u.friendshipRepo.DeletePair(ctx, u.db, userID, friendUserID)
}

func (u *friendshipUsecase) ensureOutgoingSlot(ctx context.Context, exec repository.Executor, userID int) error {
	count, err := u.friendshipRepo.CountOutgoingActive(ctx, exec, userID)
	if err != nil {
		return err
	}
	if count >= info.FriendshipMaxOutgoingActive {
		return ErrFriendshipLimitExceeded
	}
	return nil
}

func (u *friendshipUsecase) lockUser(ctx context.Context, exec repository.Executor, userID int) error {
	user, err := u.userRepo.FindByIDForUpdate(ctx, exec, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	return nil
}

// lockUsers はフレンド関係の両端を固定順でロックし、相互申請や承認・拒否の競合を防ぎます。
func (u *friendshipUsecase) lockUsers(ctx context.Context, exec repository.Executor, userID int, otherUserID int) error {
	firstID := min(userID, otherUserID)
	secondID := max(userID, otherUserID)
	if err := u.lockUser(ctx, exec, firstID); err != nil {
		return err
	}
	if secondID == firstID {
		return nil
	}
	return u.lockUser(ctx, exec, secondID)
}

func (u *friendshipUsecase) acceptMutual(ctx context.Context, exec repository.Executor, incoming *entity.Friendship, accepterID int, requesterID int, acceptedAt time.Time) error {
	if err := incoming.Accept(acceptedAt); err != nil {
		return ErrInvalidFriendRequest
	}
	if err := u.friendshipRepo.Save(ctx, exec, incoming); err != nil {
		return err
	}
	outgoing, err := entity.NewAcceptedFriendship(accepterID, requesterID, incoming.RequestedAt, acceptedAt)
	if err != nil {
		return ErrInvalidFriendRequest
	}
	return u.friendshipRepo.Save(ctx, exec, outgoing)
}

func toFriendshipUserOutputs(rows []*repository.FriendshipWithUserSummary) []*FriendshipUserOutput {
	items := make([]*FriendshipUserOutput, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.Friendship == nil || row.User == nil {
			continue
		}
		items = append(items, &FriendshipUserOutput{
			UserID:      row.User.UserID,
			Username:    row.User.Username,
			PlayerLevel: row.User.PlayerLevel,
			PlayerName:  row.User.PlayerName,
			Rating:      row.User.Rating,
			RequestedAt: row.Friendship.RequestedAt,
			AcceptedAt:  row.Friendship.AcceptedAt,
		})
	}
	return items
}
