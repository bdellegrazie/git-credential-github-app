# Git Credential Helper for Github Apps

The GitHub App Credential Helper is a [Git credential helper][1] that provides access tokens from a GitHub App installed
as an organization, user or repository Access tokens have a lifetime of 1 hour.

When combined with other Credential Helpers like [git-credential-cache][6] it can remove the need for a Personal Access Token (PAT).

# Installation

The GitHub App Credential Helper is a [Git Credential Custom Helper][2].

Copy the binary to be on the user's `$PATH` or `$GIT_EXEC_PATH` for system installation. Use the short name (`github-app`) in Git's configuration

# Configuration

## Overview

Setup is entirely within [Git's Configuration][5], using [Git Credential Helper Options][3].

Each GitHub App Installation must be configured explicitly to ensure Git matches the correct [Credential Context][4] to the Installation,
including the `useHttpPath = true` to ensure the credential context is narrow.

One Installation can be designated the _default_ by using the top level domain as the credential context to allow access tokens for public repositories.
In this context, `useHttpPath` be set to `false` or left unset (`false` is the default).

The Github App Credential helper is typically configured with:

1. [git-credential-cache][6] helper at a global or top-level domain to avoid unnecessary GitHub API calls
2. SSH -> HTTPS redirect to ensure SSH requests are redirected to HTTPS so the credential helper is used, at a top level domain or globally.

This way, Git is doing most of the heavy lifting, ensuring the most narrow credential context is used and credentials are cached so the GitHub API is not abused.

## Credential Cache Helper

The [git-credential-cache][6] is _usually_ defined as an top level context helper. It is possible to have more than one cache at different credential context levels.
GitHub App Installation access tokens last for 1 hr (3600s) so the timeout should be _at least_ 2 hr (7200s). The value shown is 12 hours.

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
    helper = 'github-app -username myAppName -appId 123 -installationId 456 -privateKeyFile /path/to/private.pem'
```
*NOTE*:

In Git configuration, `credential.<context>.helper` is multi-valued and helpers are invoked in the order specified.
So when `cache` and `github-app` are used in the same context, `cache` should be first and `github-app` after.

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
Usage of git-credential-github-app:
  -appId int
    	GitHub App AppId, mandatory
  -githubApi string
    	GitHub API Base URL (default "https://api.github.com")
  -installationId int
    	GitHub App Installation ID
  -organization string
    	GitHub App Organization
  -owner string
    	GitHub App Owner/Repo Installation (owner part)
  -privateKeyFile string
    	GitHub App Private Key File Path, mandatory
  -repo string
    	GitHub App Owner/Repo Installation (repo part)
  -user string
    	GitHub App User Installation
  -username string
    	Git Credential Username, mandatory, recommend GitHub App Name
```

For any specific installation, the `installationId` can be supplied directly or looked up by one of:
* `organization`
* `user`
* `owner` and `repo`

## `https://github.com` Credential Context

Usage, as a [Git Custom Credential Helper][2] for GitHub domain credential context (i.e. `https://github.com`), is like this:

```
git config --global --add credential."https://github.com".helper 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -installationId 456'
```

Which looks like this in `${HOME}/.gitconfig`:

```
[credential "https://github.com"]
    helper = 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -installationId 456'
```

`helper` is multi-valued config element, if the `cache` helper is used at the `https://github.com` context instead of as a global default, it should come _before_ `github-app`

## Org, User, or Repo Context

The credential context can be narrowed to an _organization_, a _user_ or an _owner/repo_ context.

Note the `useHttpPath = true` in all situations.

`installationId` can be supplied directly or as in the examples below looked up from GitHub on demand:

```
# Organization App Install
git config --global --add credential."https://github.com/exampleOrg/".helper 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -organization exampleOrg'
git config --global --add credential."https://github.com/exampleOrg/".useHttpPath true

# User App Install
git config --global --add credential."https://github.com/exampleUser/".helper 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -user exampleUser'
git config --global --add credential."https://github.com/exampleUser/".useHttpPath true

# Repo App Install
git config --global --add credential."https://github.com/exampleOwner/exampleRepo.git".helper 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -owner exampleOwner -repo exampleRepo'
git config --global --add credential."https://github.com/exampleOwner/exampleRepo.git".useHttpPath true
```

Which looks like this in `${HOME}/.gitconfig`:
```
[credential "https://github.com/exampleOrg/"]
    helper = 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -organization exampleOrg'
    useHttpPath = true

[credential "https://github.com/exampleUser/"]
    helper = 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -user exampleUser'
    useHttpPath = true

[credential "https://github.com/exampleOwner/exampleRepo.git"]
    helper = 'github-app -username myAppName -appId 123 -privateKeyFile /path/to/private.pem -owner exampleOwner -repo exampleRepo'
    useHttpPath = true
```

## Multiple App Installations

The helper supports mutliple App installations but because Git is configured statically it requires:

1. The set of supported installations (organizations, users or repositories) be known in advance
2. One of those installations should be designated the default, it is used as credential helper for the `https://github.com` context
3. Each installation is listed in Git configuration with explicit credential context, as shown above.

<!-- References -->

[1]: https://git-scm.com/docs/gitcredentials
[2]: https://git-scm.com/docs/gitcredentials#_custom_helpers
[3]: https://git-scm.com/docs/gitcredentials#_configuration_options
[4]: https://git-scm.com/docs/gitcredentials#_credential_contexts
[5]: https://git-scm.com/docs/git-config
[6]: https://git-scm.com/docs/git-credential-cache
