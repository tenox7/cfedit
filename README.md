# Cloud Functions Bucket File Editor

Browser based text editor for files in gcloud storage buckets.

- Edit files in storage buckets from anywhere even a Chromebook
- No need for `gsutil` or `gcsfuse`, specialized text editors
- No need for VMs, Containers, Cloud Run, Cloud Shells/IDE
- No frills, just enough for html textarea based file editor
- No need for IAM accounts, implements own authentication

## Deployment 

### Gconsole UI

Cloud Functions -> Create Function

* Authentication: **Allow unauthenticated invocations**
* Runtime: **Go**
* Entry point: **Main**
* Inline Editor: paste code of `cfedit.go`
* Change project name and restrict to specific bucket
* Add user accounts for basic auth

### Gcloud CLI

Edit `cfedit.go`

* Change project name and restrict to specific bucket
* Add user accounts for basic auth

Run:

```shell
$ gcloud functions deploy cfedit \
    --entry-point=Main \
    --runtime=go116 \
    --trigger-http \
    --allow-unauthenticated \
    --source=/path/to/cfedit
```

Delete:

```shell
$ gcloud functions delete cfedit
```


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
- iam auth?
  https://cloud.google.com/functions/docs/securing
  https://developers.google.com/identity/sign-in/web/sign-in
- subfolders/prefixes
  https://cloud.google.com/storage/docs/listing-objects#storage-list-objects-go
- preconditions and race conditions 
  https://cloud.google.com/storage/docs/generations-preconditions#examples_of_race_conditions
- rate limit on auth
- possibly use a better editor such as editarea, codemirror, monaco, codejar
  https://en.wikipedia.org/wiki/Comparison_of_JavaScript-based_source_code_editors
  or markdown editor like TinyMCE, CKEditor

## Legal

```text
Copyright (c) 2021 Google LLC
This is not an officially supported Google product
```
