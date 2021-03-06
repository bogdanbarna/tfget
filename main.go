package main

import (
	"archive/zip"
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

const releasesUrl = "https://releases.hashicorp.com/terraform/"
const tfgetHome = "$HOME/.tfget/versions"

// Print current version
func WhichVersion(dirPath string) {
	currentTerraformVersion, _ := os.Readlink(filepath.Join(dirPath, "terraform"))
	log.Info(currentTerraformVersion)
}

// Check if there's a globally-installed terraform, exit if so
// else symlink version to ${tfgetHome}/versions/terraform
func SwitchVersion(dirPath, terraformVersion string) {
	systemTerraformPath := "/usr/local/bin/terraform"
	if _, err := os.Stat(systemTerraformPath); os.IsNotExist(err) {
		// Check if version is present locally; download if not
		targetVersionPath := dirPath + "/terraform_" + terraformVersion
		if _, err := os.Stat(targetVersionPath); os.IsNotExist(err) {
			log.WithFields(log.Fields{
				"terraformVersion": terraformVersion,
			}).
				Info("Version not found locally. Downloading it now")
			downloadErr := DownloadTerraform(dirPath, terraformVersion)
			if downloadErr != nil {
				log.Fatal(downloadErr)
			}
		}

		symlinkPath := filepath.Join(dirPath, "terraform")
		// First remove existing symlink
		if _, err := os.Lstat(symlinkPath); err == nil {
			os.Remove(symlinkPath)
		}
		// Create new symlink
		err = os.Symlink(targetVersionPath, symlinkPath)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("To use this version, make sure you've added tfgetHome to PATH")
		log.WithFields(log.Fields{
			"TFGET_HOME": tfgetHome,
			"PATH":       os.Getenv("PATH"),
		}).Info("Example: export PATH=\"$HOME/$TFGET_HOME:$PATH\"")

		log.WithFields(log.Fields{
			"terraformVersion": terraformVersion,
			"symlinkPath":      symlinkPath,
		}).Info("Switched to specified version")
	} else {
		log.Fatal("Detected system-wide Terraform installation. Exiting")
	}
}

// Runs after Download()
func UnzipTerraformArchive(fullPath string) {
	fullPathZip := fullPath + ".zip"

	log.WithFields(log.Fields{
		"fullPathZip": fullPathZip,
	}).Info("Unzipping")
	zipReader, _ := zip.OpenReader(fullPathZip)
	for _, file := range zipReader.Reader.File {
		zippedFile, err := file.Open()
		if err != nil {
			log.Fatal(err)
		}
		defer zippedFile.Close()

		unzippedFile, err := os.OpenFile(
			fullPath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			file.Mode(),
		)
		if err != nil {
			log.Fatal(err)
		}
		defer unzippedFile.Close()

		for {
			_, err = io.CopyN(unzippedFile, zippedFile, 1024)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
			}
		}
	}
	err := os.Remove(fullPathZip)
	if err != nil {
		log.Fatal(err)
	}
}

// Download Terraform archive, then call UnzipTerraformArchive()
func DownloadTerraform(dirPath, terraformVersion string) error {
	filePath := "terraform_" + terraformVersion
	fullPath := dirPath + "/" + filePath
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fullPathZip := fullPath + ".zip"
		platform := runtime.GOOS + "_" + runtime.GOARCH
		terraformUrl := releasesUrl + terraformVersion + "/terraform_" + terraformVersion + "_" + platform + ".zip"

		log.WithFields(log.Fields{
			"terraformUrl": terraformUrl,
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
		out, err := os.Create(fullPathZip)
		if err != nil {
			return err
		}
		defer out.Close()

		// Write local file body
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}

		// Unzip and remove archive
		UnzipTerraformArchive(fullPath)

		log.WithFields(log.Fields{
			"filepath": fullPath,
		}).Info("Terraform version now on disk")
	} else {
		log.WithFields(log.Fields{
			"terraformVersion": terraformVersion,
			"fullPath":         fullPath,
		}).Info("Version already exists on disk")
	}
	return nil
}

// Crawl the releases page and get a list of released versions
func ListRemoteVersions() []string {
	// Handle HTTP request
	client := &http.Client{}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	req, err := http.NewRequestWithContext(ctx, "GET", releasesUrl, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

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
					terraformVersion := strings.Split(t.Data, "_")[1]
					versions = append(versions, terraformVersion)
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
	// Readdir(n int) reads the contents of the directory associated with file and returns a slice of up to n FileInfo values,
	// as would be returned by Lstat, in directory order.
	// Subsequent calls on the same file will yield further FileInfos.
	// If n > 0, Readdir() returns at most n FileInfo structures.
	// In this case, if Readdir() returns an empty slice, it will return a non-nil error explaining why.
	// At the end of a directory, the error is io.EOF.
	// If n <= 0, Readdir() returns all the FileInfo from the directory in a single slice
	// (Explication shamelessly copied from https://golang.cafe/blog/how-to-list-files-in-a-directory-in-go.html)
	files, err := fp.Readdir(0)
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range files {
		log.Println(v.Name())
	}
}

// Parse CLI argument and check against a list of versions
func DetermineVersion(cliArg string, versions []string) string {
	var terraformVersion string

	if cliArg != "" {
		if cliArg == "latest" {
			terraformVersion = versions[0]
		} else {
			foundit := false
			for _, aVersion := range versions {
				if strings.Contains(aVersion, cliArg) {
					terraformVersion = aVersion
					foundit = true
					break
				}
			}
			if foundit {
				terraformVersion = cliArg
			} else {
				log.WithFields(log.Fields{
					"terraformVersion": terraformVersion,
				}).Fatal("Version not found ")
			}
		}
	} else {
		log.Fatal("No CLI arguments found")
	}

	return terraformVersion
}

// Ensure local cache is present (see tfgetHome constant)
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
		log.Info("Listing all local versions")
		ListLocal(dirPath)
	case "list-remote":
		log.Info("Listing all remote versions")
		versions := ListRemoteVersions()
		for _, v := range versions {
			log.Info(v)
		}
	case "download":
		terraformVersion := DetermineVersion(os.Args[2], ListRemoteVersions())
		log.Infof("Downloading Terraform version %v", terraformVersion)
		downloadErr := DownloadTerraform(dirPath, terraformVersion)
		if downloadErr != nil {
			log.Fatal(downloadErr)
		}
	case "switch", "use":
		terraformVersion := DetermineVersion(os.Args[2], ListRemoteVersions())
		SwitchVersion(dirPath, terraformVersion)
	case "which", "which-version":
		WhichVersion(dirPath)
	default:
		log.Fatal("Help not implemented yet.")
	}
}
