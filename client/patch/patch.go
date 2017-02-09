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
	fmt.Printf("Going to patch %d with %d patch\n", len(in), len(patchSlice))

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

	out = outBuffer.Bytes()

	fmt.Printf("After patching in file of size: %d with patch %d the result is %d\n", len(in), len(patchSlice), len(out))

	return
}
