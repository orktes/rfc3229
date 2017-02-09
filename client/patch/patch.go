package patch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/kr/binarydist"
)

func Patch(in, patch io.Reader, out io.Writer) error {
	return binarydist.Patch(in, out, patch)
}

func MultiPatchSlice(in, patchSlice []byte) ([]byte, error) {
	var err error
	headerPatchCount := binary.LittleEndian.Uint64(patchSlice)
	for i := uint64(0); i < headerPatchCount; i++ {
		lengthOffset := 8 * (1 + i)
		patchIndex := binary.LittleEndian.Uint64(patchSlice[lengthOffset:])
		fmt.Printf("Start %d\n", patchIndex)
		in, err = PatchSlice(in, patchSlice[patchIndex:])
	}

	return in, err
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
