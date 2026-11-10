package api_internal

// UpdateUserCommentRequest は未設定への変更も文字列で明示させるため、欠落・nullを区別します。
type UpdateUserCommentRequest struct {
	Comment *string `json:"comment"`
}

// UserCommentResponse はコメント未設定をnullで返します。
type UserCommentResponse struct {
	Comment *string `json:"comment"`
}
