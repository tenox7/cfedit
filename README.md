# Cloud Functions Bucket File Editor

Browser based text editor for files in gcloud storage buckets.

## Why

- Edit files in storage buckets from anywhere even on a Chromebook
- No need for `gsutil` or `gcsfuse`, desktop or cloud editor apps
- Cloud Functions are super simple, easy and cheap
- No no need for VMs, Containers, Cloud Run, Cloud Shells/IDE
- Dead simple implementing just enough for html textarea file editor
- Implements own authentication, no need for IAM accounts

## Deployment 

### Gconsole UI

Cloud Functions -> Create Function

* Authentication: Allow unauthenticated invocations
* Runtime: Go
* Entry point: Main
* Inline Editor: paste code of cfedit.go

### Gcloud CLI

```shell
$ gcloud functions deploy \
    cfedit \
    --entry-point Main \
    --runtime=go116 \
    --trigger-http \
    --allow-unauthenticated \
    --source=/path/to/cfedit
```

```shell
$ gcloud functions delete cfedit
```


## Local Development

Create a Service Account under IAM. Create and download a json key file.

```shell
$ GOOGLE_APPLICATION_CREDENTIALS=~/project-1234345.json go run local/main.go
```

Reference:

https://cloud.google.com/functions/docs/running/function-frameworks#go
https://github.com/GoogleCloudPlatform/functions-framework-go


## Roadmap
- auth via secrets manager?  
  https://cloud.google.com/secret-manager/docs/reference/libraries#client-libraries-install-go
- iam auth via login button?
  https://developers.google.com/identity/sign-in/web/sign-in
- css/better ui looks
- combined bucket + folder + file selector like open file dialog
- subfolders/prefixes
 	https://cloud.google.com/storage/docs/listing-objects#storage-list-objects-go
- preconditions and race conditions 
  https://cloud.google.com/storage/docs/generations-preconditions#examples_of_race_conditions
- rate limit on auth
- possibly use a better editor such as editarea, codemirror, monaco, codejar
  https://en.wikipedia.org/wiki/Comparison_of_JavaScript-based_source_code_editors
  or markdown editor like TinyMCE, CKEditor