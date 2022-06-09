package types

import "io"

type FileContent struct {
	Filename string
	Content  *io.ReadCloser
}
