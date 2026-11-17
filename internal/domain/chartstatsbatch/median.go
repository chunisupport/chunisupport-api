package chartstatsbatch

import "slices"

// averageAndMedian はスコアの平均と中央値を返します。0件の場合はどちらも nil です。
// 中央値は昇順に並べ、奇数件なら中央の値、偶数件なら中央2件の算術平均です。scores は並べ替えられます。
func averageAndMedian(total int64, scores []int) (average, median *float64) {
	count := len(scores)
	if count == 0 {
		return nil, nil
	}
	averageValue := float64(total) / float64(count)
	slices.Sort(scores)
	medianValue := float64(scores[count/2])
	if count%2 == 0 {
		medianValue = (float64(scores[count/2-1]) + medianValue) / 2
	}
	return &averageValue, &medianValue
}
