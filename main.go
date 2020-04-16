package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"

	rice "github.com/GeertJohan/go.rice"
)

const serverVersion = "0.01"

// srvConfig is all configures for server
type srvConfig struct {
	showHelp bool

	listenAddress string

	tipHost string
	tipPort string
	tipURL  string
	srvHost string
	srvPort string

	noRice        bool
	staticBaseDir string
	htmlroot      string
	uploadDir     string
}

// String convert current server configures into a readable string
func (srvCfg *srvConfig) String() string {
	var b bytes.Buffer // A Buffer needs no initialization.
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "static base directory: %s\n", srvCfg.staticBaseDir)
	fmt.Fprintf(&b, "HTML root directory: %s\n", srvCfg.htmlroot)
	fmt.Fprintf(&b, "upload directory: %s\n", srvCfg.uploadDir)
	fmt.Fprintf(&b, "listen: %s\n", srvCfg.listenAddress)
	fmt.Fprintf(&b, "URL: %s\n", srvCfg.tipURL)
	return b.String()
}

// runtime server configures
var srvCfg srvConfig

func init() {
	const (
		defaultServerAddressString = ":8081"
		usageServerAddressString   = "Listen address for server"
	)
	flag.StringVar(&srvCfg.listenAddress, "listen", defaultServerAddressString, usageServerAddressString)
	flag.StringVar(&srvCfg.listenAddress, "l", defaultServerAddressString, usageServerAddressString+" (shorthand)")

	const (
		defaultServerStaticBaseDir = "."
		usageServerStaticBaseDir   = "base directory of server static file"
	)
	flag.StringVar(&srvCfg.staticBaseDir, "base", defaultServerStaticBaseDir, usageServerStaticBaseDir)
	flag.StringVar(&srvCfg.staticBaseDir, "b", defaultServerStaticBaseDir, usageServerStaticBaseDir+" (shorthand)")

	const (
		defaultRice = false
		usageRice   = "do not use embedded html files"
	)
	flag.BoolVar(&srvCfg.noRice, "rice", defaultRice, usageRice)
	flag.BoolVar(&srvCfg.noRice, "r", defaultRice, usageRice+" (shorthand)")

	const (
		defaultHelp = false
		usageHelp   = "Listen address for server"
	)
	flag.BoolVar(&srvCfg.showHelp, "help", defaultHelp, usageHelp)
	flag.BoolVar(&srvCfg.showHelp, "h", defaultHelp, usageHelp+" (shorthand)")
}

func requestDumpBuff(w http.ResponseWriter, r *http.Request) bytes.Buffer {
	var b bytes.Buffer // A Buffer needs no initialization.
	fmt.Fprintf(&b, "\n")
	a, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		// handle address not found
		fmt.Println("Error Retrieving local address")
		fmt.Printf("Network(%s): %s <- %s \n", r.URL, "unknown-local-addr", r.RemoteAddr)

		fmt.Fprintf(&b, "Network(%s): %s <- %s \n", r.URL, "unknown-local-addr", r.RemoteAddr)
	} else {
		fmt.Printf("Network(%s): %s <- %s \n", r.URL, a.String(), r.RemoteAddr)

		fmt.Fprintf(&b, "Network(%s): %s <- %s \n", r.URL, a.String(), r.RemoteAddr)
	}
	fmt.Fprintf(&b, "<hr>")
	fmt.Fprintf(&b, "<hr>")

	// Save a copy of this request for debugging.
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Fprintf(&b, "FAILED TO DUMP REQUEST: %+v\n", err)
	} else {
		fmt.Fprintf(&b, string(requestDump))
	}

	if b.Len() > 65535 {
		b.Truncate(65535)
	}

	fmt.Fprintf(&b, "<hr>")
	fmt.Fprintf(&b, "<hr>")

	return b
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	b := requestDumpBuff(w, r)
	defer b.Reset()

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)

		if file == nil && handler == nil {
			fmt.Fprintf(w, htmlMessagePage("Error Retrieving the File:"+err.Error(), "upload failed, select file to upload", text2html(b.String()), "/"))
		} else {
			fmt.Fprintf(w, htmlMessagePage("Error Retrieving the File:"+err.Error(), "upload failed", text2html(b.String()), "/"))
		}
		return
	}
	defer file.Close()

	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("temp-images", "upload-*.png")
	if err != nil {
		fmt.Println(err)
		// return that we have successfully uploaded our file!
		//fmt.Fprintf(w, "failed to write Uploading File: %s\n", err)
		fmt.Fprintf(w, htmlMessagePage("write to disk failed: "+err.Error(), "upload failed", text2html(b.String()), "/"))
		return
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintf(w, htmlMessagePage("read file failed: "+err.Error(), "upload failed", text2html(b.String()), "/"))
		return
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)
	// return that we have successfully uploaded our file!

	fmt.Fprintf(&b, "Uploaded File: %+v\n", handler.Filename)
	fmt.Fprintf(&b, "File Size: %+v\n", handler.Size)
	fmt.Fprintf(&b, "MIME Header: %+v\n", handler.Header)
	fmt.Fprintf(&b, "<hr>")
	fmt.Fprintf(&b, "<hr>")

	fmt.Fprintf(w, htmlMessagePage("Successfully Uploaded File\n", "Uploaded", text2html(b.String()), "/"))
}

