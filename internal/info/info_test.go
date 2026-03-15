package info

import "testing"

func TestIsKnownAccountType(t *testing.T) {
	tests := []struct {
		name          string
		accountTypeID int
		want          bool
	}{
		{name: "PLAYERは既知ロール", accountTypeID: AccountTypePlayer, want: true},
		{name: "EDITORは既知ロール", accountTypeID: AccountTypeEditor, want: true},
		{name: "ADMINは既知ロール", accountTypeID: AccountTypeAdmin, want: true},
		{name: "0は未知ロール", accountTypeID: 0, want: false},
		{name: "4は未知ロール", accountTypeID: 4, want: false},
		{name: "負数は未知ロール", accountTypeID: -1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsKnownAccountType(tt.accountTypeID)
			if got != tt.want {
				t.Fatalf("IsKnownAccountType(%d) = %v, want %v", tt.accountTypeID, got, tt.want)
			}
		})
	}
}

func TestHasRole(t *testing.T) {
	tests := []struct {
		name           string
		accountTypeID  int
		requiredRoleID int
		want           bool
	}{
		{name: "ADMIN要求にADMINは成功", accountTypeID: AccountTypeAdmin, requiredRoleID: AccountTypeAdmin, want: true},
		{name: "ADMIN要求にEDITORは失敗", accountTypeID: AccountTypeEditor, requiredRoleID: AccountTypeAdmin, want: false},
		{name: "ADMIN要求にPLAYERは失敗", accountTypeID: AccountTypePlayer, requiredRoleID: AccountTypeAdmin, want: false},
		{name: "EDITOR要求にADMINは成功", accountTypeID: AccountTypeAdmin, requiredRoleID: AccountTypeEditor, want: true},
		{name: "EDITOR要求にEDITORは成功", accountTypeID: AccountTypeEditor, requiredRoleID: AccountTypeEditor, want: true},
		{name: "EDITOR要求にPLAYERは失敗", accountTypeID: AccountTypePlayer, requiredRoleID: AccountTypeEditor, want: false},
		{name: "PLAYER要求にPLAYERは成功", accountTypeID: AccountTypePlayer, requiredRoleID: AccountTypePlayer, want: true},
		{name: "PLAYER要求にEDITORは成功", accountTypeID: AccountTypeEditor, requiredRoleID: AccountTypePlayer, want: true},
		{name: "PLAYER要求にADMINは成功", accountTypeID: AccountTypeAdmin, requiredRoleID: AccountTypePlayer, want: true},
		{name: "未知ロールIDは失敗", accountTypeID: 4, requiredRoleID: AccountTypeAdmin, want: false},
		{name: "未知ロールIDは数値が大きくても失敗", accountTypeID: 999, requiredRoleID: AccountTypeAdmin, want: false},
		{name: "未知のrequiredRoleIDは失敗", accountTypeID: AccountTypeAdmin, requiredRoleID: 4, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasRole(tt.accountTypeID, tt.requiredRoleID)
			if got != tt.want {
				t.Fatalf("HasRole(%d, %d) = %v, want %v", tt.accountTypeID, tt.requiredRoleID, got, tt.want)
			}
		})
	}
}
