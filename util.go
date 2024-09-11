package webclipboard

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

func createFile(folder string, fileName string, reader io.Reader) error {
	err := os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		return err
	}
	out, err := os.Create(folder + "/" + fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	if err != nil {
		return err
	}
	return nil
}

func getFile(folder string) (path string, info os.FileInfo, ok bool) {
	filepath.Walk(folder, func(path1 string, info1 os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info1.IsDir() {
			path = path1
			info = info1
			ok = true
		}
		return nil
	})

	return
}

func readFile(file string, length int) ([]byte, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var content []byte = make([]byte, length)
	n, _ := io.ReadFull(f, content)
	if n == length {
		return append(content[:n], []byte("\n.................\n.................")...), nil
	}
	return content[:n], nil
}

func downloadHandler(w http.ResponseWriter, r *http.Request, fileFolder, fileName string) {
	var filePath = fileFolder + "/" + fileName
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	http.ServeFile(w, r, filePath)
}

func checkTempDir() {
	tempDir := os.TempDir()
	if err := os.MkdirAll(tempDir, 1777); err != nil {
		log.Fatalf("Failed to create temporary directory %s: %s", tempDir, err)
	}
	tempFile, err := ioutil.TempFile("", "genericInit_")
	if err != nil {
		log.Fatalf("Failed to create tempFile: %s", err)
	}
	_, err = fmt.Fprintf(tempFile, "Hello, World!")
	if err != nil {
		log.Fatalf("Failed to write to tempFile: %s", err)
	}
	if err := tempFile.Close(); err != nil {
		log.Fatalf("Failed to close tempFile: %s", err)
	}
	if err := os.Remove(tempFile.Name()); err != nil {
		log.Fatalf("Failed to delete tempFile: %s", err)
	}
	log.Printf("Using temporary directory %s", tempDir)
}

// 是否是浏览器发出的请求
func isRequestFromMozilla(userAgent string) bool {
	return strings.Contains(userAgent, "Mozilla")
}

func humanBytes(size int64) string {
	kb, mb := bytesToKBMB(size)
	if mb >= 1 {
		return fmt.Sprintf("%0.2fMB", mb)
	}

	if kb >= 1 {
		return fmt.Sprintf("%0.2fKB", kb)
	}

	return fmt.Sprintf("%d", size)
}

func bytesToKBMB(bytes int64) (float64, float64) {
	kB := float64(bytes) / 1024
	MB := kB / 1024
	return kB, MB
}

func isReadableRune(r rune) bool {
	return unicode.IsPrint(r) || unicode.IsSpace(r)
}

func readableRatio(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	readableCount := 0
	totalRunes := 0
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 encoding, skip this byte
			data = data[1:]
			totalRunes++
			continue
		}
		if isReadableRune(r) {
			readableCount++
		} else {
			fmt.Println(r)
		}
		totalRunes++
		data = data[size:]
	}

	if totalRunes == 0 {
		return 0.0
	}

	return float64(readableCount) / float64(totalRunes)
}
