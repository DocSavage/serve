package main

import (
    "compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

var (
    fileDirectory = currentDir()
    
	// Display usage if true.
	showHelp = flag.Bool("help", false, "")

	// Run in debug mode if true.
	showLog = flag.Bool("log", false, "")

	// Use gzip by default
	useGzip = flag.Bool("gzip", false, "")

	// Address for http communication
	port = flag.String("port", "localhost:8080", "")
)

const helpMessage = `
Serves a directory (or present working directory) via HTTP on the given port.

Usage: serve [options] [directory]

      -port       =string   Address for HTTP communication.
      -gzip       (flag)    Use gzip compression for responses.
      -log        (flag)    Run in verbose mode.
  -h, -help       (flag)    Show help message
`

var usage = func() {
	fmt.Printf(helpMessage)
}

func currentDir() string {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalln("Could not get current directory:", err)
	}
	return currentDir
}

// Nod to Andrew Gerrand for simple gzip solution:
// See https://groups.google.com/forum/m/?fromgroups#!topic/golang-nuts/eVnTcMwNVjM
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func makeGzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		fn(gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
	}
}

// Handler for static files
func fileHandler(w http.ResponseWriter, r *http.Request) {
	path := "index.html"
	if r.URL.Path != "/" {
		path = r.URL.Path
	}
	filename := filepath.Join(fileDirectory, path)
	if *showLog {
    	fmt.Printf("URL %s -> %s\n", r.URL.Path, filename)
	}
	http.ServeFile(w, r, filename)
}

// Listen and serve HTTP requests using address and don't let stay-alive
// connections hog goroutines for more than an hour.
// See for discussion:
// http://stackoverflow.com/questions/10971800/golang-http-server-leaving-open-goroutines
func serveHTTP() {
	fmt.Printf("Web server listening at %s ...\n", *port)

	src := &http.Server{
		Addr:        *port,
		ReadTimeout: 1 * time.Hour,
	}

	if *useGzip {
		fmt.Println("HTTP server will return gzip values if permitted by browser.")
		http.HandleFunc("/", makeGzipHandler(fileHandler))
	} else {
		http.HandleFunc("/", fileHandler)
	}
	src.ListenAndServe()
}

func main() {
	flag.BoolVar(showHelp, "h", false, "Show help message")
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Capture ctrl+c and other interrupts.  Then handle graceful shutdown.
	stopSig := make(chan os.Signal)
	go func() {
		for sig := range stopSig {
			log.Printf("Captured %v.  Shutting down...\n", sig)
			os.Exit(0)
		}
	}()
	signal.Notify(stopSig, os.Interrupt, os.Kill)
	
	if flag.NArg() > 0 {
	    fileDirectory = flag.Args()[0]
	}
    serveHTTP()
}
