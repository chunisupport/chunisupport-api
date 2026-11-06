package songbatch

// ChartUpdateValues は譜面更新判定に必要な値です。nilは未取得、0や空文字は取得済みとして区別します。
type ChartUpdateValues struct {
	Const          float64
	IsConstUnknown bool
	Notes          *int64
	NotesDesigner  *string
}

// ChartUpdateInput は既存値・取り込み値と更新モードをまとめます。Existingがnilなら新規譜面です。
type ChartUpdateInput struct {
	Existing    *ChartUpdateValues
	Incoming    ChartUpdateValues
	MajorUpdate bool
}

// ChartUpdateAction は譜面の保存方針です。実際のSQL操作はインフラ層で行います。
type ChartUpdateAction int

// ChartUpdateResult は保存候補の値と保存方針です。ChartSkipの場合、候補値は保存しません。
type ChartUpdateResult struct {
	Values ChartUpdateValues
	Action ChartUpdateAction
}

// ResolveChartUpdate は既知の定数や補完済みの情報を保護しながら譜面の保存方針を決めます。
// SQL表現やI/Oに依存せず、入力の値とポインタの参照先を変更しません。
func ResolveChartUpdate(input ChartUpdateInput) ChartUpdateResult {
	chart := input.Incoming

	if input.Existing == nil {
		tConst := chart.Const
		tUnknown := chart.IsConstUnknown
		if input.MajorUpdate {
			if chart.Const < majorUpdateLevelThreshold {
				tUnknown = false
			} else {
				tUnknown = true
			}
		}
		return ChartUpdateResult{Values: ChartUpdateValues{Const: tConst, IsConstUnknown: tUnknown, Notes: chart.Notes, NotesDesigner: chart.NotesDesigner}, Action: ChartInsert}
	}

	existing := *input.Existing
	finalConst := existing.Const
	finalUnknown := existing.IsConstUnknown
	finalNotes := existing.Notes
	finalNotesDesigner := existing.NotesDesigner
	shouldUpdate := false

	if input.MajorUpdate {
		if chart.Const < majorUpdateLevelThreshold {
			finalConst = chart.Const
			finalUnknown = false
			if existing.Const != finalConst || existing.IsConstUnknown != finalUnknown {
				shouldUpdate = true
			}
		} else {
			finalUnknown = true
			rangeStart := chart.Const
			rangeEnd := chart.Const + majorUpdateLevelRange
			if existing.Const < rangeStart || existing.Const >= rangeEnd {
				finalConst = chart.Const
				shouldUpdate = true
			}

			if existing.IsConstUnknown != finalUnknown {
				shouldUpdate = true
			}
		}

		if !shouldUpdate {
			if existing.Notes == nil && chart.Notes != nil {
				shouldUpdate = true
			}
			if existing.NotesDesigner == nil && chart.NotesDesigner != nil {
				shouldUpdate = true
			}
		}
	} else {

		if existing.IsConstUnknown && !chart.IsConstUnknown {
			finalConst = chart.Const
			finalUnknown = false
			shouldUpdate = true
		} else {
			// 既知 -> 不明 への変更はブロック
			// それ以外（既知 -> 既知、不明 -> 不明）で、定数が一致し、かつノーツが新しい場合のみ更新
			isBlocked := !existing.IsConstUnknown && chart.IsConstUnknown

			if !isBlocked && existing.Const == chart.Const {
				if existing.Notes == nil && chart.Notes != nil {
					shouldUpdate = true
				}
				if existing.NotesDesigner == nil && chart.NotesDesigner != nil {
					shouldUpdate = true
				}
			}
		}
	}

	// ノーツ数の更新制御: 既存値が存在する場合、NULL値での上書きを防止
	if chart.Notes != nil {
		finalNotes = chart.Notes
	}
	if existing.NotesDesigner == nil && chart.NotesDesigner != nil {
		finalNotesDesigner = chart.NotesDesigner
	}

	values := ChartUpdateValues{Const: finalConst, IsConstUnknown: finalUnknown, Notes: finalNotes, NotesDesigner: finalNotesDesigner}
	if shouldUpdate {
		return ChartUpdateResult{Values: values, Action: ChartUpdate}
	}
	return ChartUpdateResult{Values: values, Action: ChartSkip}
}
