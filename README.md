# Cloud Functions Bucket File Editor

Browser based text editor for files in gcloud storage buckets.

- Edit files in storage buckets from anywhere even a Chromebook
- No need for `gsutil` or `gcsfuse`, specialized text editors
- No need for VMs, Containers, Cloud Run, Cloud Shells/IDE
- No frills, just enough for html textarea based file editor
- Doesn't use IAM/Google accounts, implements own authentication

## Deployment 

### Gconsole UI

Cloud Functions -> Create Function

* Authentication: **Allow unauthenticated invocations**
* Runtime: **Go**
* Entry point: **CFEdit**
* Inline Editor: paste code of `cfedit.go`
* Change project name and restrict to specific bucket
* Add user accounts for basic auth

### Gcloud CLI

Edit `cfedit.go`

* Change project name and restrict to specific bucket
* Add user accounts see user authentication section

Run:

```shell
$ gcloud functions deploy cfedit \
    --entry-point=CFEdit \
    --runtime=go116 \
    --trigger-http \
    --allow-unauthenticated \
    --source=/path/to/cfedit
```

Delete:

```shell
$ gcloud functions delete cfedit
```

## User Authentication

Cfedit uses HTTP Basic Auth with hardcoded user database. I specifically wanted
to avoid forcing users to have Google accounts,
[IAM + OAuth2](https://cloud.google.com/functions/docs/securing)
complexities and JavaScript based
[Google Sign-in](https://developers.google.com/identity/sign-in/web/sign-in)
buttons.

### User Database

The user database is stored in a struct inside `var()` section of cfedit.go.

```go
users = []struct{ login, salt, hash string }{
  {login: "admin", salt: "abc", hash: "hash of salt+pass"},
  {login: "editor", salt: "def", hash: "hash of salt+pass"},
}
```

### Adding a user

To add a user simply add a new line with `{login: "foo", salt: "bar", hash: "..."},`.

To generate the password hash:

```sh
$ echo -n "SaltMyPassword" | shasum -a 256 | cut -f 1 -d" "
```

The salt is simply a unique random string you pick. The only requirement is that has to be different for each user.
For example if login is `foo`, salt is `bar` and password is `baz`:

```sh
$ echo -n "barbaz" | shasum -a 256 | cut -f 1 -d" "
c8f8b724728a6d6684106e5e64e94ce811c9965d19dd44dd073cf86cf43bc238
```

```go
users = []struct{ login, salt, hash string } {
	{login: "foo", salt: "bar", hash: "c8f8b724728a6d6684106e5e64e94ce811c9965d19dd44dd073cf86cf43bc238"},
}
```

### No authentication

To disable authentication entirely remove/comment out all users.

## Local Development

Create a Service Account under IAM. Create and download a json key file.

Run locally:

```shell
$ GOOGLE_APPLICATION_CREDENTIALS=~/project-1234345.json go run local/main.go
```

Reference:

https://cloud.google.com/functions/docs/running/function-frameworks#go
https://github.com/GoogleCloudPlatform/functions-framework-go


## Roadmap
- rate limit on auth attempts
- subfolders/prefixes
  https://cloud.google.com/storage/docs/listing-objects#storage-list-objects-go
- preconditions and race conditions 
  https://cloud.google.com/storage/docs/generations-preconditions#examples_of_race_conditions
- possibly use a better editor such as editarea, codemirror, monaco, codejar
  https://en.wikipedia.org/wiki/Comparison_of_JavaScript-based_source_code_editors
  or markdown editor like TinyMCE, CKEditor

## Legal

```text
Copyright (c) 2021 Google LLC
This is not an officially supported Google product
```
