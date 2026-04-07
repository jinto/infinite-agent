// EVAL-VULN: xss
// This file intentionally contains XSS vulnerabilities for eval testing.
// DO NOT use this code in production.

package vulnerable

import (
	"fmt"
	"net/http"
)

func RenderProfile(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	bio := r.FormValue("bio")

	// VULN: Reflected XSS - user input directly in HTML
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body>")
	fmt.Fprintf(w, "<h1>Welcome, %s!</h1>", name)
	fmt.Fprintf(w, "<div class='bio'>%s</div>", bio)
	fmt.Fprintf(w, "</body></html>")
}

func RenderComment(w http.ResponseWriter, comment string) {
	// VULN: Stored XSS - unsanitized comment rendered as HTML
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<div class='comment'>%s</div>", comment)
}

func RenderError(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("error")
	// VULN: Error message reflected without escaping
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<div class='error'>Error: %s</div>", msg)
}
