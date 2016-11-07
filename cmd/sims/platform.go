package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/device"
)

var (
	ErrInvalidNamespace = errors.New("namespace invalid")
	ErrPlatformNotFound = errors.New("platform not found")
)

type platformApp struct {
	ARN       string `json:"arn"`
	Namespace string `json:"namespace"`
	Scheme    string `json:"scheme"`
	Platform  int    `json:"platform"`
}

type platformApps []*platformApp

func (as *platformApps) Set(input string) error {
	a := &platformApp{}
	err := json.Unmarshal([]byte(input), a)
	if err != nil {
		return err
	}

	*as = append(*as, a)

	return nil
}

func (as *platformApps) String() string {
	return fmt.Sprintf("%d apps", len(*as))
}

func appForARN(
	appFetch core.AppFetchFunc,
	pApps platformApps,
	pARN string,
) (*app.App, error) {
	var a *platformApp

	for _, pa := range pApps {
		if pa.ARN == pARN {
			a = pa
			break
		}
	}

	if a == nil {
		return nil, ErrPlatformNotFound
	}

	id, err := namespaceToID(a.Namespace)
	if err != nil {
		return nil, err
	}

	return appFetch(id)
}

func appForNamespace(appFetch core.AppFetchFunc, ns string) (*app.App, error) {
	id, err := namespaceToID(ns)
	if err != nil {
		return nil, err
	}

	return appFetch(id)
}

func namespaceToID(ns string) (uint64, error) {
	ps := strings.SplitN(ns, "_", 2)

	if len(ps) != 2 {
		return 0, ErrInvalidNamespace
	}

	return strconv.ParseUint(ps[1], 10, 64)
}

func isPlatformNotFound(err error) bool {
	return err == ErrPlatformNotFound
}

func platformAppForPlatform(
	pApps platformApps,
	a *app.App,
	p device.Platform,
) (*platformApp, error) {
	var pApp *platformApp

	for _, pa := range pApps {
		if pa.Namespace == a.Namespace() && device.Platform(pa.Platform) == p {
			pApp = pa
			break
		}
	}

	if pApp == nil {
		return nil, ErrPlatformNotFound
	}

	return pApp, nil
}
