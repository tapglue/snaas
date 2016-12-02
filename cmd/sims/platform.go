package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/service/app"
)

var (
	ErrInvalidNamespace = errors.New("namespace invalid")
)

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
