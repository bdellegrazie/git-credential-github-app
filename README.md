# Git Credential Helper for Github Apps

The GitHub App Credential Helper is a [Git credential helper][1] that provides access tokens from a GitHub App installed
as an organization, user or repository Access tokens have a lifetime of 1 hour.

When combined with other Credential Helpers like [git-credential-cache][6] it can remove the need for a Personal Access Token (PAT).

The helper can generate the recommended Git configuration for ease of use.

# Installation

The GitHub App Credential Helper is a [Git Credential Custom Helper][2].

Copy the binary to be on the user's `$PATH` or `$GIT_EXEC_PATH` for system installation. Use the short name (`github-app`) in Git's configuration

# Configuration

## Overview

Setup is entirely within [Git's Configuration][5], using [Git Credential Helper Options][3] and this helper can generate the recommended Git configuration.

Each GitHub App Installation must be configured explicitly to ensure Git matches the correct [Credential Context][4] to the Installation,
including the `useHttpPath = true` to ensure the credential context is narrow.

*IMPORTANT*:

* Define narrow credential contexts first, any `https://github.com` credential context should be defined last
* The config setting `credential.<context>.helper` is multi-valued and helpers are invoked in the order specified

The Github App Credential helper is typically configured with:

1. [git-credential-cache][6] helper at a global or https://github.com context to avoid unnecessary GitHub API calls
2. SSH -> HTTPS redirect to ensure SSH requests are redirected to HTTPS so the credential helper is used, at a top level domain or globally.

This way, Git is doing most of the heavy lifting, ensuring the most narrow credential context is used and credentials are cached so the GitHub API is not abused.

## Generating the Configuration

The helper can generate the recommended configuration on stdout via the `generate` command using the supplied credentials.
It should be copied to the appropriate Git configuration file. Typically `${HOME}/.gitconfig` or `/etc/gitconfig`

The generated configuration has the following characterstics:

* Organization / User installations are listed in the order supplied by the API. (`useHttpPath=true` is set on each)
* Github configuration is listed last and only contains a cache
* SSH -> HTTPS URL redirect is over the entire GitHub domain

Adjust the configuration to suit your circumstances.

For example in `${HOME}/.gitconfig`:
```
[credential "https://github.com/exampleOrg"]
    helper = 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -installationId 123456'
    useHttpPath = true

[credential "https://github.com/exampleUser"]
    helper = 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -installationId 123457'
    useHttpPath = true

[credential "https://github.com/exampleOrg2"]
    helper = 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -installationId 123458'
    useHttpPath = true

[credential "https://github.com"]
    helper = 'cache --timeout=43200'

[url "https://github.com"]
    insteadOf = ssh://git@github.com
```

## Credential Cache Helper

The [git-credential-cache][6] is _usually_ defined as an top level (empty) credential context helper. It is possible to have more than one cache at
different credential context levels. GitHub App Installation access tokens last for 1 hr (3600s) so the timeout should be _at least_ 2 hr (7200s).
The value shown below 12 hours.

Configuration will be similar to this:
```
git config --global --add credential.helper 'cache --timeout=43200'
```

Which looks like the following in `${HOME}/.gitconfig`:
```
[credential]
    helper = 'cache --timeout=43200'
```
This is sometimes in the _system_ level config instead (`/etc/gitconfig`)

It is possible to define a helper just for the `https://github.com` context like this:
```
git config --global --add credential."https://github.com".helper 'cache --timeout=43200'
```

Which looks like the following in `${HOME}/.gitconfig`:
```
[credential "https://github.com"]
    helper = 'cache --timeout=43200'
```

## Redirect SSH to HTTPS

To ensure that all GitHub access is via HTTPS so the credential helper is invoked use:
```
git config --global --add url."https://github.com".insteadOf "ssh://git@github.com"
```

Which looks like the following in `${HOME}/.gitconfig`:
```
[url "https://github.com"]
    insteadOf = "ssh://git@github.com"
```

## GitHub App Helper

The help reports:
```
Git Credential Helper for Github Apps
Usage:
./git-credential-github-app -h|--help
./git-credential-github-app -v|--version
./git-credential-github-app <-username USERNAME> <-appId ID> <-privateKeyFile PATH_TO_PRIVATE_KEY> <-installationID INSTALLATION_ID> <get|store|erase>
./git-credential-github-app <-username USERNAME> <-appId ID> <-privateKeyFile PATH_TO_PRIVATE_KEY> generate
Options:
  -appId int
    	GitHub App AppId, mandatory
  -installationId int
    	GitHub App Installation ID
  -privateKeyFile string
    	GitHub App Private Key File Path, mandatory
  -username string
    	Git Credential Username, mandatory, recommend GitHub App Name
  -version
    	Get application version
```

<!-- References -->

[1]: https://git-scm.com/docs/gitcredentials
[2]: https://git-scm.com/docs/gitcredentials#_custom_helpers
[3]: https://git-scm.com/docs/gitcredentials#_configuration_options
[4]: https://git-scm.com/docs/gitcredentials#_credential_contexts
[5]: https://git-scm.com/docs/git-config
[6]: https://git-scm.com/docs/git-credential-cache
