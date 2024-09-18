package webclipboard

import (
	"embed"
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
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
		err := s.handlePost(w, r)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("upload successfully\n"))
		log.Println("upload success")
	case http.MethodDelete:
		err := s.handleDelete(w, r)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("delete successfully\n"))
		log.Println("delete success")
	case http.MethodPut:
		err := s.handlePut(w, r)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("put successfully\n"))
		log.Println("put success")
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

	isFromMozilla := isRequestFromMozilla(r.Header.Get("User-Agent"))

	//文件不存在，则展示"上传页面"
	if !exist {
		if !isFromMozilla {
			return
		}

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
	if !isFromMozilla {
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

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) error {
	subPath := makePath(r.URL.Path)
	fullPath := filepath.Join(s.fileDir, subPath)

	//文件上传
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		var fileName string
		var content io.Reader
		log.Println("FormFile start")
		file, header, err := r.FormFile("file")
		if err != nil {
			return err
		}
		defer file.Close()
		fileName = header.Filename
		content = file
		log.Println("createFile start", fileName)
		err = createFile(fullPath, fileName, content)
		if err != nil {
			return err
		}
		return nil
	}

	//body上传
	err := createFile(fullPath, "default.txt", r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return nil
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) error {
	subPath := makePath(r.URL.Path)
	fullPath := filepath.Join(s.fileDir, subPath)
	err := deleteFilesInDir(fullPath)
	if err != nil {
		return err
	}
	err = deleteDirIfHasNoEntry(fullPath)
	if err != nil {
		return err
	}
	return nil
}

// 只处理curl -T file.txt http://example.com 命令时，实际的 HTTP PUT 请求通常会将文件上传到指定的路径。具体来说，如果没有指定路径，curl 会将文件名附加到 URL上
func (s *Server) handlePut(w http.ResponseWriter, r *http.Request) error {
	filename := makePath(r.URL.Path)
	if strings.Contains(filename, "/") {
		return errors.New("multi dir not support in method put")
	}

	err := createFile(s.fileDir, filename, r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return nil
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
