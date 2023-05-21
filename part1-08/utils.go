package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"

	"golang.org/x/exp/constraints"
)

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashValue := hash.Sum(nil)

	return fmt.Sprintf("%x", hashValue), nil
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// return true if two files are the same
func compareTwoFiles(f1, f2 string) (bool, error) {
	h1, err := hashFile(f1)
	if err != nil {
		return false, fmt.Errorf("could not hash %s: %v", f1, err)
	}

	h2, err := hashFile(f2)
	if err != nil {
		return false, fmt.Errorf("could not hash %s: %v", f2, err)
	}

	return h1 == h2, nil
}

func assert(expr bool, msg string, args ...any) {
	if !expr {
		str := fmt.Sprintf(msg, args...)
		panic(fmt.Sprintf("assert error: %s\n", str))
	}
}

func max[T constraints.Ordered](x, y T) T {
	if x < y {
		return y
	}
	return x
}
