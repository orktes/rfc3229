package client

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/kr/binarydist"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getFileMD5(localPath string) (string, error) {
	var returnMD5String string

	file, err := os.Open(localPath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()

	hash := md5.New()

	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}

	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)

	return returnMD5String, nil
}

func patchFile(localPath string, patchData []byte) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}

	tempDirPath, err := ioutil.TempDir("", "testclienttempdir")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Temp dir created:", tempDirPath)

	// Create a file in new temp directory
	tempFile, err := ioutil.TempFile(tempDirPath, "tempfile")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Temp file created:", tempFile.Name())

	binarydist.Patch(file, tempFile, bytes.NewReader(patchData))

	os.Rename(tempFile.Name(), localPath)

	return nil
}

// Get gets a file from the url
func Get(url, localPath string) error {
	md5sum, err := getFileMD5(localPath)
	hasLocalFile := false
	if err == nil {
		hasLocalFile = true
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)

	if hasLocalFile {
		fmt.Fprintf(os.Stderr, "Setting header If-None-Match: %v\n", md5sum)
		req.Header.Set("If-None-Match", md5sum)
		req.Header.Set("A-IM", "bsdiff")
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v", err)
		return err
	}
	fmt.Fprintln(os.Stderr, "Response size:", len(body))

	if len(body) == 0 {
		fmt.Fprintln(os.Stderr, "file not changed - not writing anything.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Headers: %v\n", resp.Header)
	hasBSDiff := false
	if resp.Header.Get("im") == "bsdiff" {
		hasBSDiff = true
	}

	if hasBSDiff {
		const lengthSize uint64 = 8
		headerPatchCount := binary.LittleEndian.Uint64(body[:lengthSize])
		fmt.Fprintf(os.Stderr, "patch count: %v\n", headerPatchCount)

		for i := uint64(0); i < headerPatchCount; i++ {
			lengthOffset := lengthSize * (1 + i)
			patchIndex := binary.LittleEndian.Uint64(body[lengthOffset : lengthOffset+lengthSize])
			fmt.Fprintf(os.Stderr, "patch index: %v %v\n", i, patchIndex)
			err = patchFile(localPath, body[patchIndex:])
			if err != nil {
				return err
			}
		}

		return nil
	}

	err = ioutil.WriteFile(localPath, body, 0644)
	return err
}
