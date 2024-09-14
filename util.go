package webclipboard

import (
	"errors"
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

// 从目录中，找到一个文件返回
func findOneFileFromDir(dir string) (path string, ok bool, err error) {
	// 打开目录
	f, err := os.Open(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	defer f.Close()

	// 读取目录内容
	files, err := f.Readdir(0)
	if err != nil {
		return "", false, err
	}

	// 遍历目录内容
	for _, file := range files {
		if !file.IsDir() {
			return file.Name(), true, nil
		}
	}
	return "", false, nil
}

func readFile(file string, length int) ([]byte, os.FileInfo, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}

	var content []byte = make([]byte, length)
	n, _ := io.ReadFull(f, content)
	if n == length {
		return append(content[:n], []byte("\n.................\n.................")...), info, nil
	}
	return content[:n], info, nil
}

func downloadHandler(w http.ResponseWriter, r *http.Request, filePath string) {
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

// 判断文件是否存在
func isFileExists(filename string) bool {
	// 使用 os.Stat 获取文件信息
	info, err := os.Stat(filename)
	// 如果 err 为 nil，表示文件存在
	if err != nil {
		return false
	}

	if info.IsDir() {
		return false
	}
	return true
}

func deleteDirIfHasNoFile(dir string) error {
	has, err := hasFileDirInDirectory(dir)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	return os.RemoveAll(dir)
}

func hasFileDirInDirectory(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func fileType(name string) string {
	// 获取文件扩展名
	ext := filepath.Ext(name)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return "image"
	case ".txt", ".log":
		return "text"
	default:
		return "unknow"
	}
}
