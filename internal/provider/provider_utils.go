package provider

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// TODO hash with ignoring line endings / OS-independent
func hashFileMD5(file string) (string, error) {
	// Open the file
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Create an MD5 hasher
	hasher := md5.New()

	// Copy the file contents to the hasher
	_, err = io.Copy(hasher, f)
	if err != nil {
		return "", err
	}

	// Get the MD5 sum as a byte slice
	md5Sum := hasher.Sum(nil)

	// Convert the byte slice to a hexadecimal string
	md5String := hex.EncodeToString(md5Sum)

	return md5String, nil
}
