package main

import (
	"runtime"
	"os"
	"io"
	"net/http"
	"context"
	"time"
	"bufio"
	"strings"
	"sort"
	"regexp"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

const releasesUrl = "https://releases.hashicorp.com/terraform/"

//func UnarchiveZipFile(filepath string) {
//	log.Println("Unzipping", filepath)
//}

func DownloadFile(filepath string, url string) error {
	// Handle HTTP request
	client := &http.Client{}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
	return err
}

func ListRemoteVersions() []string {
	// Handle HTTP request
	client := &http.Client{}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
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
			log.WithFields(log.Fields{
				"versions": versions,
			}).Info("Parsed remote versions")
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
	if !found_it {
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
		log.WithFields(log.Fields{
			"dirPath": dirPath,
		}).Info("Directory not found, creating")
	  mkdirErr := os.Mkdir(dirPath, 0700)
		if mkdirErr != nil {
			log.Fatal(mkdirErr)
		}
	}
	return dirPath
}

func init() {
	// Log as JSON instead of the default ASCII text
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above
	log.SetLevel(log.InfoLevel)
}

func ValidateVersion(version string) bool {
	if version == "latest" {
		return true
	}
	version_number_pattern := `[01]\.\d+?(\.\d)?`
	matched, regexErr := regexp.Match(version_number_pattern, []byte(version))
	if regexErr != nil {
		log.Fatal(regexErr)
	}
	return matched
}

func main() {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	var version_number string

	cliArgs := os.Args
	if len(cliArgs) > 1 {
		if !ValidateVersion(cliArgs[1]) {
			log.Fatal("Not a valid version")
		}
		version_number = cliArgs[1]
	} else {
		versions := ListRemoteVersions()
		version_number = GetVersion(versions)
	}

	filePath := "terraform_" + version_number
	dirPath := mkdirLocalCache("$HOME/.tfget")
	fullPath := dirPath + "/" + filePath
	fullPathZip := fullPath + ".zip"

	if _, err := os.Stat(fullPathZip); os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"version_number": version_number,
		}).Info("Downloading Terraform version")
		fileUrl := releasesUrl + version_number + "/terraform_" + version_number + "_" + platform + ".zip"

		downloadErr := DownloadFile(fullPathZip, fileUrl)
		if downloadErr != nil {
			log.Fatal(downloadErr)
		}
		log.WithFields(log.Fields{
			"fullPathZip": fullPathZip,
			"version_number": version_number,
		}).Info("Downloaded Terraform version on disk")
	} else {
		log.WithFields(log.Fields{
			"version_number": version_number,
		}).Info("Version already exists on disk")
	}

	//UnarchiveZipFile(fullPath)
}
