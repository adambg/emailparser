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
)

type Attachments struct {
	Mimetype string `json:"mimetype"`
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}
type Email struct {
	From        string        `json:"from"`
	To          string        `json:"to"`
	Subject     string        `json:"subject"`
	Date        string        `json:"date"`
	ContentType string        `json:"contenttype"`
	BodyText    string        `json:"bodytext"`
	BodyHtml    string        `json:"bodyhtml"`
	Attachments []Attachments `json:"attachments"`
	Error       error         `json:"error"`
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
		eml.Error = err
		return &eml
	}

	// Display only the main headers of the message. The "From","To" and "Subject" headers
	// have to be decoded if they were encoded using RFC 2047 to allow non ASCII characters.
	// We use a mime.WordDecode for that.
	dec := new(mime.WordDecoder)

	eml.From, err = dec.DecodeHeader(m.Header.Get("From"))
	if err != nil {
		eml.Error = err
	}
	eml.To, err = dec.DecodeHeader(m.Header.Get("To"))
	if err != nil {
		eml.Error = err
	}
	eml.Subject, err = dec.DecodeHeader(m.Header.Get("Subject"))
	if err != nil {
		eml.Error = err
	}
	eml.Date = m.Header.Get("Date")
	eml.ContentType = m.Header.Get("Content-Type")

	mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		eml.Error = err
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		eml.Error = fmt.Errorf("not a multipart MIME message")
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

func htmlToText(inp string) (bodyText string) {

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
			if previousStartTokenTest.Data == "script" {
				continue
			}
			TxtContent := strings.TrimSpace(html.UnescapeString(string(domDocTest.Text())))
			if len(TxtContent) > 0 {
				bodyText = fmt.Sprintf("%s\n%s", bodyText, TxtContent)
			}
		}
	}
	return (bodyText)

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
		eml.Error = err
		return
	}

	content_transfer_encoding := strings.ToUpper(part.Header.Get("Content-Transfer-Encoding"))

	switch {

	case strings.Compare(content_transfer_encoding, "BASE64") == 0:
		decoded_content, err := base64.StdEncoding.DecodeString(string(part_data))
		if err != nil {
			eml.Error = err
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
			eml.Error = err
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
			eml.Error = err
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
