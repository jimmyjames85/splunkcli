package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/jimmyjames85/splunkcli/pkg/splunk"
	"github.com/pkg/errors"
)

func promptHidden(format string, args ...interface{}) string {
	fmt.Printf(format, args...)
	answer, err := gopass.GetPasswd()
	if err != nil {
		panic(err)
	}
	return string(answer)
}

func prompt(format string, args ...interface{}) string {
	fmt.Printf(format, args...)
	var answer string
	fmt.Scanln(&answer) // ignores error
	return answer
}

func exists(fileloc string) bool { _, err := os.Stat(fileloc); return !os.IsNotExist(err) }

func DoInit(fileloc string) (*splunk.Client, error) {

	if exists(fileloc) {
		// prompt if file exists
		resp := strings.ToLower(strings.TrimSpace(prompt("Init file already exists. Do you want to overwrite: ")))
		if len(resp) == 0 || resp[0] != 'y' {
			return nil, fmt.Errorf("user aborted")
		}
	}

	addr := prompt("Splunk Address: ") // https://localhost:8089
	cli := splunk.New(addr)
	_, err := DoCreateSessionID(cli)
	if err != nil {
		return nil, err
	}
	err = cli.SaveTo(fileloc)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func DoCreateSessionID(cli *splunk.Client) (string, error) {
	user := prompt("username: ")
	pass := promptHidden("password: ")
	sid, err := cli.RenewSessionID(user, pass)
	if err != nil {
		return "", errors.Wrapf(err, "unable to authenticate with %s", cli.Addr)
	}
	return sid, nil
}

func mustLoadClient(fileloc string) *splunk.Client {
	var cli *splunk.Client

	_, err := os.Stat(fileloc)
	if os.IsNotExist(err) {
		cli, err = DoInit(fileloc)
	} else {
		cli, err = splunk.LoadClient(fileloc)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create config file: %s: %s\n", fileloc, err.Error())
		os.Exit(-1)
	}
	return cli
}

func printHelp() {
	fmt.Printf("help - TODO\n")
}

func exitf(code int, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(code)
}

// panic if error -- to be removed
func mustBeNil(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	////////// load config location
	fileloc := fmt.Sprintf("%s/.splunk", os.Getenv("HOME"))
	cli := mustLoadClient(fileloc)

	if len(os.Args) < 2 {
		printHelp()
		exitf(-1, "Please provide an argument\n")
	}

	command := os.Args[1]
	switch command {
	case "init":
		_, err := DoInit(fileloc)
		if err != nil {
			exitf(-1, "%s\n", err.Error())
		}
		return
	case "login":
		_, err := DoCreateSessionID(cli)
		mustBeNil(err)
		err = cli.SaveTo(fileloc)
		mustBeNil(err)
	case "search":
		if len(os.Args) < 3 {
			exitf(-1, "Please provide search\n")
		}
		search := os.Args[2] // fmt.Sprintf("search earliest=-1h host=*filter* event=FilterReceived OR event=processed OR event=drop")

		if strings.Index(strings.ToLower(search), "earliest") == -1 {
			fmt.Printf("%s\n", search)
			exitf(-1, "please specify time range: TODO get url or documentation\n")
		}
		r, err := cli.Search(search)
		mustBeNil(err)
		fmt.Printf("{\"searchID\": %q}\n", r.SearchID)
		cli.SaveTo(fileloc)
	case "clear":

		err := cli.ClearKnownSearches()
		mustBeNil(err)
		cli.SaveTo(fileloc)
	case "status":
		if len(os.Args) < 3 {
			exitf(-1, "Please provide search ID\n")
		}
		sid := os.Args[2]
		r, err := cli.GetSearchStatus(sid)
		if r.AuthFailed() {
			exitf(-1, "auth failed: perhaps session expired")
		}
		mustBeNil(err)
		fmt.Printf("%s\n", string(r.Body))
	case "results":
		if len(os.Args) < 3 {
			exitf(-1, "Please provide search ID\n")
		}
		sid := os.Args[2]
		r, err := cli.GetSearchResults(sid, splunk.WithParam("count", "0")) // 0 means get all results https://docs.splunk.com/Documentation/Splunk/7.2.3/RESTREF/RESTsearch#search.2Fjobs.2F.7Bsearch_id.7D.2Fresults
		if r.AuthFailed() {
			exitf(-1, "auth failed: perhaps session expired")
		}
		mustBeNil(err)
		fmt.Printf("%s\n", string(r.Body))
	default:
		printHelp()
		exitf(-1, "unkown cmd: %s", command)
		os.Exit(-1)
	}
}
