package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/jmoiron/sqlx"

	"github.com/tapglue/snaas/platform/pg"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/device"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/session"
	"github.com/tapglue/snaas/service/user"
)

const (
	cmdExport = "export"
	cmdImport = "import"

	component = "dataz"

	fileApps        = "apps.json"
	fileConnections = "connections.json"
	fileDevices     = "devices.json"
	fileEvents      = "events.json"
	fileObjects     = "objects.json"
	fileSessions    = "sessions.json"
	fileUsers       = "users.json"

	revision = "0000000-dev"

	storeService = "postgres"
)

var enabled = true

func main() {
	var (
		dataDir     = flag.String("data.dir", "", "Directory which holds the data files.")
		namespace   = flag.String("namespace", "", "App namespace for data export")
		postgresURL = flag.String("postgres.url", "", "Postgres URL to connect to.")
	)
	flag.Parse()

	logger := log.With(
		log.NewJSONLogger(os.Stdout),
		"caller", log.Caller(3),
		"component", component,
		"revision", revision,
	)

	hostname, err := os.Hostname()
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}

	logger = log.With(logger, "host", hostname)
	if len(flag.Args()) != 1 {
		logger.Log("err", "missing command", "lifecycle", "abort")
		os.Exit(1)
	}

	pgClient, err := sqlx.Connect(storeService, *postgresURL)
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}

	var (
		apps        = app.PostgresService(pgClient)
		connections = connection.PostgresService(pgClient)
		devices     = device.PostgresService(pgClient)
		events      = event.PostgresService(pgClient)
		objects     = object.PostgresService(pgClient)
		sessions    = session.PostgresService(pgClient)
		users       = user.PostgresService(pgClient)
	)

	switch flag.Args()[0] {
	case cmdImport:
		app, err := importApp(apps, *dataDir)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", cmdImport)
			os.Exit(1)
		}

		logger.Log("app", app.Namespace(), "sub", cmdImport)

		conCount, err := importConnections(connections, *dataDir, app.Namespace())
		if err != nil {
			logger.Log(
				"count_connection", conCount,
				"err", err,
				"lifecycle", "abort",
				"sub", cmdImport,
			)
			os.Exit(1)
		}

		logger.Log("count_connection", conCount, "sub", cmdImport)

		devCount, err := importDevices(devices, *dataDir, app.Namespace())
		if err != nil {
			logger.Log(
				"count_device", devCount,
				"err", err,
				"lifecycle", "abort",
				"sub", cmdImport,
			)
			os.Exit(1)
		}

		logger.Log("count_device", devCount, "sub", cmdImport)

		evCount, err := importEvents(events, *dataDir, app.Namespace())
		if err != nil {
			logger.Log(
				"count_event", evCount,
				"err", err,
				"lifecycle", "abort",
				"sub", cmdImport,
			)
			os.Exit(1)
		}

		logger.Log("count_event", evCount, "sub", cmdImport)

		objCount, err := importObjects(objects, *dataDir, app.Namespace())
		if err != nil {
			logger.Log(
				"count_object", objCount,
				"err", err,
				"lifecycle", "abort",
				"sub", cmdImport,
			)
			os.Exit(1)
		}

		logger.Log("count_object", objCount, "sub", cmdImport)

		sessCount, err := importSessions(sessions, *dataDir, app.Namespace())
		if err != nil {
			logger.Log(
				"count_session", sessCount,
				"err", err,
				"lifecycle", "abort",
				"sub", cmdImport,
			)
			os.Exit(1)
		}

		logger.Log("count_session", sessCount, "sub", cmdImport)

		userCount, err := importUsers(users, *dataDir, app.Namespace())
		if err != nil {
			logger.Log(
				"count_user", userCount,
				"err", err,
				"lifecycle", "abort",
				"sub", cmdImport,
			)
			os.Exit(1)
		}

		logger.Log("count_user", userCount, "sub", cmdImport)
	case cmdExport:
		err := os.MkdirAll(*dataDir, os.ModePerm)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", cmdExport)
			os.Exit(1)
		}

		err = exportApp(apps, *dataDir, *namespace)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", cmdExport)
			os.Exit(1)
		}

		err = exportConnections(connections, *dataDir, *namespace)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", cmdExport)
			os.Exit(1)
		}

		err = exportDevices(devices, *dataDir, *namespace)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", cmdExport)
			os.Exit(1)
		}

		err = exportEvents(events, *dataDir, *namespace)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", cmdExport)
			os.Exit(1)
		}

		err = exportObjects(objects, *dataDir, *namespace)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", cmdExport)
			os.Exit(1)
		}

		err = exportSessions(sessions, *dataDir, *namespace)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", cmdExport)
			os.Exit(1)
		}

		err = exportUsers(users, *dataDir, *namespace)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", cmdExport)
			os.Exit(1)
		}
	default:
		logger.Log(
			"err", fmt.Sprintf("unknown command '%s'", flag.Args()[0]),
			"lifecycle", "abort",
		)
		os.Exit(1)
	}
}

func exportApp(apps app.Service, dir, ns string) error {
	ps := strings.SplitN(ns, "_", 2)

	if len(ps) != 2 {
		return fmt.Errorf("invalid namespace: %s", ns)
	}

	id, err := strconv.ParseUint(ps[1], 10, 64)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(dir, fileApps))
	if err != nil {
		return err
	}

	out := json.NewEncoder(f)

	as, err := apps.Query(pg.MetaNamespace, app.QueryOptions{
		Enabled: &enabled,
		IDs: []uint64{
			id,
		},
	})
	if err != nil {
		return err
	}

	for _, a := range as {
		err := out.Encode(a)
		if err != nil {
			return err
		}
	}

	return nil
}

