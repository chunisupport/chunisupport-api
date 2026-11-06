package songbatch

const (
	// ChartSkip は今回の候補値を保存しないことを表します。
	ChartSkip ChartUpdateAction = iota
	// ChartInsert は新規譜面の保存を表します。
	ChartInsert
	// ChartUpdate は既存譜面の更新を表します。
	ChartUpdate
)

const (
	// 大型更新では10以上を表示レベルとして扱い、0.5未満の幅で既存定数を保持します。
	majorUpdateLevelThreshold = 10.0
	majorUpdateLevelRange     = 0.5
)
