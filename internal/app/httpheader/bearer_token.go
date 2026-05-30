package httpheader

import (
	"net/http"
	"strings"
)

const authorizationHeader = "Authorization"

// ExtractBearerToken は Authorization ヘッダから Bearer トークンを取り出します。
func ExtractBearerToken(header http.Header) string {
	authHeader := header.Get(authorizationHeader)
	if authHeader == "" {
		return ""
	}

	scheme, token, found := strings.Cut(authHeader, " ")
	if !found || !strings.EqualFold(strings.TrimSpace(scheme), "Bearer") {
		return ""
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}

	return token
}
