package masterdata

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
)

func TestCache_GetAccountTypeNameByID(t *testing.T) {
	tests := []struct {
		name     string
		cache    *Cache
		id       int
		expected string
	}{
		{
			name: "正常系: 存在するIDの場合は名前を返す",
			cache: &Cache{
				AccountTypes: map[string]master.AccountType{
					"PLAYER": {ID: 1, Name: "PLAYER"},
					"EDITOR": {ID: 2, Name: "EDITOR"},
					"ADMIN":  {ID: 3, Name: "ADMIN"},
				},
			},
			id:       1,
			expected: "PLAYER",
		},
		{
			name: "正常系: ADMIN IDの場合",
			cache: &Cache{
				AccountTypes: map[string]master.AccountType{
					"PLAYER": {ID: 1, Name: "PLAYER"},
					"EDITOR": {ID: 2, Name: "EDITOR"},
					"ADMIN":  {ID: 3, Name: "ADMIN"},
				},
			},
			id:       3,
			expected: "ADMIN",
		},
		{
			name: "異常系: 存在しないIDの場合はUNKNOWNを返す",
			cache: &Cache{
				AccountTypes: map[string]master.AccountType{
					"PLAYER": {ID: 1, Name: "PLAYER"},
				},
			},
			id:       999,
			expected: "UNKNOWN",
		},
		{
			name:     "異常系: キャッシュがnilの場合はUNKNOWNを返す",
			cache:    nil,
			id:       1,
			expected: "UNKNOWN",
		},
		{
			name: "異常系: AccountTypesが空の場合はUNKNOWNを返す",
			cache: &Cache{
				AccountTypes: map[string]master.AccountType{},
			},
			id:       1,
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.GetAccountTypeNameByID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCache_GetClassEmblemNameByID(t *testing.T) {
	tests := []struct {
		name     string
		cache    *Cache
		id       int
		expected string
	}{
		{
			name: "正常系: 存在するIDの場合は名前を返す",
			cache: &Cache{
				ClassEmblems: map[string]master.ClassEmblem{
					"Bronze": {ID: 1, Name: "Bronze"},
					"Silver": {ID: 2, Name: "Silver"},
				},
			},
			id:       1,
			expected: "Bronze",
		},
		{
			name: "異常系: 存在しないIDの場合は空文字を返す",
			cache: &Cache{
				ClassEmblems: map[string]master.ClassEmblem{
					"Bronze": {ID: 1, Name: "Bronze"},
				},
			},
			id:       999,
			expected: "",
		},
		{
			name:     "異常系: キャッシュがnilの場合は空文字を返す",
			cache:    nil,
			id:       1,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.GetClassEmblemNameByID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCache_GetClassEmblemBaseNameByID(t *testing.T) {
	tests := []struct {
		name     string
		cache    *Cache
		id       int
		expected string
	}{
		{
			name: "正常系: 存在するIDの場合は名前を返す",
			cache: &Cache{
				ClassEmblemBases: map[string]master.ClassEmblemBase{
					"I":   {ID: 1, Name: "I"},
					"II":  {ID: 2, Name: "II"},
					"III": {ID: 3, Name: "III"},
				},
			},
			id:       2,
			expected: "II",
		},
		{
			name: "異常系: 存在しないIDの場合は空文字を返す",
			cache: &Cache{
				ClassEmblemBases: map[string]master.ClassEmblemBase{
					"I": {ID: 1, Name: "I"},
				},
			},
			id:       999,
			expected: "",
		},
		{
			name:     "異常系: キャッシュがnilの場合は空文字を返す",
			cache:    nil,
			id:       1,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.GetClassEmblemBaseNameByID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}
