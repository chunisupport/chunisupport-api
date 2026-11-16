package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	api_internal "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

type registeredSPHonor struct {
	ID            int
	ImageFilename string
	ImageURL      string
}

// logRegisteredSPHonors はトランザクションのコミット後に、手動対応が必要な新規SP称号を通知用ログへ出力します。
func logRegisteredSPHonors(ctx context.Context, honors []registeredSPHonor) {
	for _, honor := range honors {
		slog.WarnContext(
			ctx,
			"unknown SP honor registered",
			"event", info.UnknownSPHonorRegisteredEvent,
			"honor_id", honor.ID,
			"image_filename", honor.ImageFilename,
			"image_url", honor.ImageURL,
		)
	}
}

// applyHonors はプレイヤーの称号情報を更新します。
// 既存の称号を削除し、新しい称号をバルクインサートします。
// 称号は最大3つであるため、EnsureHonorのループ内呼び出しによるN+1問題を許容します。
func (us *playerDataUsecase) applyHonors(ctx context.Context, tx repository.Executor, playerID int, honors map[string]PlayerDataHonorPayload, masters *playerDataMaster) ([]api_internal.SkippedRecord, []registeredSPHonor, error) {
	skipped := make([]api_internal.SkippedRecord, 0, 4)
	registered := make([]registeredSPHonor, 0, 1)
	if honors == nil {
		return skipped, registered, nil
	}
	preservedSlots := randomFavoriteHonorSlots(honors)
	if err := us.honorRepo.DeletePlayerHonorsExceptSlots(ctx, tx, playerID, preservedSlots); err != nil {
		return skipped, registered, err
	}

	assignments := make([]repository.HonorAssignment, 0, len(honors))

	for slotKey, honor := range honors {
		slotKey = strings.TrimSpace(slotKey)
		if slotKey == "" {
			continue
		}
		slot, convErr := strconv.Atoi(slotKey)
		if convErr != nil {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     fmt.Sprintf("invalid slot %s", slotKey),
				Details:    convErr.Error(),
			})
			continue
		}
		if slot < 1 || slot > 3 {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     fmt.Sprintf("slot out of range: %d", slot),
				Details:    fmt.Sprintf("slot=%d, title=%s", slot, honor.Title),
			})
			continue
		}
		if honor.Title == info.RandomFavoriteHonorTitle {
			continue
		}

		honorTypeKey := strings.ToLower(strings.TrimSpace(honor.Class))
		typeItem, ok := masters.HonorTypes[honorTypeKey]
		if !ok {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     fmt.Sprintf("honor_type not found: %s", honorTypeKey),
				Details:    fmt.Sprintf("slot=%d, title=%s", slot, honor.Title),
			})
			continue
		}

		if honorTypeKey == "sp" && (honor.Img == nil || strings.TrimSpace(*honor.Img) == "") {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     "sp honor image_url is required",
				Details:    fmt.Sprintf("slot=%d, title=%s", slot, honor.Title),
			})
			continue
		}

		honorTitle := honor.Title
		imageURL := honor.Img
		if honorTypeKey == "sp" {
			honorTitle = honorImageFilename(*imageURL)
		} else {
			// 通常称号の画像は既知の対応表で解決できるため、未判明のSP称号だけを保存する。
			imageURL = nil
		}

		ensureResult, err := us.honorRepo.EnsureHonor(ctx, tx, honorTitle, typeItem.ID, imageURL)
		if err != nil {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     "failed to create honor",
				Details:    fmt.Sprintf("slot=%d, title=%s, error=%s", slot, honor.Title, err.Error()),
			})
			continue
		}
		if honorTypeKey == "sp" && ensureResult.ImageURLRegistered {
			registered = append(registered, registeredSPHonor{
				ID:            ensureResult.ID,
				ImageFilename: honorTitle,
				ImageURL:      *imageURL,
			})
		}

		assignments = append(assignments, repository.HonorAssignment{
			PlayerID: playerID,
			HonorID:  ensureResult.ID,
			Slot:     slot,
		})
	}

	if len(assignments) > 0 {
		if err := us.honorRepo.BulkAssignHonors(ctx, tx, assignments); err != nil {
			for _, a := range assignments {
				skipped = append(skipped, api_internal.SkippedRecord{
					RecordType: "honor",
					Reason:     "failed to insert player_honor (bulk)",
					Details:    fmt.Sprintf("slot=%d, honor_id=%d, error=%s", a.Slot, a.HonorID, err.Error()),
				})
			}
		}
	}

	return skipped, registered, nil
}

// honorImageFilename はSP称号を一意に識別するため、画像URLからファイル名を取り出します。
func honorImageFilename(imageURL string) string {
	parsedURL, err := url.Parse(strings.TrimSpace(imageURL))
	if err != nil {
		return strings.TrimSpace(imageURL)
	}
	return path.Base(parsedURL.EscapedPath())
}

// randomFavoriteHonorSlots はCHUNITHM-NET側で毎回ランダム選択される称号のスロットを返します。
// この表示用テキストを称号として保存せず、現在の割り当てを維持するために使用します。
func randomFavoriteHonorSlots(honors map[string]PlayerDataHonorPayload) []int {
	slots := make([]int, 0, 3)
	for slotKey, honor := range honors {
		if honor.Title != info.RandomFavoriteHonorTitle {
			continue
		}
		slot, err := strconv.Atoi(strings.TrimSpace(slotKey))
		if err == nil && slot >= 1 && slot <= 3 {
			slots = append(slots, slot)
		}
	}
	return slots
}
