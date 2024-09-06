package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var listen = flag.String("listen", "0.0.0.0:80", "http server listen addr")
var fileDir = flag.String("file_dir", "file", "file dir")
var showLength = flag.Int("show_length", 10000, "show_length")

func main() {
	flag.Parse()
	parseEnv()

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

func parseEnv() {
	if lis := os.Getenv("listen"); len(lis) > 0 {
		listen = &lis
	}
	if dir := os.Getenv("file_dir"); len(dir) > 0 {
		fileDir = &dir
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	folderName := folderName(r.URL.Path)
	var opt = r.FormValue("opt")

	if r.Method == http.MethodGet {
		switch opt {
		case "download_file":
			var filename = r.FormValue("filename")
			downloadHandler(w, r, folderName, filename)
			return
		default:
			path, info, exist := getFile(folderName)
			if !exist {
				http.ServeFile(w, r, "index.html")
				return
			}

			content, err := readFile(path, *showLength)
			if err != nil {
				http.Error(w, "Failed to read file", http.StatusInternalServerError)
				return
			}
			renderTemplate(w, info.Name(), string(content))
		}
		return
	}

	if r.Method != http.MethodPost {
		return
	}

	switch opt {
	case "upload_file":
		var fileName string
		var content io.Reader
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		fileName = header.Filename
		content = file
		err = createFile(folderName, fileName, content)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		w.Write([]byte("File uploaded successfully"))
	case "upload_string":
		var fileName string
		var content io.Reader
		fileName = "noname.txt"
		str := r.FormValue("string")
		content = strings.NewReader(str)
		err := createFile(folderName, fileName, content)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		w.Write([]byte("File uploaded successfully"))
	case "delete_file":
		err := os.RemoveAll(folderName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete file:%s", err), http.StatusInternalServerError)
			return
		}
	case "download_file":

	default:
		http.Error(w, "Invalid option", http.StatusBadRequest)
	}

}

func renderTemplate(w http.ResponseWriter, fileName, content string) {
	tmpl, err := template.ParseFiles("file.html")
	if err != nil {
		http.Error(w, "Unable to parse template", http.StatusInternalServerError)
		return
	}
	data := struct {
		FileName string
		Content  string
	}{
		FileName: fileName,
		Content:  content,
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to execute template", http.StatusInternalServerError)
	}
}

func folderName(path string) string {
	var dir1 = *fileDir
	var dir2 = "root"

	if path = strings.TrimPrefix(path, "/"); len(path) > 0 {
		dir2 = path
	}

	return dir1 + "/" + dir2
}

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

func readFile(file string, length int) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var content []byte = make([]byte, length)
	n, _ := io.ReadFull(f, content)
	if n == length {
		return string(content[:n]) + "\n" + "................." + "\n" + ".................", nil
	}
	return string(content[:n]), nil
}

func downloadHandler(w http.ResponseWriter, r *http.Request, fileFolder, fileName string) {
	var filePath = fileFolder + "/" + fileName
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	http.ServeFile(w, r, filePath)
}
