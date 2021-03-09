package main

import (
	"encoding/json"
	"errors"
	"github.com/spf13/pflag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

var (
	flags = pflag.NewFlagSet("flags", pflag.ExitOnError)
)

func init() {
	pflag.String("url", "", "Pterodactyl Panel URL")
	pflag.String("apiKey", "", "Pterodactyl API Key")
	pflag.String("serverId", "", "Pterodactyl Server ID")
	pflag.String("backupDir", filepath.Join(".", "backups"), "Directory where the backups are stored")
	pflag.Parse()

	flags.AddFlag(pflag.Lookup("url"))
	flags.AddFlag(pflag.Lookup("apiKey"))
	flags.AddFlag(pflag.Lookup("serverId"))
	flags.AddFlag(pflag.Lookup("backupDir"))
}

func main() {
	urlFlag, err := flags.GetString("url")
	check(err)
	pterodactylUrl, err := url.Parse(urlFlag)
	check(err)
	apiKey, err := flags.GetString("apiKey")
	check(err)
	serverId, err := flags.GetString("serverId")
	check(err)
	backupDir, err := flags.GetString("backupDir")
	check(err)

	client := http.Client{}

	req, err := http.NewRequest("GET", pterodactylUrl.String()+"api/client/servers/"+serverId+"/backups", nil)
	check(err)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiKey)

	res, err := client.Do(req)
	check(err)

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	check(err)

	backups := backups{}
	err = json.Unmarshal(body, &backups)
	check(err)

	if len(backups.Data) == 0 {
		log.Fatal(errors.New("no backups available"))
	}

	targetBackup := backups.Data[len(backups.Data)-1]

	if !targetBackup.Attributes.IsSuccessful {
		log.Fatal(errors.New("latest pterodactyl backup was not successful"))
	}

	req, err = http.NewRequest("GET", pterodactylUrl.String()+"api/client/servers/"+serverId+"/backups/"+targetBackup.Attributes.Uuid+"/download", nil)
	check(err)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiKey)

	res, err = client.Do(req)
	check(err)

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err = ioutil.ReadAll(res.Body)
	check(err)

	backupDownload := backupDownload{}
	err = json.Unmarshal(body, &backupDownload)
	check(err)

	resp, err := http.Get(backupDownload.Attributes.Url)
	check(err)
	defer resp.Body.Close()

	path := filepath.Join(backupDir, serverId)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		mkDirErr := os.MkdirAll(path, os.ModePerm)
		check(mkDirErr)
	}

	t, err := time.Parse(time.RFC3339, targetBackup.Attributes.CreatedAt)
	check(err)

	out, err := os.Create(path + "/" + t.Format("2006-01-02_15.04.05") + ".tar.gz")
	check(err)
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	check(err)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type backup struct {
	Attributes struct {
		Uuid         string `json:"uuid"`
		Name         string `json:"name"`
		IsSuccessful bool   `json:"is_successful"`
		Checksum     string `json:"checksum"`
		Bytes        int64  `json:"bytes"`
		CreatedAt    string `json:"created_at"`
		CompletedAt  string `json:"completed_at"`
	} `json:"attributes"`
}

type backups struct {
	Data []backup `json:"data"`
}

type backupDownload struct {
	Attributes struct {
		Url string `json:"url"`
	} `json:"attributes"`
}
