// Parse raw email into email struct with text/plain, text/html and attachments
// based on https://github.com/kirabou/parseMIMEemail.go
package emailparser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

type Attachments struct {
	Mimetype string `json:"mimetype"`
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}
type Email struct {
	From        string        `json:"from"`
	OrigTo      string        `json:"origto"`
	To          string        `json:"to"`
	CC          string        `json:"cc"`
	BCC         string        `json:"bcc"`
	Subject     string        `json:"subject"`
	Date        string        `json:"date"`
	ContentType string        `json:"contenttype"`
	BodyText    string        `json:"bodytext"`
	BodyHtml    string        `json:"bodyhtml"`
	Attachments []Attachments `json:"attachments"`
	Error       string        `json:"_"`
}

var eml Email

// Read a MIME multipart email from stdio and explode its MIME parts into
// separated files, one for each part.
func Parse(inp []byte) *Email {

	//  Parse the message to separate the Header and the Body with mail.ReadMessage()
	// m, err := mail.ReadMessage(os.Stdin)
	reader := bytes.NewReader(inp)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		eml.Error = err.Error()
		return &eml
	}

	// Display only the main headers of the message. The "From","To" and "Subject" headers
	// have to be decoded if they were encoded using RFC 2047 to allow non ASCII characters.
	// We use a mime.WordDecode for that.
	dec := new(mime.WordDecoder)

	eml.From, err = dec.DecodeHeader(m.Header.Get("From"))
	if err != nil {
		eml.Error = err.Error()
	}
	eml.To, err = dec.DecodeHeader(m.Header.Get("To"))
	if err != nil {
		eml.Error = err.Error()
	}
	eml.OrigTo, _ = dec.DecodeHeader(m.Header.Get("X-Forwarded-To"))
	if eml.OrigTo == "" {
		eml.OrigTo, _ = dec.DecodeHeader(m.Header.Get("X-Original-To"))
	}
	eml.CC, _ = dec.DecodeHeader(m.Header.Get("Cc"))
	eml.BCC, _ = dec.DecodeHeader(m.Header.Get("Bcc"))
	eml.Subject, err = dec.DecodeHeader(m.Header.Get("Subject"))
	if err != nil {
		eml.Error = err.Error()
	}
	eml.Date = m.Header.Get("Date")
	eml.ContentType = m.Header.Get("Content-Type")

	mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		eml.Error = err.Error()
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		eml.Error = "not a multipart MIME message"
	}

	// Recursivey parsed the MIME parts of the Body, starting with the first
	// level where the MIME parts are separated with params["boundary"].
	parsePart(m.Body, params["boundary"], 1)
	return &eml
}

// Some mail clients sends email with empty body, and the body itself comes as an HTML attachment.
// Check for length of email.bodyText, and if empty you can extract the body from one of the attachments.
// attachmentID is optional. If not set, function will search for the first attachment of content type HTML
// and extract the email from there.
// Function returns a new email and the attachment that was used to extract body (so you can discard it
// if you wish). If no attachment used it will return -1
func ExtractBodyFromHtmlAttachment(eml Email, attachmentID ...int) (*Email, int) {

	// First check if BodyHtml has content for those cases where only BodyHtml was sent without BodyText
	if len(eml.BodyHtml) > 0 {
		eml.BodyText = htmlToText(eml.BodyHtml)
		return &eml, -1
	} else {

		for i := 0; i < len(eml.Attachments); i++ {
			if eml.Attachments[i].Mimetype == "text/html" {
				eml.BodyText = htmlToText(string(eml.Attachments[i].Data))
				eml.BodyHtml = string(eml.Attachments[i].Data)
				return &eml, i
			}
		}
	}
	// return same input object as probably no body is in the email

	return &eml, -1
}

