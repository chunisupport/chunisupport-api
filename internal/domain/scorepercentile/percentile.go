// Package scorepercentile は譜面・コースのスコア分布から上位%（0.01%〜100.00%）を算出・逆引きするドメインモデルを提供します。
package scorepercentile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sort"
)

const (
	// thresholdFormatVersion はバイナリ形式のバージョンです。
	// 先頭1バイトに書き込み、将来の形式変更時に安全に判別できるようにします。
	thresholdFormatVersion byte = 1
	thresholdEntryBytes         = 6

	// minTopPercentBP と maxTopPercentBP は上位%のベーシスポイント（0.01% = 1）の下限と上限です。
	minTopPercentBP uint16 = 1
	maxTopPercentBP uint16 = 10000
)

var (
	// ErrInvalidThresholdData は永続化されたスコア境界バイナリが不正であることを表します。
	ErrInvalidThresholdData = errors.New("invalid score percentile threshold data")
)

// Threshold は「MinScore 以上なら上位 TopPercentBP / 100 %」を表す境界1件です。
type Threshold struct {
	MinScore     uint32
	TopPercentBP uint16
}

// Entry は譜面またはコース1件分の上位%境界テーブルです。
type Entry struct {
	TargetID   int
	Thresholds []Threshold
}

// BuildThresholds は対象譜面・コースの全スコアから、任意のスコアに対して誤差ゼロで小数点以下2桁の上位%を引ける境界列を生成します。
// 上位%は競技順位ベース (自分より厳密に高いスコアの人数 + 1) / 母集団人数 * 100 を小数第3位で四捨五入し、[0.01, 100.00] にクランプします。
// 同じ 0.01% 値になる隣接スコア区間は1つに統合するため、要素数は最大でも 10,000 件に収まります。
func BuildThresholds(scores []int) []Threshold {
	total := len(scores)
	if total == 0 {
		return nil
	}

	sorted := make([]uint32, total)
	for i, s := range scores {
		if s > 0 {
			sorted[i] = uint32(s) // #nosec G115 -- スコアは0～1,010,000で、正の値だけを変換するためuint32の範囲内
		}
	}
	slices.Sort(sorted)

	thresholds := make([]Threshold, 0, min(total+1, int(maxTopPercentBP)))
	appendIfChanged := func(minScore uint32, bp uint16) {
		if len(thresholds) > 0 && thresholds[len(thresholds)-1].TopPercentBP == bp {
			return
		}
		thresholds = append(thresholds, Threshold{MinScore: minScore, TopPercentBP: bp})
	}

	// 母集団の最低点が 0 より大きい場合、最低点未満の新規スコアは上限の 100.00% にクランプされる。
	if sorted[0] > 0 {
		appendIfChanged(0, maxTopPercentBP)
	}
	for i := 0; i < total; {
		v := sorted[i]
		j := i + 1
		for j < total && sorted[j] == v {
			j++
		}
		// 昇順ソート済みのため、同点区間 [i, j) より後ろの件数 (total - j) が v より厳密に高いスコアの人数になる。
		appendIfChanged(v, calcTopPercentBP(total-j, total))
		i = j
	}

	return thresholds
}

// calcTopPercentBP は (higher + 1) / total * 100 を小数点以下2桁に四捨五入したベーシスポイント（1〜10000）を返します。
// 整数演算だけで計算することで、浮動小数点の丸め誤差を排除します。
func calcTopPercentBP(higher, total int) uint16 {
	// 呼び出し元は 0 <= higher < total を保証するため、負の値をuint64へ変換することはない。
	numerator := uint64(higher+1)*uint64(maxTopPercentBP) + uint64(total)/2 // #nosec G115 -- higherとtotalは非負
	raw := numerator / uint64(total)                                        // #nosec G115 -- totalは正
	if raw < uint64(minTopPercentBP) {
		return minTopPercentBP
	}
	if raw > uint64(maxTopPercentBP) {
		return maxTopPercentBP
	}
	return uint16(raw)
}

// EncodeThresholds は境界列を先頭バージョン付きのリトルエンディアン固定長バイナリへ変換します。
func EncodeThresholds(thresholds []Threshold) []byte {
	buf := make([]byte, 1+len(thresholds)*thresholdEntryBytes)
	buf[0] = thresholdFormatVersion
	offset := 1
	for _, t := range thresholds {
		binary.LittleEndian.PutUint32(buf[offset:offset+4], t.MinScore)
		binary.LittleEndian.PutUint16(buf[offset+4:offset+6], t.TopPercentBP)
		offset += thresholdEntryBytes
	}
	return buf
}

