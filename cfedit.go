/*
 * Copyright 2021 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Cloud Functions Bucket File Editor
package cfedit

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/dustin/go-humanize"
	"google.golang.org/api/iterator"
)

var (
	projectID    = "MyProject" // only needed for listing buckets, if you specify a bucket this can be empty
	bucketName   = ""          // restrict only to this bucket, "" will list all buckets under projectID
	functionName = os.Getenv("K_SERVICE")

	users = []struct{ login, salt, hash string }{
		{login: "admin", salt: "pepper", hash: "sha256 hash of salt+passwd"}, // comment out to disable auth
	}
)

type bmClient struct {
	w http.ResponseWriter
	r *http.Request
	b *storage.BucketHandle
	c *storage.Client
}

func Error(w http.ResponseWriter, m string, e error) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Error: %v - %v\n", m, e)
}

func ListBuckets(ctx context.Context, c *storage.Client, bSel string) (string, error) {
	var out strings.Builder
	//var bSel string
	fmt.Fprintf(&out, "<form action=\"/%s\" method=\"post\"\">\n<select name=\"b\">\n", html.EscapeString(functionName))
	var m = make(map[string]string)
	m[bSel] = "selected"

	b := c.Buckets(ctx, projectID)
	for {
		a, err := b.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&out, "<option value=\"%v\" %v>%v</option>\n",
			html.EscapeString(a.Name),
			m[a.Name],
			a.Name)
	}
	fmt.Fprint(&out, "</select>\n<input type=\"submit\" value=\"get files\">\n</form>\n<p>\n")

	if bSel == "" {
		fmt.Fprint(&out, "</center>\n</body>\n</html>\n")
	}
	return out.String(), nil
}

func (c *bmClient) ListObjects(ctx context.Context) {
	var out strings.Builder
	out.WriteString("<html>\n<body>\n<center>\n")

	var bSel string
	ba, err := c.b.Attrs(ctx)
	if err == nil && ba.Name != "" {
		bSel = ba.Name
	}
	if bucketName == "" {
		o, err := ListBuckets(ctx, c.c, bSel)
		if err != nil {
			Error(c.w, "Listing buckets", err)
			return
		}
		out.WriteString(o)
	}
	if err != nil {
		if bucketName == "" {
			c.w.Write([]byte(out.String()))
			return
		}
		Error(c.w, "Opening a bucket", err)
		return
	}

	fmt.Fprintf(&out, "<form action=\"/%s?o=e&b=%v\" method=\"post\"\">\n"+
		"<select size=\"20\" name=\"f\" style=\"min-width: 400px;\">\n",
		html.EscapeString(functionName),
		html.EscapeString(ba.Name))

	o := c.b.Objects(ctx, &storage.Query{Prefix: ""})
	for {
		oa, err := o.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			Error(c.w, "Listing files", err)
			return
		}
		fmt.Fprintf(&out, "<option value=\"%v\">%v [%v]</option>\n",
			html.EscapeString(oa.Name),
			oa.Name,
			humanize.Bytes(uint64(oa.Size)),
		)
	}
	fmt.Fprint(&out, "</select>\n<p>\n<input type=\"submit\" value=\"edit file\">\n</form>\n</center>\n</body>\n</html>\n")

	c.w.Header().Set("Content-Type", "text/html")
	c.w.Write([]byte(out.String()))
}

func (c *bmClient) EditFile(ctx context.Context, f string) {
	ba, err := c.b.Attrs(ctx)
	if err != nil {
		Error(c.w, "Getting bucket attributes", err)
		return
	}

	a, err := c.b.Object(f).Attrs(ctx)
	if err != nil {
		Error(c.w, "Getting file attributes", err)
		return
	}

	// getting latest generation specifically circumvents getting cdn cached version
	r, err := c.b.Object(f).Generation(a.Generation).NewReader(ctx)
	if err != nil {
		Error(c.w, "Opening file", err)
		return
	}
	defer r.Close()

	d, err := ioutil.ReadAll(r)
	if err != nil {
		Error(c.w, "Reading file", err)
		return
	}

	c.w.Header().Set("Content-Type", "text/html")

	fmt.Fprintf(c.w, "<html>\n<body>\n"+
		"<form name=\"edit\" action=\"/%v?o=s&b=%v&f=%v\" method=\"post\" enctype=\"multipart/form-data\">\n"+
		"<textarea name=\"c\" spellcheck=\"false\" style=\"width: 100%%; height: 90%%\">\n",
		html.EscapeString(functionName),
		html.EscapeString(ba.Name),
		html.EscapeString(f))

	c.w.Write([]byte(html.EscapeString(string(d))))

	fmt.Fprintf(c.w, "</textarea><p>\n"+
		"<input type=\"submit\" value=\"save\" style=\"float: left;\"></form>\n"+
		"<form  action=\"/%v?b=%v\" method=\"post\"><input type=\"submit\" value=\"cancel\" style=\"float: left; margin-left: 10px\">\n"+
		"</form>\n</body>\n</html>\n", html.EscapeString(functionName), html.EscapeString(ba.Name))
}

func (c *bmClient) WriteFile(ctx context.Context, f string) {
	ba, err := c.b.Attrs(ctx)
	if err != nil {
		Error(c.w, "Getting bucket attributes", err)
		return
	}

	co := c.r.FormValue("c")
	if err != nil {
		Error(c.w, "Parsing form", err)
		return
	}

	if len(co) == 0 {
		Error(c.w, "Got 0 size", err)
		return
	}

	bf := c.b.Object(f).NewWriter(ctx)
	bf.ContentType = http.DetectContentType([]byte(co))
	n, err := bf.Write([]byte(co))
	if err != nil {
		Error(c.w, "Writing file", err)
		return
	}

	err = bf.Close()
	if err != nil {
		Error(c.w, "Closing file", err)
		return
	}
	if n != len(co) {
		Error(c.w, "File lenght", fmt.Errorf("form:%v != bucket:%v", len(co), n))
		return
	}

	http.Redirect(c.w, c.r, fmt.Sprintf("/%v?o=l&b=%v", html.EscapeString(functionName), html.EscapeString(ba.Name)), http.StatusTemporaryRedirect)
}

func Auth(w http.ResponseWriter, r *http.Request) bool {
	if len(users) == 0 {
		return true
	}
	var s string
	u, p, ok := r.BasicAuth()
	if !ok {
		goto unauth
	}
	for _, usr := range users {
		if subtle.ConstantTimeCompare([]byte(u), []byte(usr.login)) != 1 {
			continue
		}

		s = fmt.Sprintf("%x", sha256.Sum256([]byte(usr.salt+p)))
		if subtle.ConstantTimeCompare([]byte(s), []byte(usr.hash)) == 1 {
			return true
		}
	}
unauth:
	w.Header().Set("WWW-Authenticate", "Basic realm=\"cfedit\"")
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	return false
}

func CFEdit(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	if !Auth(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*30)
	defer cancel()

	c, err := storage.NewClient(ctx)
	if err != nil {
		Error(w, "Opening storage client", err)
		return
	}

	if bucketName != "" {
		r.Form.Set("b", bucketName)
	}
	b := c.Bucket(r.FormValue("b"))

	bc := &bmClient{w, r, b, c}

	switch r.FormValue("o") {
	case "e":
		bc.EditFile(ctx, r.FormValue("f"))
	case "s":
		bc.WriteFile(ctx, r.FormValue("f"))
	default:
		bc.ListObjects(ctx)
	}
}
