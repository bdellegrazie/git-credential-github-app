package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v63/github"
)

var version = "v0.0.1"

type CredHelperArgs struct {
	AppId          int64
	InstallationId int64
	PrivateKeyFile string
	Username       string
}

func printVersion(verbose bool) {
	fmt.Fprintln(os.Stderr, "version", version)
	if verbose {
		buildInfo, ok := debug.ReadBuildInfo()
		if !ok {
			log.Fatal("Cannot get build information from binary")
		}
		fmt.Fprintln(os.Stderr, buildInfo.String())
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Git Credential Helper for Github Apps")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, os.Args[0], "-h|--help")
	fmt.Fprintln(os.Stderr, os.Args[0], "-v|--version [--verbose]")
	fmt.Fprintln(os.Stderr, os.Args[0], "<-username USERNAME> <-appId ID> <-privateKeyFile PATH_TO_PRIVATE_KEY> <-installationID INSTALLATION_ID> <get|store|erase>")
	fmt.Fprintln(os.Stderr, os.Args[0], "<-username USERNAME> <-appId ID> <-privateKeyFile PATH_TO_PRIVATE_KEY> generate")
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
}

func fatal(v ...any) {
	fmt.Println("quit=1")
	log.Fatal(v...)
}

func credentialGetOutput(w io.Writer, username string, token *github.InstallationToken) error {
	_, err := fmt.Fprintf(w, "username=%s\npassword=%s\npassword_expiry_utc=%d\n",
		username,
		token.GetToken(),
		token.GetExpiresAt().Unix())
	return err
}

func generateGitConfig(w io.Writer, installations []*github.Installation, args *CredHelperArgs) {
	for _, installation := range installations {
		fmt.Fprintf(w, "[credential \"%s\"]\n\tuseHttpPath = true\n\thelper = \"github-app -username %s -appId %d -privateKeyFile %s -installationId %d\n",
			installation.GetAccount().GetHTMLURL(), args.Username, args.AppId, args.PrivateKeyFile, installation.GetID())
	}
	fmt.Fprintln(w, "[credential \"https://github.com\"]\n\thelper = \"cache --timeout=43200\"")
	fmt.Fprintln(w, "[url \"https://github.com\"]\n\tinsteadOf = ssh://git@github.com")
}

func newGithubAppClient(tr http.RoundTripper, appId int64, privateKeyFile string) (*github.Client, error) {
	atr, err := ghinstallation.NewAppsTransportKeyFromFile(tr, appId, privateKeyFile)
	if err != nil {
		return nil, err
	}
	return github.NewClient(&http.Client{Transport: atr}), nil
}

func doGet(w io.Writer, args *CredHelperArgs) {
	client, err := newGithubAppClient(http.DefaultTransport, args.AppId, args.PrivateKeyFile)
	if err != nil {
		log.Fatal("Error creating client: ", err)
	}
	ctx := context.Background()

	installationToken, _, err := client.Apps.CreateInstallationToken(ctx, args.InstallationId, nil)
	if err != nil {
		fatal("Could not create Github App Installation Access Token: ", err)
	}
	credentialGetOutput(w, args.Username, installationToken)
}

func doGenerate(w io.Writer, args *CredHelperArgs) {
	client, err := newGithubAppClient(http.DefaultTransport, args.AppId, args.PrivateKeyFile)
	if err != nil {
		log.Fatal("Error creating client: ", err)
	}
	ctx := context.Background()

	var allInstallations []*github.Installation
	opt := github.ListOptions{PerPage: 10}
	for {
		installations, resp, err := client.Apps.ListInstallations(ctx, &opt)
		if err != nil {
			log.Fatal("Error retrieving installations: ", err)
		}
		allInstallations = append(allInstallations, installations...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	generateGitConfig(w, allInstallations, args)
}

func main() {
	args := CredHelperArgs{}
	versionFlagPtr := flag.Bool("version", false, "Get application version")
	verboseFlagPtr := flag.Bool("verbose", false, "Enable verbose version output")
	flag.Int64Var(&args.AppId, "appId", 0, "GitHub App AppId, mandatory")
	flag.Int64Var(&args.InstallationId, "installationId", 0, "GitHub App Installation ID")
	flag.StringVar(&args.PrivateKeyFile, "privateKeyFile", "", "GitHub App Private Key File Path, mandatory")
	flag.StringVar(&args.Username, "username", "", "Git Credential Username, mandatory, recommend GitHub App Name")

	flag.Parse()

	if *versionFlagPtr {
		printVersion(*verboseFlagPtr)
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		printUsage()
		os.Exit(1)
	}

	if args.AppId == 0 {
		log.Fatal("appId is mandatory")
	}

	if len(args.PrivateKeyFile) == 0 {
		log.Fatal("Path to Private Key file is mandatory")
	}

	if len(args.Username) == 0 {
		log.Fatal("username is mandatory")
	}

	switch operation := flag.Arg(0); operation {
	case "erase":
		os.Exit(0)
	case "store":
		os.Exit(0)
	case "get":
		if args.InstallationId == 0 {
			log.Fatal("installationId is mandatory for get operation")
		}
		doGet(os.Stdout, &args)
	case "generate":
		doGenerate(os.Stdout, &args)
	default:
		printUsage()
		os.Exit(1)
	}
}
