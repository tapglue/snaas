package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/tapglue/snaas/platform/generate"
)

const (
	argDestroy = "-destroy"
	argOut     = "-out"
	argState   = "-state"
	argVarFile = "-var-file"

	binaryTerraform = "terraform"

	cmdSetup    = "setup"
	cmdTeardown = "teardown"
	cmdUpdate   = "update"

	defaultStatesPath   = "infrastructure/terraform/states"
	defaultTemplatePath = "infrastructure/terraform/template"
	defaultTmpPath      = "/tmp"

	fmtNamespace = "%s-%s"
	fmtPlan      = "%s/%s.plan"
	fmtStateFile = "%s.tfstate"
	fmtVarsFile  = "%s.tfvars"
	fmtTFVar     = "TF_VAR_%s=%s"

	tfCmdApply   = "apply"
	tfCmdDestroy = "destroy"
	tfCmdPlan    = "plan"

	tplTFVars = `
key = {
	access = "{{.KeyAccess}}"
}
pg_password = "{{.PGPassword}}"
`

	varAccount = "account"
	varEnv     = "env"
	varRegion  = "region"
)

func main() {
	var (
		env          = flag.String("env", "", "Environment used for isolation.")
		region       = flag.String("region", "", "AWS region to deploy to.")
		sshPath      = flag.String("ssh.path", "", "Location of SSH public key to use for setup.")
		statesPath   = flag.String("states.path", defaultStatesPath, "Location to store env states.")
		templatePath = flag.String("template.path", defaultTemplatePath, "Location of the infrastructure template.")
		tmpPath      = flag.String("tmp.path", defaultTmpPath, "Location for temporary output like plans.")
		varsPath     = flag.String("vars.path", "", "Location of vars file.")
	)
	flag.Parse()

	log.SetFlags(log.Lshortfile)

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

	account, err := awsAcoount(*region)
	if err != nil {
		log.Fatalf("AWS account fetch failed: %s", err)
	}

	var (
		namespace = fmt.Sprintf(fmtNamespace, *env, *region)
		planFile  = fmt.Sprintf(fmtPlan, *tmpPath, namespace)
		statePath = filepath.Join(*statesPath, namespace)
		stateFile = filepath.Join(statePath, fmt.Sprintf(fmtStateFile, namespace))
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

	switch flag.Args()[0] {
	case cmdSetup:
		if _, err := os.Stat(*sshPath); err != nil {
			log.Fatalf("couldn't locate ssh public key: %s", err)
		}

		keyRaw, err := ioutil.ReadFile(*sshPath)
		if err != nil {
			log.Fatalf("ssh key read failed: %s", err)
		}

		if _, err := os.Stat(stateFile); err == nil {
			log.Fatalf("state file already exists: %s", stateFile)
		}

		if err := os.MkdirAll(statePath, os.ModePerm); err != nil {
			log.Fatalf("state dir creation failed: %s", err)
		}

		if _, err := os.Stat(varFile); err != nil {
			if !os.IsNotExist(err) {
				log.Fatal(err)
			}

			err := generateVarFile(varFile, string(keyRaw), generate.RandomString(32))
			if err != nil {
				log.Fatalf("var file create failed: %s", err)
			}
		}

		args := []string{
			argOut, planFile,
			argState, stateFile,
			argVarFile, varFile,
			*templatePath,
		}

		if err := prepareCmd(environ, tfCmdPlan, args...).Run(); err != nil {
			os.Exit(1)
		}

		fmt.Println("Want to apply the plan? (type 'yes')")
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

		if err := prepareCmd(environ, tfCmdApply, args...).Run(); err != nil {
			os.Exit(1)
		}
	case cmdTeardown:
		args := []string{
			argDestroy,
			argOut, planFile,
			argState, stateFile,
			argVarFile, varFile,
			*templatePath,
		}

		if err := prepareCmd(environ, tfCmdPlan, args...).Run(); err != nil {
			os.Exit(1)
		}

		args = []string{
			argState, stateFile,
			argVarFile, varFile,
			*templatePath,
		}

		if err := prepareCmd(environ, tfCmdDestroy, args...).Run(); err != nil {
			os.Exit(1)
		}
	case cmdUpdate:
		if _, err := os.Stat(stateFile); err != nil {
			log.Fatalf("couldn't locate state file: %s", err)
		}

		args := []string{
			argOut, planFile,
			argState, stateFile,
			argVarFile, varFile,
			*templatePath,
		}

		if err := prepareCmd(environ, tfCmdPlan, args...).Run(); err != nil {
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

		if err := prepareCmd(environ, tfCmdApply, args...).Run(); err != nil {
			os.Exit(1)
		}
	default:
		log.Fatalf("unknown command '%s'", flag.Args()[0])
	}
}

func awsAcoount(region string) (string, error) {
	var (
		providers = []credentials.Provider{
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{},
		}
		awsSession = session.New(&aws.Config{
			Credentials: credentials.NewChainCredentials(providers),
			Region:      aws.String(region),
		})
		stsService = sts.New(awsSession)
	)

	res, err := stsService.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return *res.Account, nil
}

func generateVarFile(path, keyAccess, pgPassword string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	tmpl, err := template.New("vars").Parse(tplTFVars)
	if err != nil {
		return err
	}

	tmpl.Execute(f, struct {
		KeyAccess  string
		PGPassword string
	}{
		KeyAccess:  strings.Trim(keyAccess, "\n"),
		PGPassword: pgPassword,
	})

	return nil
}

func prepareCmd(environ []string, command string, args ...string) *exec.Cmd {
	args = append([]string{command}, args...)

	cmd := exec.Command(binaryTerraform, args...)
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
