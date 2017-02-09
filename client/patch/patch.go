package patch

import (
	"bytes"
	"io"

	"github.com/kr/binarydist"
)

func Patch(in, patch io.Reader, out io.Writer) error {
	return binarydist.Patch(in, out, patch)
}

func PatchSlice(in, patchSlice []byte) (out []byte, err error) {
	inReader := bytes.NewReader(in)
	patchReader := bytes.NewReader(patchSlice)
	var outBuffer bytes.Buffer

	if err = Patch(inReader, patchReader, &outBuffer); err != nil {
		return
	}

	return outBuffer.Bytes(), nil
}
