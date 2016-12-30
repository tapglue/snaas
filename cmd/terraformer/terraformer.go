package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"golang.org/x/crypto/ssh"

	"github.com/tapglue/snaas/platform/generate"
)

const (
	argBackend       = "-backend"
	argBackendConfig = "-backend-config"
	argConfig        = "config"
	argDestroy       = "-destroy"
	argOut           = "-out"
	argState         = "-state"
	argVarFile       = "-var-file"

	binaryTerraform = "terraform"

	cmdSetup    = "setup"
	cmdTeardown = "teardown"
	cmdUpdate   = "update"

	defaultKeyPath      = "access.pem"
	defaultStatesPath   = "infrastructure/terraform/states"
	defaultTemplatePath = "infrastructure/terraform/template"
	defaultTmpPath      = "/tmp"

	fmtBucket      = "bucket=%s"
	fmtBucketState = "%s-snaas-state"
	fmtKey         = "key=%s/%s.tfstate"
	fmtNamespace   = "%s-%s"
	fmtPlan        = "%s/%s.plan"
	fmtRegion      = "region=%s"
	fmtStateFile   = "%s.tfstate"
	fmtVarsFile    = "%s.tfvars"
	fmtTFVar       = "TF_VAR_%s=%s"

	remoteBackendS3 = "s3"

	tfCmdApply   = "apply"
	tfCmdDestroy = "destroy"
	tfCmdPlan    = "plan"
	tfCmdRemote  = "remote"

	tplTFVars = `domain = "{{.Domain}}"
key = {
	access = "{{.KeyAccess}}"
}
pg_password = "{{.PGPassword}}"
google_client_id = "{{.GoogleID}}"
google_client_secret = "{{.GoogleSecret}}"`

	varAccount = "account"
	varEnv     = "env"
	varRegion  = "region"
)

// vars bundles together all generated or given input that is custom to the env.
type vars struct {
	Domain       string
	GoogleID     string
	GoogleSecret string
	KeyAccess    string
	PGPassword   string
}

