package main

import (
	"archive/zip"
	"bufio"
	"context"
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

func SwitchVersion(terraformVersion string) {
	/*
		Check if there's a globally-installed terraform, exit if so
		else symlink version to ${tfgetHome}/versions/terraform
	*/
	systemTerraformPath := "/usr/local/bin/terraform"
	if _, err := os.Stat(systemTerraformPath); os.IsNotExist(err) {
		// if version not found, download it

		// TODO DRY this
		// Replace $HOME with actual user home
		tfgetHomeFull := tfgetHome
		if strings.Contains(tfgetHome, "$HOME") {
			dirname, homeErr := os.UserHomeDir()
			if homeErr != nil {
				log.Fatal(homeErr)
			}
			tfgetHomeFull = strings.Replace(tfgetHome, "$HOME", dirname, -1)
		}
		//

		// Check if version is present locally
		// download if not
		if _, err := os.Stat(systemTerraformPath); os.IsNotExist(err) {
			log.WithFields(log.Fields{
				"terraformVersion": terraformVersion,
			}).
				Info("Version not found locally. Downloading it now")
			downloadErr := DownloadTerraform(tfgetHomeFull, terraformVersion)
			if downloadErr != nil {
				log.Fatal(downloadErr)
			}
		}
		targetVersionPath := tfgetHomeFull + "/terraform_" + terraformVersion

		symlinkPath := filepath.Join(tfgetHomeFull, "terraform")
		// First remove existing symlink
		if _, err := os.Lstat(symlinkPath); err == nil {
			os.Remove(symlinkPath)
		}
		// Create new symlink
		os.Symlink(targetVersionPath, symlinkPath)

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
		// TODO https://stackoverflow.com/questions/67327323/g110-potential-dos-vulnerability-via-decompression-bomb-gosec
		_, err = io.Copy(unzippedFile, zippedFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	err := os.Remove(fullPathZip)
	if err != nil {
		log.Fatal(err)
	}
}

func DownloadTerraform(dirPath string, version_number string) error {
	filePath := "terraform_" + version_number
	fullPath := dirPath + "/" + filePath
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fullPathZip := fullPath + ".zip"
		platform := runtime.GOOS + "_" + runtime.GOARCH
		terraformUrl := releasesUrl + version_number + "/terraform_" + version_number + "_" + platform + ".zip"

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
			"version_number": version_number,
			"fullPath":       fullPath,
		}).Info("Version already exists on disk")
	}
	return nil
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
		log.Info("Listing all local versions")
		ListLocal(dirPath)
	case "list-remote":
		log.Info("Listing all remote versions")
		versions := ListRemoteVersions()
		for _, v := range versions {
			log.Info(v)
		}
	case "download":
		version_number := DetermineVersion(os.Args[2], ListRemoteVersions())
		log.Infof("Downloading Terraform version %v", version_number)
		downloadErr := DownloadTerraform(dirPath, version_number)
		if downloadErr != nil {
			log.Fatal(downloadErr)
		}
	case "switch", "use":
		terraformVersion := DetermineVersion(os.Args[2], ListRemoteVersions())
		SwitchVersion(terraformVersion)
	default:
		log.Fatal("Help not implemented yet.")
	}
}
