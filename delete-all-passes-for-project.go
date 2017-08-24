// delete-all-passes-for-project
package main

import (
	//"bytes"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type requestResult struct {
	p []byte
	e error
}

type GetProjectResponse struct {
	Templates []struct {
		ProjectType string    `json:"projectType"`
		Description string    `json:"description"`
		ExternalID  string    `json:"externalId"`
		VendorID    int       `json:"vendorId"`
		Type        string    `json:"type"`
		CreatedAt   time.Time `json:"createdAt"`
		Deleted     string    `json:"deleted"`
		Vendor      string    `json:"vendor"`
		Name        string    `json:"name"`
		Disabled    string    `json:"disabled"`
		ID          string    `json:"id"`
		ProjectID   int       `json:"projectId"`
		UpdatedAt   time.Time `json:"updatedAt"`
	} `json:"templates"`
}

func printCommand(cmd *exec.Cmd) {
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
}

func printError(err error) {
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("==> Error: %s\n", err.Error()))
	}
}

func printOutput(outs []byte) {
	if len(outs) > 0 {
		fmt.Printf("==> Output: %s\n", string(outs))
	}
}

func doRequest(done chan<- requestResult, client *http.Client, req *http.Request) {
	rs, err := client.Do(req)
	if err != nil {
		done <- requestResult{e: err}
		return
	}
	defer rs.Body.Close()

	p, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		done <- requestResult{e: err}
		return
	}
	done <- requestResult{p: p}
}

func PassProviderRequest(method string, servicePath string, authKey string) *http.Request {
	req, _ := http.NewRequest(method, "https://wallet-api.urbanairship.com/v1"+servicePath, nil)
	req.Header.Add("Api-Revision", "1.2")
	if authKey != "" {
		req.Header.Add("Authorization", "Basic "+authKey)
	}
	return req
}

func main() {
	var authKey, projectId string
	var interactive bool

	flag.StringVar(&authKey, "authKey", "", "[optional] The authKey is the email and api key combined into a string [YOUR_EMAIL]:[YOUR_KEY] and base64 encoded. If supplied as a command-line option, it is used to create the HTTP Authorization header for Basic Authentication (I personally use a local proxy to add the Auth header instead, refer to https://www.charlesproxy.com/documentation/tools/rewrite/)")
	flag.StringVar(&projectId, "projectId", "", "[required] The projectId which you want to delete all passes from")
	flag.BoolVar(&interactive, "interactive", true, "[optional] only run deletions one template at a time (requires user to press enter for each template)")

	flag.Parse()

	req := PassProviderRequest("GET", "/project/"+projectId, authKey)
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	done := make(chan requestResult, 1)

	go doRequest(done, client, req)

	res := <-done
	if res.e != nil {
		panic(res.e)
	}

	projectResponse := GetProjectResponse{}
	json.Unmarshal(res.p, &projectResponse)

	fmt.Println("Project has " + strconv.Itoa(cap(projectResponse.Templates)) + " template(s) whose passes will be deleted")

	scanner := bufio.NewScanner(os.Stdin)

	for _, template := range projectResponse.Templates {
		if interactive == true {
			fmt.Println("Getting ready to run delete-all-passes-for-templateId with templateId=" + template.ID)
			fmt.Println("Press enter key to continue...")
			scanner.Scan() // blocks until user presses enter
		}

		cmd := exec.Command("delete-all-passes-for-templateId", "-templateId="+template.ID)

		// Combine stdout and stderr
		printCommand(cmd)
		output, err := cmd.CombinedOutput()
		printError(err)
		printOutput(output) // => go version go1.3 darwin/amd64
		if err != nil {
			panic(err)
		}
		if interactive != true {
			time.Sleep(time.Millisecond * 100)
		}
	}
}