func fromISO88591(inp string) (string, error) {
	enc, name, ok := charset.DetermineEncoding([]byte(inp), "")
	fmt.Printf("%+v\n%+v\n%+v", enc, name, ok)

	r, err := charset.NewReaderLabel(name, strings.NewReader(inp))
	if err != nil {
		return "", err
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func htmlToTextOld(inp string) (bodyText string) {

	domDocTest := html.NewTokenizer(strings.NewReader(inp))
	previousStartTokenTest := domDocTest.Token()

loopDomTest:
	for {
		tt := domDocTest.Next()
		switch {
		case tt == html.ErrorToken:
			break loopDomTest // End of the document,  done
		case tt == html.StartTagToken:
			previousStartTokenTest = domDocTest.Token()
		case tt == html.TextToken:
			if previousStartTokenTest.Data == "script" || previousStartTokenTest.Data == "style" {
				continue
			}
			TxtContent := strings.TrimSpace(html.UnescapeString(string(domDocTest.Text())))
			if len(TxtContent) > 0 {
				bodyText = fmt.Sprintf("%s\n%s", bodyText, TxtContent)
			}
		}
	}

	// fmt.Printf("ORIG: %s", bodyText)
	// bodyText, _ = fromISO88591(bodyText)

	return (bodyText)

}

func htmlToText(inp string) string {
	doc, err := html.Parse(strings.NewReader(inp))
	if err != nil {
		return htmlToTextTokenizer(inp)

	}

	var result strings.Builder
	extractText(doc, &result)
	return strings.TrimSpace(result.String())
}

func extractText(n *html.Node, result *strings.Builder) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			result.WriteString(text)
			result.WriteString(" ")
		}
		return
	}

	if n.Type == html.ElementNode {
		switch n.Data {
		case "script", "style", "head":
			// Skip these elements entirely
			return
		case "a":
			// Handle links specially
			handleLink(n, result)
			return
		case "br", "p", "div", "h1", "h2", "h3", "h4", "h5", "h6":
			// Add line breaks for block elements
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extractText(c, result)
			}
			result.WriteString("\n")
			return
		}
	}

	// Process all child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, result)
	}
}

func handleLink(n *html.Node, result *strings.Builder) {
	// Extract href attribute
	var href string
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			href = attr.Val
			break
		}
	}

	// Extract link text
	var linkText strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, &linkText)
	}

	text := strings.TrimSpace(linkText.String())
	if text == "" {
		text = href // Use URL as text if no text content
	}

	if href != "" {
		// Format: "link text (URL)" or just "link text" if URL is same as text
		if href != text && !strings.Contains(text, href) {
			result.WriteString(fmt.Sprintf("%s %s", text, href))
		} else {
			result.WriteString(text)
		}
	} else {
		result.WriteString(text)
	}
	result.WriteString(" ")
}

// Alternative approach using tokenizer with better state tracking
func htmlToTextTokenizer(inp string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(inp))
	var result strings.Builder
	var tagStack []string

	for {
		tokenType := tokenizer.Next()

		switch tokenType {
		case html.ErrorToken:
			return strings.TrimSpace(result.String())

		case html.StartTagToken:
			token := tokenizer.Token()
			tagStack = append(tagStack, token.Data)

			// Handle link tags
			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						// Store href for later use
						tagStack[len(tagStack)-1] = "a:" + attr.Val
						break
					}
				}
			}

			// Add line breaks for block elements
			if isBlockElement(token.Data) {
				result.WriteString("\n")
			}

		case html.EndTagToken:
			token := tokenizer.Token()
			if len(tagStack) > 0 {
				lastTag := tagStack[len(tagStack)-1]
				if strings.HasPrefix(lastTag, "a:") {
					// This was a link with href
					href := strings.TrimPrefix(lastTag, "a:")
					if href != "" {
						result.WriteString(fmt.Sprintf(" (%s)", href))
					}
				}
				tagStack = tagStack[:len(tagStack)-1]
			}

			// Add line breaks for block elements
			if isBlockElement(token.Data) {
				result.WriteString("\n")
			}

		case html.TextToken:
			// Skip text inside script/style tags
			if len(tagStack) > 0 {
				currentTag := tagStack[len(tagStack)-1]
				if strings.HasPrefix(currentTag, "script") ||
					strings.HasPrefix(currentTag, "style") ||
					strings.HasPrefix(currentTag, "head") {
					continue
				}
			}

			text := strings.TrimSpace(html.UnescapeString(string(tokenizer.Text())))
			if text != "" {
				result.WriteString(text)
				result.WriteString(" ")
			}
		}
	}
}

