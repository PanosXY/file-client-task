package utils

import (
	"io"

	"go.uber.org/zap"
	"golang.org/x/net/html"
)

type Logger struct {
	*zap.Logger
}

func NewLogger(debug bool) (*Logger, error) {
	var l *zap.Logger
	var err error
	if debug {
		l, err = zap.NewDevelopment()
	} else {
		l, err = zap.NewProduction()
	}
	if err != nil {
		return nil, err
	}
	return &Logger{l}, nil
}

// Extract links from http's response body
func GetLinks(body io.Reader) []string {
	var links []string
	z := html.NewTokenizer(body)
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			//todo: links list shoudn't contain duplicates
			return links
		case html.StartTagToken, html.EndTagToken:
			token := z.Token()
			if "a" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}

				}
			}
		}
	}
}
