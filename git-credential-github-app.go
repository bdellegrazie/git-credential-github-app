package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v63/github"
)

var version = "v0.0.1"

type CredHelperArgs struct {
	AppId          int64
	GithubApi      string
	InstallationId int64
	Organization   string
	Owner          string
	PrivateKeyFile string
	Repo           string
	User           string
	Username       string
}

func GithubAppResolveInstallationId(
	ctx context.Context,
	client *github.Client,
	installationId int64,
	organization string,
	user string,
	owner string, repo string) (int64, error) {

	if installationId != 0 {
		return installationId, nil
	}
	// Lookup by Repo + Owner (narrow)
	if len(repo) > 0 && len(owner) > 0 {
		appInstall, _, err := client.Apps.FindRepositoryInstallation(ctx, strings.ToLower(owner), strings.ToLower(repo))
		if err != nil {
			return 0, err
		} else {
			return appInstall.GetID(), nil
		}
	}

	// Lookup by Organisation
	if len(organization) > 0 {
		appInstall, _, err := client.Apps.FindOrganizationInstallation(ctx, strings.ToLower(organization))
		if err != nil {
			return 0, err
		} else {
			return appInstall.GetID(), nil
		}
	}

	// Lookup by User
	if len(user) > 0 {
		appInstall, _, err := client.Apps.FindUserInstallation(ctx, strings.ToLower(user))
		if err != nil {
			return 0, err
		} else {
			return appInstall.GetID(), nil
		}
	}

	return 0, fmt.Errorf("could not resolve Installation ID")
}

func PrintVersion(verbose bool) {
	fmt.Fprintln(os.Stderr, "version", version)
	if verbose {
		buildInfo, ok := debug.ReadBuildInfo()
		if !ok {
			log.Fatal("Cannot get build information from binary")
		}
		fmt.Fprintln(os.Stderr, buildInfo.String())
	}
}

func PrintUsage() {
	fmt.Fprintln(os.Stderr, "Git Credential Helper for Github Apps")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, os.Args[0], "-h|--help")
	fmt.Fprintln(os.Stderr, os.Args[0], "<-username USERNAME> <-appId ID> <-privateKeyFile PATH_TO_PRIVATE_KEY> [-installationID INSTALLATION_ID] [-organization ORGANIZATION] [-user USER] [<-owner OWNER> <-repo REPOSITORY>] [-githubApi GITHUB_API_URL] <get|store|erase>", os.Args[0])
	flag.PrintDefaults()
}

func Fatal(v ...any) {
	fmt.Println("quit=1")
	log.Fatal(v...)
}

func main() {
	args := CredHelperArgs{}
	versionFlagPtr := flag.Bool("version", false, "Get application version")
	verboseFlagPtr := flag.Bool("verbose", false, "Enable verbose version output")
	flag.Int64Var(&args.AppId, "appId", 0, "GitHub App AppId, mandatory")
	flag.StringVar(&args.GithubApi, "githubApi", "https://api.github.com", "GitHub API Base URL")
	flag.Int64Var(&args.InstallationId, "installationId", 0, "GitHub App Installation ID")
	flag.StringVar(&args.Organization, "organization", "", "GitHub App Organization")
	flag.StringVar(&args.Owner, "owner", "", "GitHub App Owner/Repo Installation (owner part)")
	flag.StringVar(&args.PrivateKeyFile, "privateKeyFile", "", "GitHub App Private Key File Path, mandatory")
	flag.StringVar(&args.Repo, "repo", "", "GitHub App Owner/Repo Installation (repo part)")
	flag.StringVar(&args.User, "user", "", "GitHub App User Installation")
	flag.StringVar(&args.Username, "username", "", "Git Credential Username, mandatory, recommend GitHub App Name")

	flag.Parse()

	if *versionFlagPtr {
		PrintVersion(*verboseFlagPtr)
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		PrintUsage()
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
	default:
		PrintUsage()
		os.Exit(1)
	}

	tr := http.DefaultTransport
	atr, err := ghinstallation.NewAppsTransportKeyFromFile(tr, args.AppId, args.PrivateKeyFile)
	if err != nil {
		log.Fatal("Either appId or privateKeyFile are invalid", err)
	}

	ctx := context.Background()
	client := github.NewClient(&http.Client{Transport: atr})
	installationId, err := GithubAppResolveInstallationId(
		ctx,
		client,
		args.InstallationId,
		args.Organization,
		args.User,
		args.Owner, args.Repo)
	if err != nil {
		log.Fatal("failed to get installation Id", err)
	}

	installationToken, _, err := client.Apps.CreateInstallationToken(ctx, installationId, nil)
	if err != nil {
		Fatal("Could not create Github App Installation Access Token", err)
	}

	fmt.Println("username=", args.Username)
	fmt.Println("password=", installationToken.GetToken())
	fmt.Println("password_expiry_utc", installationToken.GetExpiresAt().Unix())
}
