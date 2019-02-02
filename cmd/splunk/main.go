package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

func mustLoadClient() *splunk.Client {

	cli, err := splunk.LoadClient(configLocation)
	if err != nil {
		exitf(-1, "failed to load config file: %s: %s\ntry %s init\n", configLocation, err.Error(), os.Args[0])
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

func StatusHelp() string {
	return "status HELP"
}

var configLocation = fmt.Sprintf("%s/.splunk", os.Getenv("HOME"))

var ErrCantParseDateFromSID = fmt.Errorf("Can't Parse Date From SID")

func timestampFromSID(sid string) (int64, error) {
	spl := strings.Split(sid, ".")
	if len(spl) > 0 {
		return strconv.ParseInt(spl[0], 10, 64)
	}
	return 0, ErrCantParseDateFromSID
}

func StatusAll() {
	cli := mustLoadClient()
	for i, s := range cli.Searches {
		date := "            UNKNOWN"
		ts, err := timestampFromSID(s.SearchID)
		if err == nil {
			date = time.Unix(ts, 0).Format("Jan 2 2006 15:04:05")
		}
		fmt.Printf("%d: %s %s: %s\n", i, date, s.SearchID, s.Search) // todo display date
	}
	println()
	return
}

func Status(args []string) {
	if len(args) == 0 {
		StatusAll()
		return
	}

	cli := mustLoadClient()
	var raw bool
	var sid string
	for _, arg := range args {
		switch arg {
		case "--raw":
			raw = true
		default:
			sid = arg
			// attempt to use reference number from StatusAll
			i, err := strconv.Atoi(sid)
			if err == nil && i < len(cli.Searches) {
				sid = cli.Searches[i].SearchID
			}
		}
	}

	r, err := cli.GetSearchStatus(sid)
	if r.AuthFailed() {
		exitf(-1, "auth failed: perhaps session expired\n")
	}
	mustBeNil(err) // TODO
	if raw {
		fmt.Printf("%s\n", string(r.Body))
		return
	}

	if r.StatusCode < 200 || r.StatusCode >= 300 {
		exitf(-1, "Non-200: response: %s\n", r.Body)
	}

	var resp splunk.StatusResponse
	err = json.Unmarshal(r.Body, &resp)
	mustBeNil(err) // todo
	for _, e := range resp.Entry {
		c := e.Content
		status := "  FINISHED"
		if !c.IsDone {
			status = "unfinished"
			if c.DoneProgress == 1.0 {
				status = "unknown"
			}
		}
		ttl := c.TTL
		ts, err := timestampFromSID(c.Sid)
		if err == nil {
			ttl = int(ts + int64(c.TTL) - time.Now().Unix())
		}

		fmt.Printf("%0.2f %d/%d [ttl = %d]\t%s %s\n", c.DoneProgress, c.ResultCount, c.ResultPreviewCount, ttl, c.Sid, status)
	}

	// TODO

}

func Results(args []string) {
	cli := mustLoadClient()
	var raw bool
	var sid string
	var count int // 0 means get all results https://docs.splunk.com/Documentation/Splunk/7.2.3/RESTREF/RESTsearch#search.2Fjobs.2F.7Bsearch_id.7D.2Fresults

	// todo creaete ParseArgsResults
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--raw":
			raw = true
		case "-c", "--count":
			i++
			if i >= len(args) {
				exitf(-1, "please provide count\n")
			}
			c, err := strconv.Atoi(args[i])
			if err != nil {
				exitf(-1, "Invalid count\n")
			}
			count = c
		default:
			sid = args[i]
			// attempt to use reference number from StatusAll
			s, err := strconv.Atoi(sid)
			if err == nil && s < len(cli.Searches) {
				sid = cli.Searches[s].SearchID
			}
		}
	}
	if len(sid) == 0 {
		exitf(-1, "Please provide search ID\n")
	}

	r, err := cli.GetSearchResults(sid, splunk.WithParam("count", fmt.Sprintf("%d", count)))
	if r.AuthFailed() {
		exitf(-1, "auth failed: perhaps session expired\n")
	}
	mustBeNil(err) // todo
	if raw {
		// TODO rename raw to something more applicabile
		fmt.Printf("%s\n", string(r.Body))
		return
	}
	var resp splunk.ResultsResponse
	err = json.Unmarshal(r.Body, &resp)
	mustBeNil(err) // todo
	for _, r := range resp.Results {
		var m map[string]interface{} // TODO is there a way for user to specify the structure type here? can we search for something specific? way in the future... perhaps a --highlight flag that bolds specified keys
		err = json.Unmarshal([]byte(r.Raw_), &m)
		if err != nil {
			fmt.Printf("%s\n", r.Raw_)
			continue
		}

		m["_host"] = r.Host
		m["_time"] = r.Time_

		byts, err := json.Marshal(m)
		if err != nil {
			fmt.Printf("%s\n", r.Raw_)
			continue
		}
		fmt.Printf("%s\n", string(byts))
	}
}

func main() {
	////////// load config location

	if len(os.Args) < 2 {
		printHelp()
		exitf(-1, "Please provide an argument\n")
	}

	cli := mustLoadClient()
	command := os.Args[1]
	switch command {
	case "init":
		_, err := DoInit(configLocation)
		if err != nil {
			exitf(-1, "%s\n", err.Error())
		}
		return
	case "login":
		_, err := DoCreateSessionID(cli)
		mustBeNil(err)
		err = cli.SaveTo(configLocation)
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
		cli.SaveTo(configLocation)
	case "clear":
		err := cli.ClearKnownSearches()
		mustBeNil(err)
		cli.SaveTo(configLocation)
	case "status":
		Status(os.Args[2:])
	case "results":
		Results(os.Args[2:])
	default:
		printHelp()
		exitf(-1, "unkown cmd: %s\n", command)
		os.Exit(-1)
	}
}
