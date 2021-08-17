package main

import (
	"runtime"
	"os"
	"io"
	"net/http"
	"bufio"
	"strings"
	"sort"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

func UnarchiveZipFile(filepath string) {
	log.Println("Unzipping", filepath)
}

func DownloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
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
	return err
}

func ListRemoteVersions(releasesUrl string) ([]string, error) {
	resp, err := http.Get(releasesUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var versions []string

	// https://html.spec.whatwg.org/multipage/parsing.html#tokenization
	z := html.NewTokenizer(bufio.NewReader(resp.Body))
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken: // Reached EOF
			// Sort from latest to oldest
			sort.Sort(sort.Reverse(sort.StringSlice(versions)))
			return versions, nil
		case html.StartTagToken:
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

func GetVersion(versions []string) string {
	var version_number string

	// index, element
	found_it := false
	for _, a_version := range versions {
		if strings.Contains(a_version, version_number) {
			version_number = a_version
			found_it = true
			break
		}
	}
	if found_it == false {
		log.Fatal("Version not found ", version_number)
	}
	return version_number
}

// Local cache
func mkdirLocalCache(dirPath string ) string {
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
		log.Println("Directory not found, creating", dirPath)
	  mkdirErr := os.Mkdir(dirPath, 0700)
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
	releasesUrl := "https://releases.hashicorp.com/terraform/"
	platform := runtime.GOOS + "_" + runtime.GOARCH
	var version_number string

	cliArgs := os.Args
	if len(cliArgs) > 1 && cliArgs[1] != "latest" {
		// TODO parse version
		version_number = cliArgs[1]
	} else {
		versions, err := ListRemoteVersions(releasesUrl)
		if err != nil {
			log.Fatal(err)
		}
		version_number = GetVersion(versions)
	}

	filePath := "terraform_" + version_number
	dirPath := mkdirLocalCache("$HOME/.tfget")
	fullPath := dirPath + "/" + filePath
	fullPathZip := fullPath + ".zip"

	if _, err := os.Stat(fullPathZip); os.IsNotExist(err) {
		log.Println("Downloading Terraform version", version_number)
		fileUrl := releasesUrl + version_number + "/terraform_" + version_number + "_" + platform + ".zip"

		downloadErr := DownloadFile(fullPathZip, fileUrl)
		if downloadErr != nil {
			log.Fatal(downloadErr)
		}
		log.Println("Downloaded " + fullPathZip)
	} else {
		log.Println("Version", version_number, "already exists on disk")
	}

	//UnarchiveZipFile(fullPath)
}
