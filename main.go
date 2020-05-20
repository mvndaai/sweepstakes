package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/mvndaai/ctxerr"
	"github.com/tebeka/selenium"
)

type (
	sweepstake struct {
		Start    civil.Date `json:"start"`
		End      civil.Date `json:"end"`
		URL      string     `json:"url"`
		Pages    []page     `json:"pages"`
		FinalURL string     `json:"finalURL"`
		Filename string     `json:"-"`
	}

	page struct {
		Inputs     []input    `json:"inputs"`
		Checkboxes []checkbox `json:"checkboxes"`
		Next       button     `json:"next"`
	}

	element struct {
		By    string `json:"by"`
		Value string `json:"value"`
	}

	input struct {
		element
		ConfigValue string `json:"configValue"`
		Optional    bool   `json:"optional"`
	}

	checkbox struct {
		element
		Selected bool `json:"Selected"`
	}

	button struct {
		element
	}
)

func main() {
	ctx := context.Background()

	if err := initConfig(ctx); err != nil {
		ctxerr.Handle(err)
		return
	}

	sweeps, err := getSweepstakes(ctx, "./pages")
	if err != nil {
		ctxerr.Handle(err)
		return
	}

	// fmt.Println(sweeps)

	var entered int
	for _, sweep := range sweeps {
		ctx := ctxerr.SetField(ctx, "file", sweep.Filename)
		if err := sweep.enter(ctx); err != nil {
			ctxerr.Handle(err)
			continue
		}
		entered++
	}

	if l := len(sweeps); l != entered {
		fmt.Printf("Entered %v of %v\n", entered, len(sweeps))
	} else {
		fmt.Printf("Entered all sweepstates (%v)\n", entered)
	}
}

func getSweepstakes(ctx context.Context, directoryPath string) ([]sweepstake, error) {
	var files []string
	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		ctxerr.QuickWrap(ctx, err)
	}

	var sweeps []sweepstake

	for _, file := range files {
		if !strings.HasSuffix(file, ".json") {
			continue
		}

		s, err := getSweepstake(ctx, file)
		if err != nil {
			return nil, ctxerr.QuickWrap(ctx, err)
		}
		sweeps = append(sweeps, s)
	}

	return sweeps, nil
}

func getSweepstake(ctx context.Context, file string) (sweepstake, error) {
	ctx = ctxerr.SetField(ctx, "filepath", file)

	var sweep sweepstake
	f, err := os.Open(file)
	if err != nil {
		return sweep, ctxerr.QuickWrap(ctx, err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return sweep, ctxerr.QuickWrap(ctx, err)
	}

	if err := json.Unmarshal([]byte(b), &sweep); err != nil {
		return sweep, ctxerr.QuickWrap(ctx, err)
	}

	sweep.Filename = file
	return sweep, nil
}

func (s sweepstake) enter(ctx context.Context) error {
	if err := validDataRange(ctx, s.Start, s.End); err != nil {
		return ctxerr.QuickWrap(ctx, err)
	}

	// TODO check the date

	// Get this working
	// wd, err := startWebdriver(ctx, s.URL)
	// if err != nil {
	// 	return ctxerr.QuickWrap(ctx, err)
	// }
	// defer wd.Quit()

	// loop through pages

	// Check that it is on the final page

	return nil
}

func validDataRange(ctx context.Context, start, end civil.Date) error {
	today := civil.DateOf(time.Now())
	if start.IsValid() {
		if today.Before(start) {
			ctx = ctxerr.SetField(ctx, "start", start.String())
			return ctxerr.New(ctx, "", "sweepstake hasn't started")
		}
	}

	if end.IsValid() {
		if today.After(end) {
			ctx = ctxerr.SetField(ctx, "end", end.String())
			return ctxerr.New(ctx, "", "sweepstake is over")
		}
	}

	return nil
}

func startWebdriver(ctx context.Context, startURL string) (selenium.WebDriver, error) {

	seleniumPath := "vendor/selenium-server-standalone-3.4.jar"
	port := 8080

	service, err := selenium.NewSeleniumService(seleniumPath, port)
	if err != nil {
		return nil, ctxerr.QuickWrap(ctx, err)
	}
	defer service.Stop()

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "chrome"}
	wd, err := selenium.NewRemote(caps, startURL)
	if err != nil {
		return nil, ctxerr.QuickWrap(ctx, err)
	}
	return wd, nil
}
