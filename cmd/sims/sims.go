package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/go-kit/kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tapglue/snaas/core"
	"github.com/tapglue/snaas/platform/metrics"
	platformSNS "github.com/tapglue/snaas/platform/sns"
	platformSQS "github.com/tapglue/snaas/platform/sqs"
	"github.com/tapglue/snaas/service/app"
	"github.com/tapglue/snaas/service/connection"
	"github.com/tapglue/snaas/service/device"
	"github.com/tapglue/snaas/service/event"
	"github.com/tapglue/snaas/service/object"
	"github.com/tapglue/snaas/service/user"
)

// Logging and telemetry identifiers.
const (
	component        = "sims"
	namespaceService = "service"
	namespaceSource  = "source"
	sourceService    = "sqs"
	storeService     = "postgres"
	subsystemQueue   = "queue"
)

// Queue names.
const (
	queueEndpointChanges = "endpoint-changes"
)

// Buildtime vars.
var (
	revision = "0000000-dev"
)

func main() {
	var (
		begin = time.Now()
		pApps = platformApps{}

		awsID         = flag.String("aws.id", "", "Identifier for AWS requests")
		awsRegion     = flag.String("aws.region", "us-east-1", "AWS region to operate in")
		awsSecret     = flag.String("aws.secret", "", "Identification secret for AWS requests")
		postgresURL   = flag.String("postgres.url", "", "Postgres URL to connect to")
		telemetryAddr = flag.String("telemetry.addr", ":9001", "Address to expose telemetry on")
	)
	flag.Var(&pApps, "app", "Repeated platform apps.")
	flag.Parse()

	logger := log.NewContext(
		log.NewJSONLogger(os.Stdout),
	).With(
		"caller", log.Caller(3),
		"component", component,
		"revision", revision,
	)

	hostname, err := os.Hostname()
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
	}

	logger = log.NewContext(logger).With("host", hostname)

	// Setup instrumentation.
	go func(addr string) {
		logger.Log(
			"duration", time.Now().Sub(begin).Nanoseconds(),
			"lifecycle", "start",
			"listen", addr,
			"sub", "telemetry",
		)

		http.Handle("/metrics", prometheus.Handler())

		err := http.ListenAndServe(addr, nil)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort", "sub", "telemetry")
			os.Exit(1)
		}
	}(*telemetryAddr)

	serviceErrCount, serviceOpCount, serviceOpLatency := metrics.KeyMetrics(
		namespaceService,
		metrics.FieldComponent,
		metrics.FieldMethod,
		metrics.FieldNamespace,
		metrics.FieldService,
		metrics.FieldStore,
	)

	sourceFieldKeys := []string{
		metrics.FieldComponent,
		metrics.FieldMethod,
		metrics.FieldNamespace,
		metrics.FieldSource,
		metrics.FieldStore,
	}

	sourceErrCount, sourceOpCount, sourceOpLatency := metrics.KeyMetrics(
		namespaceSource,
		sourceFieldKeys...,
	)

	sourceQueueLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespaceSource,
			Subsystem: subsystemQueue,
			Name:      "latency_seconds",
			Help:      "Distribution of message queue latency in seconds",
			Buckets:   metrics.BucketsQueue,
		},
		sourceFieldKeys,
	)
	prometheus.MustRegister(sourceQueueLatency)

	// Setup clients.
	var (
		aSession = awsSession.New(&aws.Config{
			Credentials: credentials.NewStaticCredentials(*awsID, *awsSecret, ""),
			Region:      aws.String(*awsRegion),
		})
		snsAPI = sns.New(aSession)
		sqsAPI = sqs.New(aSession)
	)

	pgClient, err := sqlx.Connect(storeService, *postgresURL)
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}

	// Setup services.
	var apps app.Service
	apps = app.PostgresService(pgClient)
	apps = app.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(apps)
	apps = app.LogServiceMiddleware(logger, storeService)(apps)

	var connections connection.Service
	connections = connection.PostgresService(pgClient)
	connections = connection.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(connections)
	connections = connection.LogServiceMiddleware(logger, storeService)(connections)

	var devices device.Service
	devices = device.PostgresService(pgClient)
	devices = device.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(devices)
	devices = device.LogServiceMiddleware(logger, storeService)(devices)

	var objects object.Service
	objects = object.PostgresService(pgClient)
	objects = object.InstrumentServiceMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(objects)
	objects = object.LogServiceMiddleware(logger, storeService)(objects)

	var users user.Service
	users = user.PostgresService(pgClient)
	users = user.InstrumentMiddleware(
		component,
		storeService,
		serviceErrCount,
		serviceOpCount,
		serviceOpLatency,
	)(users)
	users = user.LogMiddleware(logger, storeService)(users)

	// Setup sources.
	conSource, err := connection.SQSSource(sqsAPI)
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}
	conSource = connection.InstrumentSourceMiddleware(
		component,
		sourceService,
		sourceErrCount,
		sourceOpCount,
		sourceOpLatency,
		sourceQueueLatency,
	)(conSource)
	conSource = connection.LogSourceMiddleware(sourceService, logger)(conSource)

	eventSource, err := event.SQSSource(sqsAPI)
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}
	eventSource = event.InstrumentSourceMiddleware(
		component,
		sourceService,
		sourceErrCount,
		sourceOpCount,
		sourceOpLatency,
		sourceQueueLatency,
	)(eventSource)
	eventSource = event.LogSourceMiddleware(sourceService, logger)(eventSource)

	objectSource, err := object.SQSSource(sqsAPI)
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}
	objectSource = object.InstrumentSourceMiddleware(
		component,
		sourceService,
		sourceErrCount,
		sourceOpCount,
		sourceOpLatency,
		sourceQueueLatency,
	)(objectSource)
	objectSource = object.LogSourceMiddleware(sourceService, logger)(objectSource)

	logger.Log(
		"duration", time.Now().Sub(begin).Nanoseconds(),
		"lifecycle", "start",
		"sub", "worker",
	)

	// React to SNS endpoint changes.
	qName, err := queueName(sqsAPI, queueEndpointChanges)
	if err != nil {
		logger.Log("err", err, "lifecycle", "abort")
		os.Exit(1)
	}

	changec := make(chan endpointChange)

	go func() {
		err := consumeEndpointChange(sqsAPI, qName, changec)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort")
			os.Exit(1)
		}
	}()

	go func() {
		for c := range changec {
			a, err := appForARN(core.AppFetch(apps), pApps, c.Resource)
			if err != nil {
				if isPlatformNotFound(err) {
					continue
				}

				logger.Log("err", err, "lifecycle", "abort")
				os.Exit(1)
			}

			err = endpointUpdate(core.DeviceDisable(devices), a, c)
			if err != nil {
				logger.Log("err", err, "lifecycle", "abort")
				os.Exit(1)
			}
		}
	}()

	// Consume entity state changes.
	batchc := make(chan batch)

	go func() {
		err := consumeConnection(
			core.AppFetch(apps),
			conSource,
			batchc,
			conRuleFollower(
				core.UserFetch(users),
			),
			conRuleFriendConfirmed(
				core.UserFetch(users),
			),
			conRuleFriendRequest(
				core.UserFetch(users),
			),
		)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort")
			os.Exit(1)
		}
	}()

	go func() {
		err := consumeEvent(
			core.AppFetch(apps),
			eventSource,
			batchc,
			eventRuleLikeCreated(
				core.ConnectionFollowerIDs(connections),
				core.ConnectionFriendIDs(connections),
				core.PostFetch(objects),
				core.UserFetch(users),
				core.UsersFetch(users),
			),
		)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort")
			os.Exit(1)
		}
	}()

	go func() {
		err := consumeObject(
			core.AppFetch(apps),
			objectSource,
			batchc,
			objectRuleCommentCreated(
				core.ConnectionFollowerIDs(connections),
				core.ConnectionFriendIDs(connections),
				core.PostFetch(objects),
				core.UserFetch(users),
				core.UsersFetch(users),
			),
			objectRulePostCreated(
				core.ConnectionFollowerIDs(connections),
				core.ConnectionFriendIDs(connections),
				core.UserFetch(users),
				core.UsersFetch(users),
			),
		)
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort")
			os.Exit(1)
		}
	}()

	// Distribute messages to channels.
	cs := []channelFunc{
		channelPush(
			core.DeviceListUser(devices),
			core.DeviceSyncEndpoint(
				devices,
				platformSNS.EndpointCreate(snsAPI),
				platformSNS.EndpointRetrieve(snsAPI),
				platformSNS.EndpointUpdate(snsAPI),
			),
			platformSNS.Push(snsAPI),
			pApps,
		),
	}

	for batch := range batchc {
		for _, msg := range batch.messages {
			for _, channel := range cs {
				err := channel(batch.app, msg)
				if err != nil {
					logger.Log("err", err, "lifecycle", "abort")
					os.Exit(1)
				}
			}
		}

		err = batch.ackFunc()
		if err != nil {
			logger.Log("err", err, "lifecycle", "abort")
			os.Exit(1)
		}
	}

	logger.Log("lifecycle", "stop")
}

func queueName(api platformSQS.API, name string) (string, error) {
	res, err := api.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(name),
	})
	if err != nil {
		return "", err
	}

	return *res.QueueUrl, nil
}
