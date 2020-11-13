package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/mvndaai/ctxerr"
	"github.com/mvndaai/sweepstakes/internal/config"
	"github.com/spf13/viper"
	"github.com/tebeka/selenium"
)

type (
	sweepstake struct {
		Disabled bool       `json:"disabled"`
		Start    civil.Date `json:"start"`
		End      civil.Date `json:"end"`
		URL      string     `json:"url"`
		Pages    []page     `json:"pages"`
		FinalURL string     `json:"finalURL"`
		Filename string     `json:"-"`
	}

	page struct {
		Iframe element `json:"iframe"`
		Inputs []input `json:"inputs"`
		Clicks []click `json:"clicks"`
		Next   button  `json:"next"`
	}

	element struct {
		By    string `json:"by"`
		Value string `json:"value"`
	}

	input struct {
		element
		ConfigVar string `json:"configVar"`
	}

	click struct {
		element
		Selected bool `json:"Selected"`
	}

	button struct {
		element
	}
)

func main() {
	ctx := context.Background()

	if err := config.InitConfig(ctx); err != nil {
		ctxerr.Handle(err)
		return
	}

	sweeps, err := getSweepstakes(ctx, "./pages")
	if err != nil {
		ctxerr.Handle(err)
		return
	}

	opts := []selenium.ServiceOption{
		//	selenium.StartFrameBuffer(),           // Start an X frame buffer for the browser to run in.
		//	selenium.GeckoDriver(geckoDriverPath), // Specify the path to GeckoDriver in order to use Firefox.
		// selenium.Output(os.Stderr), // Output debug information to STDERR.
	}
	port, err := pickUnusedPort()
	if err != nil {
		ctxerr.Handle(err)
		return
	}
	service, err := selenium.NewChromeDriverService("./drivers/chromedriver", port, opts...)
	if err != nil {
		ctxerr.Handle(err)
		return
	}
	defer service.Stop()
	// selenium.SetDebug(true)

	seleniumURL := "http://127.0.0.1:" + strconv.Itoa(port) + "/wd/hub"

	var entered int
	for _, sweep := range sweeps {
		if sweep.Disabled {
			fmt.Println("skipping disabled file: ", sweep.Filename)
			continue
		}

		ctx := ctxerr.SetField(ctx, "file", sweep.Filename)
		if err := sweep.enter(ctx, seleniumURL); err != nil {
			ctxerr.Handle(err)
			continue
		}
		fmt.Println("successfully entered", sweep.Filename)
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

func (e element) FindElement(ctx context.Context, wd selenium.WebDriver) (selenium.WebElement, error) {
	if e.By == "" {
		e.By = selenium.ByCSSSelector
	}

	var el selenium.WebElement
	if err := wd.WaitWithTimeout(
		func(wd selenium.WebDriver) (bool, error) {
			var err error
			el, err = wd.FindElement(e.By, e.Value)
			if err != nil {
				if strings.Contains(err.Error(), "Unable to locate element") {
					return false, nil
				}
				return false, err
			}
			return el.IsDisplayed()
		},
		10*time.Second,
	); err != nil {
		ctx = ctxerr.SetField(ctx, "by", e.By)
		ctx = ctxerr.SetField(ctx, "value", e.Value)
		return nil, ctxerr.QuickWrap(ctx, err)
	}

	return el, nil
}

func (s sweepstake) enter(ctx context.Context, seleniumURL string) error {
	if err := validDataRange(ctx, s.Start, s.End); err != nil {
		return ctxerr.QuickWrap(ctx, err)
	}

	wd, err := startWebdriver(ctx, seleniumURL)
	if err != nil {
		return ctxerr.QuickWrap(ctx, err)
	}
	defer wd.Quit()

	if err := wd.Get(s.URL); err != nil {
		return ctxerr.QuickWrap(ctx, err)
	}

	for _, p := range s.Pages {
		if p.Iframe.Value != "" {
			iframe, err := p.Iframe.FindElement(ctx, wd)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "", "could not find iframe")
			}

			if err := wd.SwitchFrame(iframe); err != nil {
				return ctxerr.Wrap(ctx, err, "", "could not switch to iframe")
			}
		}

		for _, in := range p.Inputs {
			ctx := ctxerr.SetField(ctx, "config-variable", in.ConfigVar)
			el, err := in.FindElement(ctx, wd)
			if err != nil {
				return ctxerr.QuickWrap(ctx, err)
			}

			v := viper.GetString(in.ConfigVar)
			if v == "" {
				return ctxerr.New(ctx, "", "no value to enter")
			}

			if err := el.SendKeys(v); err != nil {
				return ctxerr.QuickWrap(ctx, err)
			}
		}

		for _, ch := range p.Clicks {
			el, err := ch.FindElement(ctx, wd)
			if err != nil {
				return ctxerr.QuickWrap(ctx, err)
			}

			if err := el.Click(); err != nil {
				return ctxerr.QuickWrap(ctx, err)
			}
		}

		el, err := p.Next.FindElement(ctx, wd)
		if err != nil {
			return ctxerr.QuickWrap(ctx, err)
		}

		if err := el.Click(); err != nil {
			return ctxerr.Wrap(ctx, err, "", "could not click next button")
		}

		if p.Iframe.Value != "" {
			if err := wd.SwitchFrame(nil); err != nil {
				return ctxerr.Wrap(ctx, err, "", "could not leave iframe")
			}
		}
	}

	time.Sleep(2 * time.Second)

	if err := wd.WaitWithTimeout(
		func(wd selenium.WebDriver) (bool, error) {
			cURL, err := wd.CurrentURL()
			if err != nil {
				return false, err
			}
			return strings.HasPrefix(cURL, s.FinalURL), nil
		},
		10*time.Second,
	); err != nil {
		ctx = ctxerr.SetField(ctx, "currentURL", func() string { cURL, _ := wd.CurrentURL(); return cURL })
		return ctxerr.QuickWrap(ctx, err)
	}
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

func startWebdriver(ctx context.Context, seleniumURL string) (selenium.WebDriver, error) {
	caps := selenium.Capabilities{"browserName": "chrome"}
	wd, err := selenium.NewRemote(caps, seleniumURL)
	if err != nil {
		ctx = ctxerr.SetField(ctx, "url", seleniumURL)
		return nil, ctxerr.QuickWrap(ctx, err)
	}
	return wd, nil
}

// https://github.com/tebeka/selenium/issues/103
func pickUnusedPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		return 0, err
	}
	return port, nil
}
