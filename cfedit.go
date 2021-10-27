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
	"time"

	"cloud.google.com/go/storage"
	"github.com/dustin/go-humanize"
	"google.golang.org/api/iterator"
)

var (
	projectID    = "myproject"
	bucketName   = "" // restrict only to this bucket, "" all buckets
	functionName = os.Getenv("K_SERVICE")

	users = []struct {
		username, sha256pw string
	}{ // to generate password hash: echo -n "mypassword" | shasum -a 256
		{username: "admin", sha256pw: "5234lkj14j34foobar"},
	}
)

type bmClient struct {
	w http.ResponseWriter
	r *http.Request
	b *storage.BucketHandle
	c *storage.Client
}

func Error(w http.ResponseWriter, h bool, m string, e error) {
	if !h {
		w.Header().Set("Content-Type", "text/plain")
	}
	fmt.Fprintf(w, "Error: %v - %v\n", m, e)
	if h {
		fmt.Fprintln(w, "</html></body>")
	}
}

func (c *bmClient) ListObjects(ctx context.Context) {
	c.w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(c.w, "<html><body><center>")
	var bName string
	ba, err := c.b.Attrs(ctx)
	if err == nil && ba.Name != "" {
		bName = ba.Name
	}

	fmt.Fprintf(c.w, "<form action=\"/%s\" method=\"post\"\">\n<select name=\"b\">\n", html.EscapeString(functionName))
	var m = make(map[string]string)
	m[bName] = "selected"

	b := c.c.Buckets(ctx, projectID)
	for {
		a, err := b.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			Error(c.w, true, "Listing buckets", err)
			return
		}
		if bucketName != "" && a.Name != bucketName {
			continue
		}
		fmt.Fprintf(c.w, "<option value=\"%v\" %v>%v</option>\n",
			html.EscapeString(a.Name),
			m[a.Name],
			a.Name)
	}
	fmt.Fprint(c.w, "</select>\n<input type=\"submit\" value=\"get files\">\n</form>\n<p>\n")

	if bName == "" {
		fmt.Fprint(c.w, "</center>\n</body>\n</html>\n")
		return
	}

	fmt.Fprintf(c.w, "<form action=\"/%s?o=e&b=%v\" method=\"post\"\">\n"+
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
			Error(c.w, true, "Listing files", err)
			return
		}
		fmt.Fprintf(c.w, "<option value=\"%v\">%v [%v]</option>\n",
			html.EscapeString(oa.Name),
			oa.Name,
			humanize.Bytes(uint64(oa.Size)),
		)
	}
	fmt.Fprint(c.w, "</select>\n<p>\n<input type=\"submit\" value=\"edit file\">\n</form>\n</center>\n</body>\n</html>\n")
}

func (c *bmClient) DumpFile(ctx context.Context, f string, esc bool) {
	a, err := c.b.Object(f).Attrs(ctx)
	if err != nil {
		Error(c.w, false, "Getting file attributes", err)
		return
	}

	// getting latest generation circumvents file caching
	r, err := c.b.Object(f).Generation(a.Generation).NewReader(ctx)
	if err != nil {
		Error(c.w, false, "Opening file", err)
		return
	}
	defer r.Close()

	d, err := ioutil.ReadAll(r)
	if err != nil {
		Error(c.w, false, "Reading file", err)
		return
	}

	c.w.Header().Set("Content-Type", a.ContentType)
	if esc {
		c.w.Write([]byte(html.EscapeString(string(d))))
		return
	}
	c.w.Write(d)
}

func (c *bmClient) EditFile(ctx context.Context, f string) {
	ba, err := c.b.Attrs(ctx)
	if err != nil {
		Error(c.w, false, "Getting bucket attributes", err)
		return
	}

	c.w.Header().Set("Content-Type", "text/html")

	fmt.Fprintf(c.w, "<html>\n<body>\n"+
		"<form name=\"edit\" action=\"/%v?o=s&b=%v&f=%v\" method=\"post\" enctype=\"multipart/form-data\">\n"+
		"<textarea name=\"c\" spellcheck=\"false\" style=\"width: 100%%; height: 95%%\">\n",
		html.EscapeString(functionName),
		html.EscapeString(ba.Name),
		html.EscapeString(f))

	c.DumpFile(ctx, f, true)

	fmt.Fprintf(c.w, "</textarea>\n"+
		"<input type=\"submit\" name=\"txtbtn\" value=\"Save\"> \n"+
		"<input type=\"submit\" name=\"txtbtn\" value=\"Cancel\">\n"+
		"</form></body></html>\n")
}

func (c *bmClient) WriteFile(ctx context.Context, f string) {
	ba, err := c.b.Attrs(ctx)
	if err != nil {
		Error(c.w, false, "Getting bucket attributes", err)
		return
	}

	co := c.r.FormValue("c")
	if err != nil {
		Error(c.w, false, "Parsing form", err)
		return
	}

	if len(co) == 0 {
		Error(c.w, false, "Got 0 size", err)
		return
	}

	bf := c.b.Object(f).NewWriter(ctx)
	bf.ContentType = http.DetectContentType([]byte(co))
	n, err := bf.Write([]byte(co))
	if err != nil {
		Error(c.w, false, "Writing file", err)
		return
	}

	err = bf.Close()
	if err != nil {
		Error(c.w, false, "Closing file", err)
		return
	}
	if n != len(co) {
		Error(c.w, false, "File lenght", fmt.Errorf("form:%v != bucket:%v", len(co), n))
		return
	}

	http.Redirect(c.w, c.r, fmt.Sprintf("/%v?o=l&b=%v", html.EscapeString(functionName), html.EscapeString(ba.Name)), http.StatusTemporaryRedirect)
}

func (c *bmClient) Auth(ctx context.Context) bool {
	if len(users) == 0 {
		return true
	}
	var s string
	u, p, ok := c.r.BasicAuth()
	if !ok {
		goto unauth
	}
	s = fmt.Sprintf("%x", sha256.Sum256([]byte(p)))
	for _, usr := range users {
		if subtle.ConstantTimeCompare([]byte(u), []byte(usr.username)) == 1 && subtle.ConstantTimeCompare([]byte(s), []byte(usr.sha256pw)) == 1 {
			return true
		}
	}
unauth:
	c.w.Header().Set("WWW-Authenticate", "Basic realm=\""+projectID+"\"")
	http.Error(c.w, "Unauthorized", http.StatusUnauthorized)
	return false
}

func Main(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*30)
	defer cancel()

	c, err := storage.NewClient(ctx)
	if err != nil {
		Error(w, false, "Opening storage client", err)
		return
	}

	r.ParseMultipartForm(10 << 20)

	if r.FormValue("txtbtn") == "Cancel" {
		r.Form.Set("o", "l")
	}
	if bucketName != "" {
		r.Form.Set("b", bucketName)
	}

	bc := &bmClient{
		w: w,
		r: r,
		b: c.Bucket(r.FormValue("b")),
		c: c,
	}

	if !bc.Auth(ctx) {
		return
	}

	switch r.FormValue("o") {
	case "d":
		bc.DumpFile(ctx, r.FormValue("f"), false)
	case "e":
		bc.EditFile(ctx, r.FormValue("f"))
	case "s":
		bc.WriteFile(ctx, r.FormValue("f"))
	default:
		bc.ListObjects(ctx)
	}
}