func main() {
	var (
		env          = flag.String("env", "", "Environment used for isolation.")
		region       = flag.String("region", "", "AWS region to deploy to.")
		statesPath   = flag.String("states.path", defaultStatesPath, "Location to store env states.")
		stateRemote  = flag.Bool("state.remote", false, "Control if state is stored remotely in s3.")
		templatePath = flag.String("template.path", defaultTemplatePath, "Location of the infrastructure template.")
		tmpPath      = flag.String("tmp.path", defaultTmpPath, "Location for temporary output like plans.")
		varsPath     = flag.String("vars.path", "", "Location of vars file.")
	)
	flag.Parse()

	log.SetFlags(log.Lshortfile)

	awsSession, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.EnvProvider{},
				&credentials.SharedCredentialsProvider{},
			},
		),
		Region: aws.String(*region),
	})
	if err != nil {
		log.Fatalf("%#v\n", err)
	}

	if len(flag.Args()) != 1 {
		log.Fatal("provide command: setup, teardown, update")
	}

	if *env == "" {
		log.Fatal("provide env")
	}

	if *region == "" {
		log.Fatal("provide region")
	}

	if _, err := exec.LookPath(binaryTerraform); err != nil {
		log.Fatal("terraform must be in your PATH")
	}

	account, err := awsAcoount(awsSession)
	if err != nil {
		log.Fatalf("AWS account fetch failed: %s", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("getting current directory failed: %s", err)
	}

	var (
		namespace = fmt.Sprintf(fmtNamespace, *env, *region)
		planFile  = fmt.Sprintf(fmtPlan, *tmpPath, namespace)
		statePath = filepath.Join(cwd, *statesPath, namespace)
		stateFile = filepath.Join(statePath, fmt.Sprintf(fmtStateFile, namespace))
		tmplPath  = filepath.Join(cwd, *templatePath)
		varFile   = filepath.Join(statePath, fmt.Sprintf(fmtVarsFile, namespace))
		environ   = append(
			os.Environ(),
			tfVar(varAccount, account),
			tfVar(varEnv, *env),
			tfVar(varRegion, *region),
		)
	)

	if *varsPath != "" {
		varFile = *varsPath
	}

	update := func() {
		args := []string{
			argOut, planFile,
			argState, stateFile,
			argVarFile, varFile,
			tmplPath,
		}

		if err := prepareCmd(statePath, environ, tfCmdPlan, args...).Run(); err != nil {
			os.Exit(1)
		}

		fmt.Println("\nWant to apply the plan? (type 'yes')")
		fmt.Print("(no) |> ")

		response := "no"
		fmt.Scanf("%s", &response)

		if response != "yes" {
			os.Exit(1)
		}

		args = []string{
			argState, stateFile,
			planFile,
		}

		if err := prepareCmd(statePath, environ, tfCmdApply, args...).Run(); err != nil {
			os.Exit(1)
		}
	}

	switch flag.Args()[0] {
	case cmdSetup:
		if _, err := os.Stat(stateFile); err == nil {
			log.Fatalf("state file already exists: %s", stateFile)
		}

		if err := os.MkdirAll(statePath, os.ModePerm); err != nil {
			log.Fatalf("state dir creation failed: %s", err)
		}

		fmt.Println("\nWhat is the domain the env should be reachable at?")
		fmt.Print("|> ")
		domain := ""
		fmt.Scanf("%s", &domain)

		if domain == "" {
			log.Fatal("Can't work without a domain.")
		}

		fmt.Println("\nIn order to guard the monitoring setup we need Google OAuth credentials.\nWhat is your Google client ID?")
		fmt.Print("|> ")
		googleID := ""
		fmt.Scanf("%s", &googleID)

		if googleID == "" {
			log.Fatal("Can't work without a Google OAuth credentials.")
		}

		fmt.Println("\nWhat is your Google client Secret?")
		fmt.Print("|> ")
		googleSecret := ""
		fmt.Scanf("%s", &googleSecret)

		if googleSecret == "" {
			log.Fatal("Can't work without a Google OAuth credentials.")
		}

		pubKey, err := generateKeyPair(filepath.Join(statePath, defaultKeyPath))
		if err != nil {
			log.Fatal(err)
		}

		if err = generateVarFile(varFile, vars{
			Domain:       domain,
			GoogleID:     googleID,
			GoogleSecret: googleSecret,
			KeyAccess:    strings.Trim(string(pubKey), "\n"),
			PGPassword:   generate.RandomStringSafe(32),
		}); err != nil {
			log.Fatalf("var file create failed: %s", err)
		}

		if *stateRemote {
			var (
				bucket = fmt.Sprintf(fmtBucketState, account)
				svcS3  = s3.New(awsSession, aws.NewConfig().WithRegion(*region))
			)

			_, err = svcS3.HeadBucket(&s3.HeadBucketInput{
				Bucket: aws.String(bucket),
			})
			if err != nil {
				if awsErr, ok := err.(awserr.RequestFailure); ok &&
					awsErr.StatusCode() == 404 {
					_, err := svcS3.CreateBucket(&s3.CreateBucketInput{
						Bucket: aws.String(bucket),
					})
					if err != nil {
						log.Fatalf("bucket create failed: %s", err)
					}

					_, err = svcS3.PutBucketVersioning(&s3.PutBucketVersioningInput{
						Bucket: aws.String(bucket),
						VersioningConfiguration: &s3.VersioningConfiguration{
							Status: aws.String("Enabled"),
						},
					})
					if err != nil {
						if awsErr, ok := err.(awserr.RequestFailure); ok {
							log.Fatalf("bucket versioning failed: %s", awsErr.Error())
						}
						log.Fatalf("bucket versioning failed: %s", err)
					}
				} else {
					log.Fatalf("bucket check failed: %s", err)
				}
			}

			args := []string{
				argConfig,
				argBackend, remoteBackendS3,
				argBackendConfig, fmt.Sprintf(fmtRegion, *region),
				argBackendConfig, fmt.Sprintf(fmtBucket, bucket),
				argBackendConfig, fmt.Sprintf(fmtKey, *region, *region),
				argState, stateFile,
			}

			if err := prepareCmd(statePath, environ, tfCmdRemote, args...).Run(); err != nil {
				os.Exit(1)
			}
		}

		update()
	case cmdTeardown:
		args := []string{
			argDestroy,
			argOut, planFile,
			argState, stateFile,
			argVarFile, varFile,
			tmplPath,
		}

		if err := prepareCmd(statePath, environ, tfCmdPlan, args...).Run(); err != nil {
			os.Exit(1)
		}

		args = []string{
			argState, stateFile,
			argVarFile, varFile,
			tmplPath,
		}

		if err := prepareCmd(statePath, environ, tfCmdDestroy, args...).Run(); err != nil {
			os.Exit(1)
		}
	case cmdUpdate:
		if _, err := os.Stat(stateFile); !*stateRemote && err != nil {
			log.Fatalf("couldn't locate state file: %s", err)
		}

		update()
	default:
		log.Fatalf("unknown command '%s'", flag.Args()[0])
	}
}

func awsAcoount(sess *session.Session) (string, error) {
	stsService := sts.New(sess)

	res, err := stsService.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return *res.Account, nil
}

func generateKeyPair(privateKeyPath string) ([]byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	privateFile, err := os.OpenFile(privateKeyPath, os.O_CREATE|os.O_WRONLY, 0400)
	if err != nil {
		return nil, err
	}
	defer privateFile.Close()

	privatePEM := &pem.Block{
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		Type:  "RSA PRIVATE KEY",
	}
	if err := pem.Encode(privateFile, privatePEM); err != nil {
		return nil, err
	}

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return ssh.MarshalAuthorizedKey(pub), nil
}

func generateVarFile(path string, vs vars) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	tmpl, err := template.New("vars").Parse(tplTFVars)
	if err != nil {
		return err
	}

	return tmpl.Execute(f, vs)
}

func prepareCmd(dir string, environ []string, command string, args ...string) *exec.Cmd {
	args = append([]string{command}, args...)

	cmd := exec.Command(binaryTerraform, args...)
	cmd.Dir = dir
	cmd.Env = append(
		environ,
		"TF_LOG=TRACE",
		fmt.Sprintf("TF_LOG_PATH=/tmp/%s.log", command),
	)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	return cmd
}

func tfVar(k, v string) string {
	return fmt.Sprintf(fmtTFVar, k, v)
}
