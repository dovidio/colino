package youtube

import (
	neturl "net/url"
	"strings"
)

func ExtractYouTubeID(u string) string {
	parsed, err := neturl.Parse(u)
	if err != nil {
		return ""
	}
	h := strings.ToLower(parsed.Host)
	if h == "youtu.be" {
		return strings.Trim(parsed.Path, "/")
	}
	if ytHostRe.MatchString(h) {
		if strings.HasPrefix(parsed.Path, "/watch") {
			q := parsed.Query()
			return strings.TrimSpace(q.Get("v"))
		}
		if strings.HasPrefix(parsed.Path, "/shorts/") {
			return strings.Trim(strings.TrimPrefix(parsed.Path, "/shorts/"), "/")
		}
	}
	return ""
}
