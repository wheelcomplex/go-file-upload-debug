package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
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
		defaultHelp = false
		usageHelp   = "Listen address for server"
	)
	flag.BoolVar(&srvCfg.showHelp, "help", defaultHelp, usageHelp)
	flag.BoolVar(&srvCfg.showHelp, "h", defaultHelp, usageHelp+" (shorthand)")
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	a, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		// handle address not found
		fmt.Println("Error Retrieving local address")
		fmt.Println(ok)
		return
	}
	fmt.Printf("File Upload Endpoint Hit: %s <- %s \n", a.String(), r.RemoteAddr)

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
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)
	// return that we have successfully uploaded our file!
	fmt.Fprintf(w, "Successfully Uploaded File\n")
}

func listUploadFile(w http.ResponseWriter, r *http.Request) {
	a, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		// handle address not found
		fmt.Println("Error Retrieving local address")
		fmt.Println(ok)
		return
	}
	fmt.Printf("File Upload Endpoint Hit: %s <- %s \n", a.String(), r.RemoteAddr)

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
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)
	// return that we have successfully uploaded our file!
	fmt.Fprintf(w, "Successfully Uploaded File\n")
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
	htmlroot, err := rice.FindBox(srvCfg.htmlroot)
	if err != nil {
		return err
	}
	http.Handle("/", http.FileServer(htmlroot.HTTPBox()))
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
