package main

import (
	"crypto/rsa"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type githubApp struct {
	url            string
	clientId       string
	privateKey     *rsa.PrivateKey
	installationId uint64
	organisation   string
	slug           string
}

func PrivateKeyFromPath(keyFile string) (*rsa.PrivateKey, error) {
	f, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPrivateKeyFromPEM(f)
}

func GithubAppJwt(app *githubApp, issuedAt time.Time) (string, error) {
	claims := &jwt.RegisteredClaims{
		Issuer:    app.clientId,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(issuedAt.Add(time.Minute * 10)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(app.privateKey)
}

func GithubAppCommonHeaders(req *http.Request, token string) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
}

type GithubAppOrgGetInstallationResponseBody struct {
	Id uint64 `json:"id"`
}

func GithubAppOrgGetInstallation(app *githubApp, token string, client *http.Client) (uint64, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/orgs/%s/installation", app.url, app.organisation), nil)
	if err != nil {
		return 0, err
	}
	GithubAppCommonHeaders(req, token)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var dat GithubAppOrgGetInstallationResponseBody
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&dat); err != nil {
		return 0, err
	}
	return dat.Id, nil
}

type GithubAppSlugGetInstallationResponseBody struct {
	Id uint64 `json:"id"`
}

func GithubAppSlugGetInstallation(app *githubApp, token string, client *http.Client) (uint64, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/repos/%s/installation", app.url, app.slug), nil)
	if err != nil {
		return 0, err
	}
	GithubAppCommonHeaders(req, token)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var dat GithubAppSlugGetInstallationResponseBody
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&dat); err != nil {
		return 0, err
	}
	return dat.Id, nil
}

type GithubAppGetInstallationAccessTokenResponseBody struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"` // "expires_at": "2016-07-11T22:14:10Z"
}

type GithubAppInstallationAccessToken struct {
	Token     string
	ExpiresAt time.Time
}

func GithubAppGetInstallationAccessToken(app *githubApp, token string, client *http.Client) (*GithubAppInstallationAccessToken, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/app/installations/%d/access_tokens", app.url, app.installationId), nil)
	if err != nil {
		return nil, err
	}
	GithubAppCommonHeaders(req, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var dat GithubAppGetInstallationAccessTokenResponseBody
	if err = decoder.Decode(&dat); err != nil {
		return nil, err
	}

	expiresAt, err := time.Parse(time.RFC3339, dat.ExpiresAt)
	if err != nil {
		return nil, err
	}

	return &GithubAppInstallationAccessToken{Token: dat.Token, ExpiresAt: expiresAt}, nil
}

func PrintUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, os.Args[0], "-h|--help")
	fmt.Fprintln(os.Stderr, os.Args[0], "<-username USERNAME> <-clientId ID> <-privateKey PATH_TO_PRIVATE_KEY> <-installationId INSTALLATION_ID | -organisation ORGANISATION | -slug OWNER/REPO> [-url GITHUB_API_URL] <get|store|erase>", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var app githubApp
	flag.StringVar(&app.clientId, "clientId", "", "GitHub App ClientId (preferred) or AppId, mandatory")
	privateKeyFile := flag.String("privateKey", "", "GitHub App Private Key File, mandatory")
	flag.Uint64Var(&app.installationId, "installationId", 0, "GitHub App Installation Id, recommended")
	flag.StringVar(&app.organisation, "organisation", "", "GitHub App Organisation, required if InstallationId not supplied and installation is an org")
	flag.StringVar(&app.slug, "slug", "", "GitHub App Owner/Repo slug, required if InstallationId not supplied and installation is a specific repo")
	flag.StringVar(&app.url, "url", "https://api.github.com", "GitHub API Base URL")
	usernamePtr := flag.String("username", "", "GitHub Username, mandatory, recommend GitHub App Name")

	flag.Parse()

	if flag.NArg() != 1 {
		PrintUsage()
		os.Exit(1)
	}

	if len(app.clientId) == 0 {
		panic("ClientId is mandatory")
	}

	if len(*privateKeyFile) == 0 {
		panic("Path to Private Key file is mandatory")
	}

	if app.installationId == 0 {
		// Get InstallationId
		if len(app.organisation) > 0 {
			app.organisation = strings.ToLower(app.organisation)
		} else if len(app.slug) > 0 {
			app.slug = strings.ToLower(app.slug)
		} else {
			panic("When InstallationId is not supplied, Organisation or Slug is required")
		}
	}

	if len(*usernamePtr) == 0 {
		panic("Username is mandatory")
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

	var err error
	app.privateKey, err = PrivateKeyFromPath(*privateKeyFile)
	if err != nil {
		fmt.Println("quit=1")
		panic("Private Key not parseable")
	}

	token, err := GithubAppJwt(&app, time.Now())
	if err != nil {
		fmt.Println("quit=1")
		panic("Could not generate JWT token")
	}

	if app.installationId == 0 {
		if len(app.organisation) > 0 {
			app.installationId, err = GithubAppOrgGetInstallation(&app, token, http.DefaultClient)
			if err != nil {
				fmt.Println("quit=1")
				panic("Could not get installation ID from Organisation")
			}
		} else if len(app.slug) > 0 {
			app.installationId, err = GithubAppSlugGetInstallation(&app, token, http.DefaultClient)
			if err != nil {
				fmt.Println("quit=1")
				panic("Could not get installation ID from Slug")
			}
		} else {
			panic("Neither Organisation nor Slug available")
		}
	}

	accessTokenPtr, err := GithubAppGetInstallationAccessToken(&app, token, http.DefaultClient)
	if err != nil {
		fmt.Println("quit=1")
		panic("Could not get Access Token for Installation")
	}

	fmt.Println("username=", *usernamePtr)
	fmt.Println("password=", accessTokenPtr.Token)
	fmt.Println("password_expiry_utc", accessTokenPtr.ExpiresAt.Unix())
}
