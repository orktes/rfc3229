package patch

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kr/binarydist"
)

const example1 = `
This is the old text
`
const example2 = `
This is the new text
`

func TestPatch(t *testing.T) {
	example1Reader := strings.NewReader(example1)
	example2Reader := strings.NewReader(example2)

	var b bytes.Buffer
	err := binarydist.Diff(example1Reader, example2Reader, &b)
	if err != nil {
		t.Error(err)
	}

	var out bytes.Buffer
	err = Patch(strings.NewReader(example1), bytes.NewReader(b.Bytes()), &out)
	if err != nil {
		t.Error(err)
	}

	if string(out.Bytes()) != example2 {
		t.Errorf("Failed %d %+v %d", len(out.Bytes()), out.Bytes(), len(example2))
	}
}

func TestPatchSlice(t *testing.T) {
	example1Reader := strings.NewReader(example1)
	example2Reader := strings.NewReader(example2)

	var b bytes.Buffer
	err := binarydist.Diff(example1Reader, example2Reader, &b)
	if err != nil {
		t.Error(err)
	}

	out, _ := PatchSlice(
		[]byte(example1),
		b.Bytes(),
	)

	if string(out) != example2 {
		t.Errorf("Failed %d %+v %d", len(out), out, len(example2))
	}
}

func mustReturnBytes(bytes []byte, err error) []byte {
	if err != nil {
		panic(err)
	}

	return bytes
}
