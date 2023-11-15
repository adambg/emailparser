## emailparser

Parse raw email into email struct with text/plain, text/html and attachments  
based on https://github.com/kirabou/parseMIMEemail.go
### Usage

See `emailparser_test.go` for full example

```
import (
    github.com/adambg/emailparser
)

func main() {
    emlObject := Parse([]byte(demoEmail))
    
    fmt.Printf("From: %s", emlObject.From)
	fmt.Printf("To: %s", emlObject.To)
	fmt.Printf("Subject: %s", emlObject.Subject)
	fmt.Printf("Date: %s", emlObject.Date)
	fmt.Printf("BodyHtml: %s", emlObject.BodyHtml)
	fmt.Printf("BodyText: %s", emlObject.BodyText)
	fmt.Printf("ContentType: %s", emlObject.ContentType)
	fmt.Printf("Attchments: %d", len(emlObject.Attachments))
}
```
The return object is this:
```
type email struct {
	From        string        `json:"from"`
	To          string        `json:"to"`
	Subject     string        `json:"subject"`
	Date        string        `json:"date"`
	ContentType string        `json:"contenttype"`
	BodyText    string        `json:"bodytext"`
	BodyHtml    string        `json:"bodyhtml"`
	Attachments []attachments `json:"attachments"`
	Error       error         `json:"error"`
}

type attachments struct {
	Mimetype string `json:"mimetype"`
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}
```
