package web

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"PhoenixOracle/lib/logger"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr"
)

const (
	acceptEncodingHeader  = "Accept-Encoding"
	contentEncodingHeader = "Content-Encoding"
	contentLengthHeader   = "Content-Length"
	rangeHeader           = "Range"
	varyHeader            = "Vary"
)

type ServeFileSystem interface {
	http.FileSystem
	Exists(prefix string, path string) bool
}

type BoxFileSystem struct {
	packr.Box
}

func (b *BoxFileSystem) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		return b.Has(p)
	}

	return false
}

type gzipFileHandler struct {
	root ServeFileSystem
}

func GzipFileServer(root ServeFileSystem) http.Handler {
	return &gzipFileHandler{root}
}

func (f *gzipFileHandler) openAndStat(path string) (http.File, os.FileInfo, error) {
	file, err := f.root.Open(path)
	var info os.FileInfo
	if err == nil {
		info, err = file.Stat()
	}
	if err != nil {
		return file, nil, err
	}
	if info.IsDir() {
		return file, nil, fmt.Errorf("%s is directory", path)
	}
	return file, info, nil
}

var preferredEncodings = []string{"gzip"}

func extensionForEncoding(encname string) string {
	switch encname {
	case "gzip":
		return ".gz"
	}
	return ""
}

func (f *gzipFileHandler) findBestFile(w http.ResponseWriter, r *http.Request, fpath string) (http.File, os.FileInfo, error) {
	ae := r.Header.Get(acceptEncodingHeader)
	if ae == "" {
		return f.openAndStat(fpath)
	}

	var available []string
	for _, posenc := range preferredEncodings {
		ext := extensionForEncoding(posenc)
		fname := fpath + ext

		if f.root.Exists("/", fname) {
			available = append(available, posenc)
		}
	}

	negenc := negotiateContentEncoding(r, available)
	if negenc == "" {
		// If we fail to negotiate anything try the base file
		return f.openAndStat(fpath)
	}

	ext := extensionForEncoding(negenc)
	if file, info, err := f.openAndStat(fpath + ext); err == nil {
		wHeader := w.Header()
		wHeader[contentEncodingHeader] = []string{negenc}
		wHeader.Add(varyHeader, acceptEncodingHeader)

		if len(r.Header[rangeHeader]) == 0 {
			wHeader[contentLengthHeader] = []string{strconv.FormatInt(info.Size(), 10)}
		}
		return file, info, nil
	}

	return f.openAndStat(fpath)
}

func negotiateContentEncoding(r *http.Request, available []string) string {
	values := strings.Split(r.Header.Get(acceptEncodingHeader), ",")
	aes := []string{}

	for _, v := range values {
		aes = append(aes, strings.TrimSpace(v))
	}

	for _, a := range available {
		for _, acceptEnc := range aes {
			if acceptEnc == a {
				return a
			}
		}
	}

	return ""
}

func (f *gzipFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}

	fpath := path.Clean(upath)
	if strings.HasSuffix(fpath, "/") {
		http.NotFound(w, r)
		return
	}

	// Find the best acceptable file, including trying uncompressed
	if file, info, err := f.findBestFile(w, r, fpath); err == nil {
		http.ServeContent(w, r, fpath, info.ModTime(), file)
		logger.ErrorIfCalling(file.Close)
		return
	}

	http.NotFound(w, r)
}

func ServeGzippedAssets(urlPrefix string, fs ServeFileSystem) gin.HandlerFunc {
	fileserver := GzipFileServer(fs)
	if urlPrefix != "" {
		fileserver = http.StripPrefix(urlPrefix, fileserver)
	}
	return func(c *gin.Context) {
		if fs.Exists(urlPrefix, c.Request.URL.Path) {
			fileserver.ServeHTTP(c.Writer, c.Request)
			c.Abort()
		} else {
			c.AbortWithStatus(http.StatusNotFound)
		}
	}
}
