package webclipboard

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed html
var assets embed.FS

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
	subPath := makePath(r.URL.Path)
	fullPath := filepath.Join(s.fileDir, subPath)

	//如果能直接访问到文件地址，则直接下载文件
	if isFileExists(fullPath) {
		downloadHandler(w, r, fullPath)
		return
	}

	//看目录下是否有文件
	filename, exist, err := findOneFileFromDir(fullPath)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	//文件不存在，则展示"上传页面"
	if !exist {
		data, err := assets.ReadFile("html/upload.html")
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	newPath := filepath.Join(fullPath, filename)
	//非浏览器的请求，则直接下载文件
	if !isRequestFromMozilla(r.Header.Get("User-Agent")) {
		downloadHandler(w, r, newPath)
		return
	}

	//浏览器的请求，展示下载页面
	content, fileInfo, err := readFile(newPath, s.showLength)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	var radio = readableRatio(content)
	if radio < 0.60 {
		content = []byte("unreadable binary file")
	}
	renderDownloadPage(w, filepath.Join(subPath, filename), string(content), fileInfo.Size())
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	subPath := makePath(r.URL.Path)
	fullPath := filepath.Join(s.fileDir, subPath)
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
		err = createFile(fullPath, fileName, content)
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
		err := createFile(fullPath, fileName, content)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		w.Write([]byte("File uploaded successfully"))
	case "delete_file":
		filename := r.FormValue("filename")
		err := os.Remove(filepath.Join(fullPath, filename))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete file:%s", err), http.StatusInternalServerError)
			return
		}
		err = deleteDirIfHasNoFile(fullPath)
		if err != nil {
			log.Println("deleteDirIfHasNoFile failed", err)
			return
		}
	default:
		http.Error(w, "Invalid option", http.StatusBadRequest)
	}
}

func renderDownloadPage(w http.ResponseWriter, filePath, content string, fileSize int64) {
	tmpl, err := template.ParseFS(assets, "html/download.html")
	if err != nil {
		http.Error(w, "Unable to parse template", http.StatusInternalServerError)
		return
	}
	data := struct {
		FileName string
		Content  string
		FileSize string
		FilePath string
		FileType string
	}{
		FileName: filepath.Base(filePath),
		Content:  content,
		FileSize: humanBytes(fileSize),
		FilePath: filePath,
		FileType: fileType(filePath),
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to execute template", http.StatusInternalServerError)
	}
}

func makePath(urlPath string) (subPath string) {
	if path := strings.TrimPrefix(urlPath, "/"); len(path) > 0 {
		return path
	}

	return ""
}
