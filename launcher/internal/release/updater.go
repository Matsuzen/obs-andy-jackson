package release

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

type GithubRelease struct {
	TagName string `json:"tag_name"`
	Assets []struct {
		Name string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type Updater struct {
	ApiUrl string
	CurrentTagName string
}

func NewUpdater(currentTagName string) *Updater {
	return &Updater{
		ApiUrl: "https://api.github.com/repos/matsuzen/obs-andy-jackson/releases",
		CurrentTagName: currentTagName,
	}
}

func (u *Updater) GetLatestRelease() (*GithubRelease, error) {
	res, err := http.Get(fmt.Sprintf("%s/latest", u.ApiUrl))
	if err != nil {
		fmt.Printf("Error fetching latest release: %v\n", err)
		return nil, err
	}
	defer res.Body.Close()

	var release GithubRelease
	json.NewDecoder(res.Body).Decode(&release)

	return &release, nil
}

func (u *Updater) Apply(release *GithubRelease) error {
    if release.TagName == u.CurrentTagName {
        return errors.New("Already up to date")
    }

    assetName := fmt.Sprintf("launcher-%s-%s", runtime.GOOS, runtime.GOARCH)
    if runtime.GOOS == "windows" {
        assetName += ".exe"
    }

    var downloadURL string
    for _, asset := range release.Assets {
        if asset.Name == assetName {
            downloadURL = asset.BrowserDownloadURL
          	break
        }
    }

    execPath, _ := os.Executable()
    tmpPath := execPath + ".new"

	out, err := os.Create(tmpPath)
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating temp file for new release: %s\n", err.Error()))
	}
	defer out.Close()
	res, err := http.Get(downloadURL)
	if err != nil {
		return errors.New(fmt.Sprintf("Error downloading new release: %s\n", err.Error()))

	}
	defer res.Body.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		return errors.New(fmt.Sprintf("Error copying new release to temp file: %s\n", err))
	}

    if runtime.GOOS == "windows" {
        oldPath := execPath + ".old"
        os.Rename(execPath, oldPath)
        os.Rename(tmpPath, execPath)
    } else {
        os.Rename(tmpPath, execPath)
        os.Chmod(execPath, 0755)
    }

	u.CurrentTagName = release.TagName
	return nil
  }

