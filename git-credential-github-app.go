package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v72/github"
)

var version = "v0.0.1"

type CredHelperArgs struct {
	AppId          int64
	InstallationId int64
	Organization   string
	PrivateKeyFile string
	Username       string
	Domain         string
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
	fmt.Fprintln(os.Stderr, os.Args[0], "-v|--version")
	fmt.Fprintln(os.Stderr, os.Args[0], "<-username USERNAME> <-appId ID> <-privateKeyFile PATH_TO_PRIVATE_KEY> <[-installationId INSTALLATION_ID] | [-organization ORGANIZATION]> [-domain GHE_DOMAIN] <get|store|erase>")
	fmt.Fprintln(os.Stderr, os.Args[0], "<-username USERNAME> <-appId ID> <-privateKeyFile PATH_TO_PRIVATE_KEY> [-domain GHE_DOMAIN] generate")
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
	domain := "github.com"
	if args.Domain != "" {
		domain = args.Domain
	}

	for _, installation := range installations {
		fmt.Fprintf(w, "[credential \"%s\"]\n\tuseHttpPath = true\n\thelper = \"github-app -username %s -appId %d -privateKeyFile %s -installationId %d\"\n",
			installation.GetAccount().GetHTMLURL(), args.Username, args.AppId, args.PrivateKeyFile, installation.GetID())
	}
	fmt.Fprintf(w, "[credential \"https://%s\"]\n\thelper = \"cache --timeout=43200\"\n", domain)
	fmt.Fprintf(w, "[url \"https://%s\"]\n\tinsteadOf = ssh://git@github.com\n", domain)
}

func newGithubAppClient(tr http.RoundTripper, appId int64, privateKeyFile, domain string) (*github.Client, error) {
	atr, err := ghinstallation.NewAppsTransportKeyFromFile(tr, appId, privateKeyFile)
	if err != nil {
		return nil, err
	}

	client := github.NewClient(&http.Client{Transport: atr})
	if domain == "" {
		return client, nil
	}

	baseUrl := "https://" + domain
	atr.BaseURL = baseUrl + "/api/v3"
	// Enterprise URLs need a terminating slash
	return client.WithEnterpriseURLs(baseUrl+"/api/v3/", baseUrl+"/api/uploads/")
}

func doGet(w io.Writer, args *CredHelperArgs) {
	client, err := newGithubAppClient(http.DefaultTransport, args.AppId, args.PrivateKeyFile, args.Domain)
	if err != nil {
		log.Fatal("Error creating client: ", err)
	}
	ctx := context.Background()

	if args.InstallationId == 0 {
		installation, _, err := client.Apps.FindOrganizationInstallation(ctx, args.Organization)
		if err != nil {
			fatal("Could not get InstallationId from Organization: ", err)
		}
		args.InstallationId = *installation.ID
	}

	installationToken, _, err := client.Apps.CreateInstallationToken(ctx, args.InstallationId, nil)
	if err != nil {
		fatal("Could not create Github App Installation Access Token: ", err)
	}
	credentialGetOutput(w, args.Username, installationToken)
}

func doGenerate(w io.Writer, args *CredHelperArgs) {
	client, err := newGithubAppClient(http.DefaultTransport, args.AppId, args.PrivateKeyFile, args.Domain)
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
	flag.Int64Var(&args.AppId, "appId", 0, "GitHub App AppId, mandatory")
	flag.Int64Var(&args.InstallationId, "installationId", 0, "GitHub App Installation ID")
	flag.StringVar(&args.Organization, "organization", "", "GitHub App Organization, optional")
	flag.StringVar(&args.PrivateKeyFile, "privateKeyFile", "", "GitHub App Private Key File Path, mandatory")
	flag.StringVar(&args.Username, "username", "", "Git Credential Username, mandatory, recommend GitHub App Name")
	flag.StringVar(&args.Domain, "domain", "", "GitHub Enterprise domain, optional")

	flag.Parse()

	if *versionFlagPtr {
		printVersion(true)
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

	// Resolve private key file path or generated configurations may not work correctly
	var err error
	if args.PrivateKeyFile, err = filepath.Abs(args.PrivateKeyFile); err != nil {
		log.Fatal("Path to Private Key could not be made absolute with error: ", err)
	}

	switch operation := flag.Arg(0); operation {
	case "erase":
		os.Exit(0)
	case "store":
		os.Exit(0)
	case "get":
		if args.InstallationId == 0 && len(args.Organization) == 0 {
			log.Fatal("installationId or Organization is mandatory for get operation")
		}
		doGet(os.Stdout, &args)
	case "generate":
		doGenerate(os.Stdout, &args)
	default:
		printUsage()
		os.Exit(1)
	}
}
