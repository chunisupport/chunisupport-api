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
	// displayLevelBandTenths は公式表示レベル1段階（majorUpdateLevelRange）を0.1単位で表します。
	// 10以上では "13" は 13.0〜13.4、"13+" は 13.5〜13.9 です。比較は0.1刻みの整数で行い、帯の端を浮動小数の誤差で誤判定しません。
	displayLevelBandTenths = int(majorUpdateLevelRange * 10)
	// displayLevelFineConstThresholdTenths は細かい譜面定数が存在する下限を0.1単位で表します。
	// レベル10未満は 9→9.0、9+→9.5 のように、表示レベルの下限1点だけが譜面定数です。
	displayLevelFineConstThresholdTenths = int(majorUpdateLevelThreshold * 10)
)
