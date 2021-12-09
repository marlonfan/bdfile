package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	outDir     string
	input      string
	thread     int
	strictMode bool
	helpFlag   bool
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxConnsPerHost: 1 << 10,
	},
}

var mimeTypeSet = map[string]string{
	"image/jpeg":                                    "jpg",
	"image/jp2":                                     "jp2",
	"image/png":                                     "png",
	"image/gif":                                     "gif",
	"image/webp":                                    "webp",
	"image/x-canon-cr2":                             "cr2",
	"image/tiff":                                    "tif",
	"image/bmp":                                     "bmp",
	"image/vnd.ms-photo":                            "jxr",
	"image/vnd.adobe.photoshop":                     "psd",
	"image/vnd.microsoft.icon":                      "ico",
	"image/heif":                                    "heif",
	"image/vnd.dwg":                                 "dwg",
	"application/wasm":                              "wasm",
	"application/vnd.android.dex":                   "dex",
	"application/vnd.android.dey":                   "dey",
	"application/epub+zip":                          "epub",
	"application/zip":                               "zip",
	"application/x-tar":                             "tar",
	"application/vnd.rar":                           "rar",
	"application/gzip":                              "gz",
	"application/x-bzip2":                           "bz2",
	"application/x-7z-compressed":                   "7z",
	"application/x-xz":                              "xz",
	"application/zstd":                              "zst",
	"application/pdf":                               "pdf",
	"application/x-shockwave-flash":                 "swf",
	"application/rtf":                               "rtf",
	"application/octet-stream":                      "eot",
	"application/postscript":                        "ps",
	"application/vnd.sqlite3":                       "sqlite",
	"application/x-nintendo-nes-rom":                "nes",
	"application/x-google-chrome-extension":         "crx",
	"application/vnd.ms-cab-compressed":             "cab",
	"application/vnd.debian.binary-package":         "deb",
	"application/x-unix-archive":                    "ar",
	"application/x-compress":                        "Z",
	"application/x-lzip":                            "lz",
	"application/x-rpm":                             "rpm",
	"application/x-executable":                      "elf",
	"application/dicom":                             "dcm",
	"application/x-iso9660-image":                   "iso",
	"application/x-mach-binary":                     "macho",
	"application/msword":                            "doc",
	"application/vnd.ms-excel":                      "xls",
	"application/vnd.microsoft.portable-executable": "exe",
	"application/vnd.ms-powerpoint":                 "ppt",
	"application/font-sfnt":                         "ttf",
	"application/font-woff":                         "woff",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   "docx",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
}

var helpTemplate = `a small tool for download some file, like image excel, html and everything...
-i string
	input string, like: a,b,a...
-o string
	output dir
-t int
	tread number (default 10)
-s bool
	in case of strict mode, an error will terminate the operation
example:
	bdfile -t 10 -o ./output_dir -i 'http://baidu.com/1.png,http://baidu.com/2.png'`

func main() {
	flag.StringVar(&outDir, "o", "", "output dir")
	flag.StringVar(&input, "i", "", "input string, like: a,b,a...")
	flag.IntVar(&thread, "t", 10, "tread number")
	flag.BoolVar(&helpFlag, "h", false, "help")
	flag.BoolVar(&strictMode, "s", false, "in case of strict mode, an error will terminate the operation")
	flag.Parse()

	if helpFlag {
		fmt.Fprintln(os.Stdout, helpTemplate)
		return
	}

	if strings.TrimSpace(input) == "" || strings.TrimSpace(outDir) == "" {
		fmt.Fprintln(os.Stderr, "param is invalid!")
		return
	}

	fileList := strings.Split(input, ",")
	if len(fileList) == 0 {
		fmt.Fprintln(os.Stderr, "param is invalid!")
		return
	}

	queue := make(chan struct{}, thread)
	wg := &sync.WaitGroup{}

	for _, file := range fileList {
		queue <- struct{}{}
		wg.Add(1)

		go func(file string) {
			defer func() {
				<-queue
				wg.Done()
			}()

			downloadFile(file, outDir)
		}(file)
	}

	wg.Wait()
	fmt.Fprintln(os.Stdout, "download success!")
}

func downloadFile(file, path string) {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		check(file, err)
		return
	}
	resp, err := httpClient.Get(file)
	if err != nil {
		check(file, err)
		return
	}
	defer resp.Body.Close()

	filename := filepath.Base(file)
	// uri.Path
	kind, ok := mimeTypeSet[resp.Header.Get("Content-Type")]
	if !ok {
		check(file, errors.New("not match mime type"))
		return
	}
	if !strings.HasSuffix(filename, kind) {
		filename = fmt.Sprintf("%s.%s", filename, kind)
	}
	writer, err := os.Create(filepath.Join(path, filename))
	if err != nil {
		check(file, err)
		return
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		check(file, err)
	}
}

func check(file string, err error) {
	fmt.Fprintf(os.Stderr, "%s download err, reason: %s\n", file, err.Error())
	if strictMode {
		os.Exit(1)
	}
}
