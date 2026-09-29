// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package guidance

import (
	"net/url"
	"strconv"
	"strings"
)

// Markdown renders guidance as a hover: the check id, what it means, the
// steps, and the documentation link when the document gave one.
func Markdown(g Guidance, doc string) string {
	var b strings.Builder
	b.WriteString("**" + escape(g.ID) + "**\n\n")
	if g.Means != "" {
		b.WriteString(g.Means + "\n\n")
	}
	for i, s := range g.Steps {
		b.WriteString(strconv.Itoa(i+1) + ". **" + escape(s.Title) + "**: " + s.Body + "\n")
	}
	if len(g.Steps) > 0 {
		b.WriteString("\n")
	}
	if link, ok := safeLink(doc); ok {
		b.WriteString("[Documentation](" + link + ")\n\n")
	}
	b.WriteString("_Guidance from passmcp's catalogue._")
	return b.String()
}

// safeLink accepts an absolute http(s) URL with nothing in it that could
// end the Markdown link early. The link comes from the document, which is
// not this program's to trust.
func safeLink(doc string) (string, bool) {
	u, err := url.Parse(doc)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return "", false
	}
	link := u.String()
	if strings.ContainsAny(link, "()<> \t\n") {
		return "", false
	}
	return link, true
}

// Unavailable renders a hover that says why there is no guidance.
func Unavailable(id, why string) string {
	return "**" + escape(id) + "**\n\n_" + why + "_"
}

// escape keeps an id from opening emphasis or a link in the hover.
func escape(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "`", "\\`")
	return r.Replace(s)
}