// DecodeThresholds は永続化されたバイナリを境界列へ復元し、形式の整合性を検証します。
func DecodeThresholds(data []byte) ([]Threshold, error) {
	if len(data) < 1+thresholdEntryBytes {
		return nil, fmt.Errorf("%w: data too short (%d bytes)", ErrInvalidThresholdData, len(data))
	}
	if data[0] != thresholdFormatVersion {
		return nil, fmt.Errorf("%w: unsupported version %d", ErrInvalidThresholdData, data[0])
	}
	payloadLen := len(data) - 1
	if payloadLen%thresholdEntryBytes != 0 {
		return nil, fmt.Errorf("%w: invalid payload length %d", ErrInvalidThresholdData, payloadLen)
	}

	count := payloadLen / thresholdEntryBytes
	// 境界は上位%の値ごとに1件のため最大10,000件。超える場合は破損データとして扱い、逆引き表の件数をuint16に収める。
	if count > int(maxTopPercentBP) {
		return nil, fmt.Errorf("%w: too many thresholds (%d)", ErrInvalidThresholdData, count)
	}
	thresholds := make([]Threshold, count)
	offset := 1
	for i := range count {
		minScore := binary.LittleEndian.Uint32(data[offset : offset+4])
		bp := binary.LittleEndian.Uint16(data[offset+4 : offset+6])
		if bp < minTopPercentBP || bp > maxTopPercentBP {
			return nil, fmt.Errorf("%w: top_percent_bp out of range: %d", ErrInvalidThresholdData, bp)
		}
		if i == 0 && minScore != 0 {
			return nil, fmt.Errorf("%w: first threshold min_score must be 0, got %d", ErrInvalidThresholdData, minScore)
		}
		if i > 0 && minScore <= thresholds[i-1].MinScore {
			return nil, fmt.Errorf("%w: min_score is not strictly increasing at index %d", ErrInvalidThresholdData, i)
		}
		thresholds[i] = Threshold{MinScore: minScore, TopPercentBP: bp}
		offset += thresholdEntryBytes
	}
	return thresholds, nil
}

type rangeRef struct {
	offset uint32
	length uint16
}

// lookupTable は全譜面分の境界配列をポインタのないフラットなスライスへ格納し、GC負荷を抑えて高速に逆引きします。
type lookupTable struct {
	minScores []uint32
	bps       []uint16
	index     map[int]rangeRef
}

func buildLookupTable(entries []Entry) lookupTable {
	totalThresholds := 0
	for _, e := range entries {
		totalThresholds += len(e.Thresholds)
	}
	t := lookupTable{
		minScores: make([]uint32, 0, totalThresholds),
		bps:       make([]uint16, 0, totalThresholds),
		index:     make(map[int]rangeRef, len(entries)),
	}
	for _, e := range entries {
		// 空の境界列を登録すると lookup で範囲外参照になるため、索引に載せない。
		if len(e.Thresholds) == 0 {
			continue
		}
		offset := uint32(len(t.minScores)) // #nosec G115 -- 1件あたり最大10,000件で、全譜面・コース分でもuint32の範囲内
		for _, th := range e.Thresholds {
			t.minScores = append(t.minScores, th.MinScore)
			t.bps = append(t.bps, th.TopPercentBP)
		}
		t.index[e.TargetID] = rangeRef{
			offset: offset,
			length: uint16(len(e.Thresholds)), // #nosec G115 -- BuildThresholdsとDecodeThresholdsが最大10,000件を保証する
		}
	}
	return t
}

func (t *lookupTable) lookup(targetID int, score uint32) *float64 {
	ref, ok := t.index[targetID]
	if !ok {
		return nil
	}
	end := ref.offset + uint32(ref.length)
	scores := t.minScores[ref.offset:end]
	bps := t.bps[ref.offset:end]

	// score 以下の最大の minScore を二分探索で求める。
	// BuildThresholds・DecodeThresholds 経由なら scores[0] == 0 だが、NewSnapshot へ直接渡された境界列にも備えて先頭へ寄せる。
	idx := sort.Search(len(scores), func(i int) bool {
		return scores[i] > score
	}) - 1
	if idx < 0 {
		idx = 0
	}
	value := float64(bps[idx]) / 100.0
	return &value
}

// Snapshot は同一世代の通常譜面・WORLD'S END譜面・コースの上位%境界をまとめた不変スナップショットです。
// 読み取り専用のため、複数ゴルーチンからロックなしで安全に参照できます。
type Snapshot struct {
	revision        uint64
	charts          lookupTable
	worldsendCharts lookupTable
	courses         lookupTable
}

// NewSnapshot は世代番号と各境界エントリから不変スナップショットを構築します。
func NewSnapshot(revision uint64, charts, worldsendCharts, courses []Entry) *Snapshot {
	return &Snapshot{
		revision:        revision,
		charts:          buildLookupTable(charts),
		worldsendCharts: buildLookupTable(worldsendCharts),
		courses:         buildLookupTable(courses),
	}
}

// Revision はスナップショットを構築したときのDB上の統計変更世代を返します。
func (s *Snapshot) Revision() uint64 {
	if s == nil {
		return 0
	}
	return s.revision
}

// LookupChart は通常譜面IDとスコアから上位%を返します。統計が存在しない場合は nil です。
func (s *Snapshot) LookupChart(chartID int, score uint32) *float64 {
	if s == nil {
		return nil
	}
	return s.charts.lookup(chartID, score)
}

// LookupWorldsendChart は WORLD'S END 譜面IDとスコアから上位%を返します。統計が存在しない場合は nil です。
func (s *Snapshot) LookupWorldsendChart(worldsendChartID int, score uint32) *float64 {
	if s == nil {
		return nil
	}
	return s.worldsendCharts.lookup(worldsendChartID, score)
}

// LookupCourse はコースIDとスコアから上位%を返します。統計が存在しない場合は nil です。
func (s *Snapshot) LookupCourse(courseID int, score uint32) *float64 {
	if s == nil {
		return nil
	}
	return s.courses.lookup(courseID, score)
}