func listUploadFile(w http.ResponseWriter, r *http.Request) {
	b := requestDumpBuff(w, r)
	defer b.Reset()

	// fmt.Fprintf(w, "File listing for Upload no implemented\n")
	fmt.Fprintf(w, htmlMessagePage("File listing for Upload no implemented\n", "File listing failed", text2html(b.String()), "/"))
}

// text2html convert /r/n in strings to '<br />'
func text2html(text string) string {
	return strings.ReplaceAll(text, "\n", "<br />")
}

// htmlMessagePage return a html page with input message/title/go back URL
func htmlMessagePage(message, title, addon, backURL string) string {
	if title == "" {
		title = "backend message"
	}
	if backURL == "" {
		backURL = "/"
	}
	return `
	<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
	<title>` + title + `</title>
  </head>
  <body>
    <p>
	  <h1>` + text2html(title) + `</h1>
	  
	  <p>
	  
	<p>
	<h2>` + text2html(message) + `</h2>
	<p>
    <p>
	  <a href="` + backURL + `">Return</a>
	  <p>
	  <h1>DEBUG INFORMATION</h1>
	  <hr>
	  <h3>
	  ` + text2html(addon) + `
	  </h3>
	  <hr>
	  <p>
  </body>
</html>
	`
}

// realPath returns an absolute representation of path or symbolic links.
func realPath(path string) (string, error) {
	rp, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path, err
	}
	return filepath.Abs(rp)
}

func setupRoutes() error {
	if srvCfg.noRice {

		htmlroot, err := realPath(srvCfg.htmlroot + "/htmlroot")
		if err != nil {
			log.Fatalln(err)
		}
		fs := http.FileServer(http.Dir(htmlroot))
		http.Handle("/", fs)
	} else {

		htmlroot, err := rice.FindBox("webroot/htmlroot")
		if err != nil {
			return err
		}

		http.Handle("/", http.FileServer(htmlroot.HTTPBox()))
	}
	http.HandleFunc("/upload", uploadFile)
	http.HandleFunc("/files", listUploadFile)
	return http.ListenAndServe(srvCfg.listenAddress, nil)
}

func main() {
	var err error
	log.Printf("Http multipart upload server with go.rice, version %s\n", serverVersion)

	flag.Parse()
	if srvCfg.showHelp {
		flag.Usage()
		os.Exit(0)
	}
	os.MkdirAll(srvCfg.staticBaseDir, 0755)
	srvCfg.staticBaseDir, err = realPath(srvCfg.staticBaseDir)
	if err != nil {
		log.Fatalln(err)
	}

	os.MkdirAll(srvCfg.staticBaseDir+"/htmlroot", 0755)
	srvCfg.htmlroot, err = realPath(srvCfg.staticBaseDir + "/htmlroot")
	if err != nil {
		log.Fatalln(err)
	}

	os.MkdirAll(srvCfg.staticBaseDir+"/upload", 0755)
	srvCfg.uploadDir, err = realPath(srvCfg.staticBaseDir + "/upload")
	if err != nil {
		log.Fatalln(err)
	}

	listenInfo := strings.Split(srvCfg.listenAddress, ":")
	if len(listenInfo) <= 1 {
		srvCfg.tipPort = listenInfo[0]
	} else {
		srvCfg.tipPort = listenInfo[len(listenInfo)-1]
		srvCfg.tipHost = listenInfo[len(listenInfo)-2]
	}
	srvCfg.srvHost = srvCfg.tipHost
	srvCfg.srvPort = srvCfg.tipPort
	if srvCfg.tipHost == "0.0.0.0" || srvCfg.tipHost == "" {
		srvCfg.tipHost = "127.0.0.1"
	}
	srvCfg.listenAddress = srvCfg.srvHost + ":" + srvCfg.srvPort
	srvCfg.tipURL = "http://" + srvCfg.tipHost
	if srvCfg.tipPort != "80" {
		srvCfg.tipURL += ":" + srvCfg.tipPort
	}

	log.Println(srvCfg.String())
	if err := setupRoutes(); err != nil {
		log.Fatalln(err)
	}
}