func exportConnections(connections connection.Service, dir, ns string) error {
	f, err := os.Create(filepath.Join(dir, fileConnections))
	if err != nil {
		return err
	}

	out := json.NewEncoder(f)

	cs, err := connections.Query(ns, connection.QueryOptions{
		Enabled: &enabled,
	})
	if err != nil {
		return err
	}

	for _, con := range cs {
		err := out.Encode(con)
		if err != nil {
			return err
		}
	}

	return nil
}

func exportDevices(devices device.Service, dir, ns string) error {
	f, err := os.Create(filepath.Join(dir, fileDevices))
	if err != nil {
		return err
	}

	out := json.NewEncoder(f)

	ds, err := devices.Query(ns, device.QueryOptions{})
	if err != nil {
		return err
	}

	for _, d := range ds {
		err := out.Encode(d)
		if err != nil {
			return err
		}
	}

	return nil
}

func exportEvents(events event.Service, dir, ns string) error {
	f, err := os.Create(filepath.Join(dir, fileEvents))
	if err != nil {
		return err
	}

	out := json.NewEncoder(f)

	es, err := events.Query(ns, event.QueryOptions{
		Enabled: &enabled,
	})
	if err != nil {
		return err
	}

	for _, ev := range es {
		err := out.Encode(ev)
		if err != nil {
			return err
		}
	}

	return nil
}

func exportObjects(objects object.Service, dir, ns string) error {
	f, err := os.Create(filepath.Join(dir, fileObjects))
	if err != nil {
		return err
	}

	out := json.NewEncoder(f)

	ls, err := objects.Query(ns, object.QueryOptions{})
	if err != nil {
		return err
	}

	for _, o := range ls {
		err := out.Encode(o)
		if err != nil {
			return err
		}
	}

	return nil
}

func exportSessions(sessions session.Service, dir, ns string) error {
	f, err := os.Create(filepath.Join(dir, fileSessions))
	if err != nil {
		return err
	}

	out := json.NewEncoder(f)

	ss, err := sessions.Query(ns, session.QueryOptions{
		Enabled: &enabled,
	})
	if err != nil {
		return err
	}

	for _, s := range ss {
		err := out.Encode(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func exportUsers(users user.Service, dir, ns string) error {
	f, err := os.Create(filepath.Join(dir, fileUsers))
	if err != nil {
		return err
	}

	out := json.NewEncoder(f)

	us, err := users.Query(ns, user.QueryOptions{
		Enabled: &enabled,
	})
	if err != nil {
		return err
	}

	for _, u := range us {
		err := out.Encode(u)
		if err != nil {
			return err
		}
	}

	return nil
}

func importApp(apps app.Service, dir string) (*app.App, error) {
	f, err := os.Open(filepath.Join(dir, fileApps))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	a := &app.App{}

	err = json.NewDecoder(f).Decode(a)
	if err != nil {
		return nil, err
	}

	a, err = apps.Put(pg.MetaNamespace, a)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func importConnections(connections connection.Service, dir, ns string) (int, error) {
	f, err := os.Open(filepath.Join(dir, fileConnections))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var (
		count = 0
		dec   = json.NewDecoder(f)
	)

	for dec.More() {
		c := &connection.Connection{}

		err := dec.Decode(c)
		if err != nil {
			return count, err
		}

		_, err = connections.Put(ns, c)
		if err != nil {
			return count, err
		}

		count++
	}

	return count, nil
}

func importDevices(devices device.Service, dir, ns string) (int, error) {
	f, err := os.Open(filepath.Join(dir, fileDevices))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var (
		count = 0
		dec   = json.NewDecoder(f)
	)

	for dec.More() {
		d := &device.Device{}

		err := dec.Decode(d)
		if err != nil {
			return count, err
		}

		_, err = devices.Put(ns, d)
		if err != nil {
			return count, err
		}

		count++
	}

	return count, nil
}

func importEvents(events event.Service, dir, ns string) (int, error) {
	f, err := os.Open(filepath.Join(dir, fileEvents))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var (
		count = 0
		dec   = json.NewDecoder(f)
	)

	for dec.More() {
		e := &event.Event{}

		err := dec.Decode(e)
		if err != nil {
			return count, err
		}

		e.ID = 0

		_, err = events.Put(ns, e)
		if err != nil {
			return count, err
		}

		count++
	}

	return count, nil
}

func importObjects(objects object.Service, dir, ns string) (int, error) {
	f, err := os.Open(filepath.Join(dir, fileObjects))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var (
		count = 0
		dec   = json.NewDecoder(f)
	)

	for dec.More() {
		o := &object.Object{}

		err := dec.Decode(o)
		if err != nil {
			return count, err
		}

		o.ID = 0

		_, err = objects.Put(ns, o)
		if err != nil {
			return count, err
		}

		count++
	}

	return count, nil
}

func importSessions(sessions session.Service, dir, ns string) (int, error) {
	f, err := os.Open(filepath.Join(dir, fileSessions))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var (
		count = 0
		dec   = json.NewDecoder(f)
	)

	for dec.More() {
		s := &session.Session{}

		err := dec.Decode(s)
		if err != nil {
			return count, err
		}

		_, err = sessions.Put(ns, s)
		if err != nil {
			return count, err
		}

		count++
	}

	return count, nil
}

func importUsers(users user.Service, dir, ns string) (int, error) {
	f, err := os.Open(filepath.Join(dir, fileUsers))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var (
		count = 0
		dec   = json.NewDecoder(f)
	)

	for dec.More() {
		u := &user.User{}

		err := dec.Decode(u)
		if err != nil {
			return count, err
		}

		u.ID = 0

		_, err = users.Put(ns, u)
		if err != nil {
			fmt.Printf("%#v\n", u)
			return count, err
		}

		count++
	}

	return count, nil
}