func isBlockElement(tag string) bool {
	blockElements := map[string]bool{
		"p": true, "div": true, "br": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"ul": true, "ol": true, "li": true,
		"table": true, "tr": true, "td": true, "th": true,
		"blockquote": true, "pre": true,
	}
	return blockElements[tag]
}

// BuildFileName builds a file name for a MIME part, using information extracted from
// the part itself, as well as a radix and an index given as parameters.
func buildFileName(part *multipart.Part, radix string, index int) (filename string) {

	// 1st try to get the true file name if there is one in Content-Disposition
	filename = part.FileName()
	if len(filename) > 0 {
		return
	}

	// If no defaut filename defined, try to build one of the following format :
	// "radix-index.ext" where extension is comuputed from the Content-Type of the part
	mediaType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	if err == nil {
		mime_type, e := mime.ExtensionsByType(mediaType)
		if e == nil {
			return fmt.Sprintf("%s-%d%s", radix, index, mime_type[0])
		}
	}
	return
}

// WitePart decodes the data of MIME part and writes it to the file filename.
func writePart(part *multipart.Part, filename string, mediaType string) {

	// Read the data for this MIME part
	part_data, err := io.ReadAll(part)
	if err != nil {
		eml.Error = err.Error()
		return
	}

	content_transfer_encoding := strings.ToUpper(part.Header.Get("Content-Transfer-Encoding"))

	switch {

	case strings.Compare(content_transfer_encoding, "BASE64") == 0:
		decoded_content, err := base64.StdEncoding.DecodeString(string(part_data))
		if err != nil {
			eml.Error = err.Error()
		} else {
			// ioutil.WriteFile(filename, decoded_content, 0644)
			var atch Attachments
			atch.Filename = filename
			atch.Mimetype = mediaType
			atch.Data = decoded_content
			eml.Attachments = append(eml.Attachments, atch)
		}

	case strings.Compare(content_transfer_encoding, "QUOTED-PRINTABLE") == 0:
		decoded_content, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(part_data)))
		if err != nil {
			eml.Error = err.Error()
		} else {
			// ioutil.WriteFile(filename, decoded_content, 0644)
			var atch Attachments
			atch.Filename = filename
			atch.Mimetype = mediaType
			atch.Data = decoded_content
			eml.Attachments = append(eml.Attachments, atch)
		}

	default:
		// ioutil.WriteFile(filename, part_data, 0644)
		var atch Attachments
		atch.Filename = filename
		atch.Mimetype = mediaType
		atch.Data = part_data
		eml.Attachments = append(eml.Attachments, atch)
		if mediaType == "text/plain" {
			eml.BodyText = string(part_data)
		} else if mediaType == "text/html" {
			eml.BodyHtml = string(part_data)
		}
	}
}

// ParsePart parses the MIME part from mime_data, each part being separated by
// boundary. If one of the part read is itself a multipart MIME part, the
// function calls itself to recursively parse all the parts. The parts read
// are decoded and written to separate files, named uppon their Content-Descrption
// (or boundary if no Content-Description available) with the appropriate
// file extension. Index is incremented at each recursive level and is used in
// building the filename where the part is written, as to ensure all filenames
// are distinct.
func parsePart(mime_data io.Reader, boundary string, index int) {

	// Instantiate a new io.Reader dedicated to MIME multipart parsing
	// using multipart.NewReader()
	reader := multipart.NewReader(mime_data, boundary)
	if reader == nil {
		return
	}

	// Go through each of the MIME part of the message Body with NextPart(),
	// and read the content of the MIME part with ioutil.ReadAll()
	for {

		new_part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			eml.Error = err.Error()
			break
		}

		mediaType, params, err := mime.ParseMediaType(new_part.Header.Get("Content-Type"))

		if err == nil && strings.HasPrefix(mediaType, "multipart/") {
			parsePart(new_part, params["boundary"], index+1)
		} else {
			filename := buildFileName(new_part, boundary, 1)
			writePart(new_part, filename, mediaType)
		}
	}
}
