package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/jessevdk/go-flags"
)

type CacheEntry struct {
	Content     []byte
	ContentType string
}

type Arguments struct {
	DefaultDoc string `short:"d" long:"default-doc" description:"On 404, return this document" default:"index.html"`
	Port       int    `short:"p" long:"port" description:"Port to listen on" default:"80"`
	MemCache   bool   `short:"c" long:"cache" description:"Enable memcache"`
	Positional struct {
		Directory string `positional-arg-name:"DIR" description:"Directory to host" required:"true"`
	} `positional-args:"yes"`
}

var args Arguments

func main() {
	_, err := flags.Parse(&args)
	if err != nil {
		if !flags.WroteHelp(err) {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}

	args.Positional.Directory, err = filepath.Abs(args.Positional.Directory)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	cache := map[string]CacheEntry{}
	types := map[string]string{
		".js":   "application/javascript",
		".css":  "text/css",
		".html": "text/html",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
	}

	defaultDoc := filepath.Join(args.Positional.Directory, args.DefaultDoc)
	if !strings.HasPrefix(defaultDoc, args.Positional.Directory) {
		panic("default doc is not in the directory")
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// parse URL down to the file being asked for
		path := r.URL.Path
		origPath := path
		if path == "/" {
			path = args.DefaultDoc
		}

		fullpath := filepath.Join(args.Positional.Directory, path)
		if !strings.HasPrefix(fullpath, args.Positional.Directory) {
			fullpath = defaultDoc
		}

	again:
		relPath := strings.TrimPrefix(fullpath, args.Positional.Directory)

		// check if we have a cached version
		if args.MemCache {
			if entry, ok := cache[fullpath]; ok {
				clr := color.Green // used a cached version
				if origPath != relPath {
					clr = color.Yellow // corrected to default doc
				}
				clr("%s => %s (%s)", origPath, relPath, entry.ContentType)
				w.Header().Add("Content-Type", entry.ContentType)
				_, _ = w.Write(entry.Content)

				return
			}
		}

		file, err := os.Open(fullpath)
		if err != nil {
			color.Red("unable to open file: %s", fullpath)
			if fullpath != defaultDoc {
				fullpath = defaultDoc

				goto again
			} else {
				http.Error(w, err.Error(), http.StatusNotFound)
				color.Red("%s => ??? (404)", origPath)

				return
			}
		}

		defer file.Close()

		raw, err := ioutil.ReadAll(file)
		if err != nil {
			color.Red("unable to read file: %s", fullpath)
			http.Error(w, "unable to read file", http.StatusInternalServerError)
			color.Red("%s => ??? (404)", origPath)
			return
		}

		ext := filepath.Ext(fullpath)
		contentType, ok := types[ext]
		if !ok {
			length := len(raw)
			if length > 512 {
				length = 512
			}

			contentType := http.DetectContentType(raw[:length])
			if contentType != "application/octet-stream" {
				types[ext] = contentType
			}
		}

		if args.MemCache {
			cache[fullpath] = CacheEntry{
				Content:     raw,
				ContentType: contentType,
			}
		}

		fmt.Printf("%s => %s", origPath, relPath)
		if args.MemCache {
			fmt.Printf("(%s)", color.MagentaString("added to cache"))
		}

		fmt.Println()

		w.Header().Add("Content-Type", contentType)
		_, _ = w.Write(raw)
	})

	srv := &http.Server{
		Addr:    net.JoinHostPort("", strconv.Itoa(args.Port)),
		Handler: mux,
	}

	fmt.Printf("now listening on %s\n", srv.Addr)
	_ = srv.ListenAndServe()
}
