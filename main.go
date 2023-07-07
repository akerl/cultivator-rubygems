package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	"github.com/akerl/cultivator/plugin"
)

var filePattern = regexp.MustCompile(`^(Gemfile|.*\.gemspec)$`)
var pattern = regexp.MustCompile(`^(.*(?:_dependency|gem)) '([\w]+)', '~> ([\d.]+)'$`)

func run(_ string) error {
	doBundlerUpdate := false

	files, err := os.ReadDir(".")
	if err != nil {
		return err
	}

	for _, file := range files {
		if filePattern.MatchString(file.Name()) {
			err := plugin.FindReplace(file.Name(), pattern, gemCheck)
			if err != nil {
				return err
			}
		} else if file.Name() == "Gemfile.lock" {
			doBundlerUpdate = true
		}
	}

	if !doBundlerUpdate {
		return nil
	}

	cmd := exec.Command("bundle", "update")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(string(out))
	}
	return nil
}

type gem struct {
	Version string `json:"version"`
}

func gemCheck(matches []string) string {
	apiURL := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", matches[2])

	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return matches[0]
	}

	resp, err := client.Do(req)
	if err != nil {
		return matches[0]
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return matches[0]
	}
	var g gem
	err = json.Unmarshal(body, &g)
	if err != nil {
		return matches[0]
	}

	return fmt.Sprintf("%s '%s', '~> %s'", matches[1], matches[2], g.Version)
}

func main() {
	p := plugin.Plugin{
		Commit: plugin.SimpleCommit(
			"Update Ruby gems",
			"cultivator/update-rubygems",
			"Update Ruby dependencies in gemspec and gemfile",
			"update ruby gems",
		),
		Executor: run,
	}

	p.Run()
}
