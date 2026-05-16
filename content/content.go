package content

import (
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// AbsHrefSrc tokenizes htm, makes href and src attributes absolute (using urlpath),
// and returns the result. The tokenizer uses the contextTag "body".
func AbsHrefSrc(htm, urlpath string) string {
	tokenizer := html.NewTokenizerFragment(strings.NewReader(htm), "body")
	var result strings.Builder
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break // assuming tokenizer.Err() == io.EOF
		}
		if tokenType != html.StartTagToken {
			result.Write(tokenizer.Raw()) // raw copy everything except start tags
			continue
		}

		token := tokenizer.Token()
		for i, a := range token.Attr {
			key := strings.ToLower(a.Key)
			if key == "href" || key == "src" {
				if u, err := url.Parse(strings.TrimSpace(a.Val)); err == nil {
					if key == "href" && u.Scheme == "" && u.Path != "" && !path.IsAbs(u.Path) {
						token.Attr[i].Val = path.Join(urlpath, a.Val)
					}
					if key == "src" && u.Scheme == "" && !path.IsAbs(u.Path) {
						token.Attr[i].Val = path.Join(urlpath, a.Val)
					}
				}
			}
		}
		result.WriteString(token.String())
	}
	return result.String()
}
