package main

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

const releasesUrl = "https://releases.hashicorp.com/terraform/"
const tfgetHome = "$HOME/.tfget/versions"

//func UnarchiveZipFile(filepath string) {
//	log.Println("Unzipping", filepath)
//}

func DownloadTerraform(filepath string, version_number string) error {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	terraformUrl := releasesUrl + version_number + "/terraform_" + version_number + "_" + platform + ".zip"

	log.WithFields(log.Fields{
		"filepath": filepath,
	}).Info("Downloading Terraform version")

	// Handle HTTP request
	client := &http.Client{}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", terraformUrl, nil)
	if err != nil {
		return err
	}

	// Get the data
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create local file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write local file body
	_, err = io.Copy(out, resp.Body)
	log.WithFields(log.Fields{
		"filepath": filepath,
	}).Info("Downloaded Terraform version on disk")
	return err
}

func ListRemoteVersions() []string {
	// Handle HTTP request
	client := &http.Client{}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", releasesUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Get the data
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var versions []string

	// https://html.spec.whatwg.org/multipage/parsing.html#tokenization
	z := html.NewTokenizer(bufio.NewReader(resp.Body))
	for {
		tt := z.Next()
		if tt == html.ErrorToken { // Reached EOF
			// Sort from latest to oldest
			sort.Sort(sort.Reverse(sort.StringSlice(versions)))
			return versions
		} else if tt == html.StartTagToken {
			t := z.Token()
			// <li>
			// 	<a href="/terraform/$version_number/">terraform_$version_number</a>
			// </li>
			if t.Data == "li" {
				z.Next()
				z.Next()
				z.Next()
				t = z.Token()
				if strings.Contains(t.Data, "terraform") {
					// terraform_$version_number
					version_number := strings.Split(t.Data, "_")[1]
					versions = append(versions, version_number)
				}
			}
		}
	}
}

func ListLocal(dirPath string) {
	fp, err := os.Open(dirPath)
	if err != nil {
		log.Fatal(err)
	}
	/*
		Readdir(n int) reads the contents of the directory associated with file and returns a slice of up to n FileInfo values,
		as would be returned by Lstat, in directory order.
		Subsequent calls on the same file will yield further FileInfos.
		If n > 0, Readdir() returns at most n FileInfo structures.
		In this case, if Readdir() returns an empty slice, it will return a non-nil error explaining why.
		At the end of a directory, the error is io.EOF.
		If n <= 0, Readdir() returns all the FileInfo from the directory in a single slice
		(Explication shamelessly copied from https://golang.cafe/blog/how-to-list-files-in-a-directory-in-go.html)
	*/
	files, err := fp.Readdir(0)
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range files {
		log.Println(v.Name())
	}
}

func DetermineVersion(cliArg string, versions []string) string {
	var version_number string

	if cliArg != "" {
		if cliArg == "latest" {
			version_number = versions[0]
		} else {
			found_it := false
			for _, a_version := range versions {
				if strings.Contains(a_version, cliArg) {
					version_number = a_version
					found_it = true
					break
				}
			}
			if found_it {
				version_number = cliArg
			} else {
				log.WithFields(log.Fields{
					"version_number": version_number,
				}).Fatal("Version not found ")
			}
		}
	} else {
		log.Fatal("No CLI arguments found")
	}

	return version_number
}

// Local cache
func mkdirLocalCache(dirPath string) string {
	// Replace $HOME with actual user home
	if strings.Contains(dirPath, "$HOME") {
		dirname, homeErr := os.UserHomeDir()
		if homeErr != nil {
			log.Fatal(homeErr)
		}
		dirPath = strings.Replace(dirPath, "$HOME", dirname, -1)
	}

	// dirPermissions := int(0700)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"dirPath": dirPath,
		}).Info("Directory not found, creating")
		mkdirErr := os.MkdirAll(dirPath, 0700)
		if mkdirErr != nil {
			log.Fatal(mkdirErr)
		}
	}
	return dirPath
}

func init() {
	// Log as JSON instead of the default ASCII text
	//log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above
	log.SetLevel(log.InfoLevel)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Help not implemented yet.")
	}
	dirPath := mkdirLocalCache(tfgetHome)

	switch option := os.Args[1]; option {
	case "list", "list-local":
		ListLocal(dirPath)
	case "list-remote":
		log.Info("Listing all remote versions")
		versions := ListRemoteVersions()
		for _, v := range versions {
			log.Info(v)
		}
	case "download":
		version_number := DetermineVersion(os.Args[2], ListRemoteVersions())
		filePath := "terraform_" + version_number
		fullPath := dirPath + "/" + filePath
		fullPathZip := fullPath + ".zip"
		if _, err := os.Stat(fullPathZip); os.IsNotExist(err) {
			downloadErr := DownloadTerraform(fullPathZip, version_number)
			if downloadErr != nil {
				log.Fatal(downloadErr)
			}
		} else {
			log.WithFields(log.Fields{
				"version_number": version_number,
				"fullPathZip":    fullPathZip,
			}).Info("Version already exists on disk")
		}
		//UnarchiveZipFile(fullPath)
	case "use":
		log.Fatal("Not implemented yet.")
	default:
		log.Fatal("Help not implemented yet.")
	}
}
