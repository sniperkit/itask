package profileutil

import (
	"crypto/md5"
	"fmt"
	"strings"
)

func GravatarHashFromEmail(email string) string {
	input := strings.ToLower(strings.TrimSpace(email))
	return fmt.Sprintf("%x", md5.Sum([]byte(input)))
}
