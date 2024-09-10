package webclipboard

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Server struct {
	showLength int
	fileDir    string
	listen     string
}

func NewServer(listen string, showLength int, fileDir string) *Server {
	return &Server{
		showLength: showLength,
		fileDir:    fileDir,
		listen:     listen,
	}
}

func (s *Server) Run() {
	checkTempDir()
	http.HandleFunc("/", s.handler)
	go func() {
		log.Fatal(http.ListenAndServe(s.listen, nil))
	}()
	return
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	log.Println("handler start", r)
	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r)
	case http.MethodPost:
		s.handlePost(w, r)
	default:

	}
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	folderName := folderName(s.fileDir, r.URL.Path)
	var opt = r.FormValue("Option")
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
		//浏览器的请求，展示file页面，非浏览器的请求，下载文件
		if isRequestFromMozilla(r.Header.Get("User-Agent")) {
			content, err := readFile(path, s.showLength)
			if err != nil {
				http.Error(w, "Failed to read file", http.StatusInternalServerError)
				return
			}
			renderTemplate(w, info.Name(), string(content), info.Size())
		} else {
			downloadHandler(w, r, folderName, info.Name())
		}
	}
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	folderName := folderName(s.fileDir, r.URL.Path)
	var opt = r.Header.Get("Option")

	switch opt {
	case "upload_file":
		var fileName string
		var content io.Reader
		log.Println("FormFile start")
		file, header, err := r.FormFile("file")
		if err != nil {
			log.Println("formfile fail", err)
			http.Error(w, "Failed to get file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		fileName = header.Filename
		content = file
		log.Println("createFile start", fileName)
		err = createFile(folderName, fileName, content)
		if err != nil {
			log.Println("createFile failed", err)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		w.Write([]byte("File uploaded successfully"))
		log.Println("File uploaded success", fileName)
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
	default:
		http.Error(w, "Invalid option", http.StatusBadRequest)
	}
}

func renderTemplate(w http.ResponseWriter, fileName, content string, fileSize int64) {
	tmpl, err := template.ParseFiles("file.html")
	if err != nil {
		http.Error(w, "Unable to parse template", http.StatusInternalServerError)
		return
	}
	data := struct {
		FileName string
		Content  string
		FileSize string
	}{
		FileName: fileName,
		Content:  content,
		FileSize: humanBytes(fileSize),
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to execute template", http.StatusInternalServerError)
	}
}

func folderName(dir string, path string) string {
	var dir1 = dir
	var dir2 = "root"

	if path = strings.TrimPrefix(path, "/"); len(path) > 0 {
		dir2 = path
	}

	return dir1 + "/" + dir2
}
