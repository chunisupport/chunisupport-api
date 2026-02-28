package recoverycode

import "testing"

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "有効なリカバリーコード",
			value:   "ABCD-EFGH-IJKL",
			wantErr: false,
		},
		{
			name:    "小文字を含むリカバリーコードも有効",
			value:   "abcd-efgh-ijkl",
			wantErr: false,
		},
		{
			name:    "ハイフン区切りが不足している場合は無効",
			value:   "ABCDEFGHIJKL",
			wantErr: true,
		},
		{
			name:    "空文字は無効",
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			input := tt.value

			// When
			_, err := New(input)

			// Then
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
