package entity

import (
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

// AppAnnouncementKind は利用者がお知らせを選択したときの動作の種別です。
type AppAnnouncementKind string

const (
	// AppAnnouncementKindMessage は選択するとダイアログで本文を表示するお知らせです。
	AppAnnouncementKindMessage AppAnnouncementKind = "message"
	// AppAnnouncementKindLink はバナー自体がリンクになっており、選択すると target_url へ遷移するお知らせです。
	AppAnnouncementKindLink AppAnnouncementKind = "link"
)

// 列の型（VARCHAR(200)・TEXT・VARCHAR(2048)・TINYINT UNSIGNED）に収まる上限です。
// 管理画面から登録できるため、DBエラーではなく入力エラーとして先に拒否します。
const (
	AppAnnouncementTitleMaxLength     = 200
	AppAnnouncementBodyMaxBytes       = 65535
	AppAnnouncementTargetURLMaxLength = 2048
	AppAnnouncementRevisionMax        = 255
)

var (
	ErrAppAnnouncementKindInvalid         = errors.New("app announcement kind is invalid")
	ErrAppAnnouncementTitleRequired       = errors.New("app announcement title is required")
	ErrAppAnnouncementTitleTooLong        = errors.New("app announcement title is too long")
	ErrAppAnnouncementTargetURLTooLong    = errors.New("app announcement target url is too long")
	ErrAppAnnouncementBodyRequired        = errors.New("message app announcement requires a body")
	ErrAppAnnouncementBodyTooLong         = errors.New("app announcement body is too long")
	ErrAppAnnouncementTargetURLInvalid    = errors.New("app announcement target url must be an app path or an http(s) url")
	ErrAppAnnouncementBodyMustBeNull      = errors.New("link app announcement must not have a body")
	ErrAppAnnouncementTargetURLRequired   = errors.New("link app announcement requires a target url")
	ErrAppAnnouncementTargetURLMustBeNull = errors.New("message app announcement must not have a target url")
	ErrAppAnnouncementPeriodInvalid       = errors.New("app announcement starts_at must be before expires_at")
	ErrAppAnnouncementRevisionInvalid     = errors.New("app announcement revision must be between 1 and 255")
)

// AppAnnouncement はプロフィールカード上部に表示する期間限定のお知らせです。
// 本文型は Body、リンク型は TargetURL だけを持ちます。
type AppAnnouncement struct {
	DisplayID displayid.DisplayID
	Kind      AppAnnouncementKind
	Title     string
	Body      *string
	TargetURL *string
	StartsAt  time.Time
	ExpiresAt time.Time
	// Revision は閉じた利用者にも再表示させたいときに運営が増やす値です。
	Revision int
}

// RestoreAppAnnouncement はお知らせを検証して生成・復元します。
// 管理画面に加えてSQLでも登録できるため、復元時も種別と項目の組み合わせを必ず検証します。
// 種別の追加で組み合わせが変わってもマイグレーションを不要にするため、組み合わせはDB制約にせずここだけで検証します。
func RestoreAppAnnouncement(
	displayID string,
	kind string,
	title string,
	body *string,
	targetURL *string,
	startsAt time.Time,
	expiresAt time.Time,
	revision int,
) (*AppAnnouncement, error) {
	validatedID, err := displayid.NewDisplayID(displayID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(title) == "" {
		return nil, ErrAppAnnouncementTitleRequired
	}
	if utf8.RuneCountInString(title) > AppAnnouncementTitleMaxLength {
		return nil, ErrAppAnnouncementTitleTooLong
	}
	// DATETIME 列は秒未満を四捨五入するため、先に切り捨てて保存値とレスポンスを揃えます。
	startsAt = startsAt.Truncate(time.Second)
	expiresAt = expiresAt.Truncate(time.Second)
	// 開始日時の指定漏れ（ゼロ値）を、即時に配信が始まるお知らせとして受け入れないようにします。
	if startsAt.IsZero() || !startsAt.Before(expiresAt) {
		return nil, ErrAppAnnouncementPeriodInvalid
	}
	if revision <= 0 || revision > AppAnnouncementRevisionMax {
		return nil, ErrAppAnnouncementRevisionInvalid
	}

	announcement := &AppAnnouncement{
		DisplayID: validatedID,
		Kind:      AppAnnouncementKind(kind),
		Title:     title,
		StartsAt:  startsAt,
		ExpiresAt: expiresAt,
		Revision:  revision,
	}
	switch announcement.Kind {
	case AppAnnouncementKindMessage:
		if body == nil || strings.TrimSpace(*body) == "" {
			return nil, ErrAppAnnouncementBodyRequired
		}
		if len(*body) > AppAnnouncementBodyMaxBytes {
			return nil, ErrAppAnnouncementBodyTooLong
		}
		if targetURL != nil {
			return nil, ErrAppAnnouncementTargetURLMustBeNull
		}
		announcement.Body = new(*body)
	case AppAnnouncementKindLink:
		if body != nil {
			return nil, ErrAppAnnouncementBodyMustBeNull
		}
		if targetURL == nil || *targetURL == "" {
			return nil, ErrAppAnnouncementTargetURLRequired
		}
		if utf8.RuneCountInString(*targetURL) > AppAnnouncementTargetURLMaxLength {
			return nil, ErrAppAnnouncementTargetURLTooLong
		}
		if !isAppAnnouncementTargetURL(*targetURL) {
			return nil, ErrAppAnnouncementTargetURLInvalid
		}
		// 書き手はADMINに限られるため、リンク先の内容（許可リストなど）は検証しません。
		announcement.TargetURL = new(*targetURL)
	default:
		return nil, ErrAppAnnouncementKindInvalid
	}
	return announcement, nil
}

// isAppAnnouncementTargetURL は遷移先がアプリ内パスか http(s) のURLかを返します。
// フロントエンドが遷移させない値を登録すると、黙って表示されないお知らせになるため拒否します。
func isAppAnnouncementTargetURL(value string) bool {
	if strings.HasPrefix(value, "/") {
		return !strings.HasPrefix(value, "//") && !strings.ContainsAny(value, "\\ \t\r\n")
	}
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != ""
}

// IsActiveAt は now が表示期間 [StartsAt, ExpiresAt) に含まれるかを返します。
func (a *AppAnnouncement) IsActiveAt(now time.Time) bool {
	return !now.Before(a.StartsAt) && now.Before(a.ExpiresAt)
}
