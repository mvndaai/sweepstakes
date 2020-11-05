package main

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/mvndaai/ctxerr"
)

func main() {
	ctx := context.Background()
	cv, err := getChromeVersionMac(ctx)
	if err != nil {
		ctxerr.Handle(err)
		return
	}
	fmt.Println("chrome version", cv)

	cdv, err := getChromedriverVersion(ctx)
	if err != nil {
		ctxerr.Handle(err)
		return
	}
	fmt.Println("chromedriver version", cdv)

}

func getChromeVersionMac(ctx context.Context) (string, error) {
	path := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	v, err := runCommand(ctx, path, "--version")
	if err != nil {
		return "", ctxerr.QuickWrap(ctx, err)
	}
	return strings.TrimSpace(v), nil
}

func getChromedriverVersion(ctx context.Context) (string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", ctxerr.New(ctx, "", "could not get caller")
	}
	path := path.Join(path.Dir(filename), "./chromedriver")
	v, err := runCommand(ctx, path, "--version")
	if err != nil {
		return "", ctxerr.QuickWrap(ctx, err)
	}
	return strings.TrimSpace(v), nil
}

func runCommand(ctx context.Context, path string, args ...string) (string, error) {
	c := exec.Command(path, args...)
	b, err := c.Output()
	if err != nil {
		return "", ctxerr.QuickWrap(ctx, err)
	}

	return string(b), nil
}
