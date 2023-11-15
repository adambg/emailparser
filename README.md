## emailparser

Parse raw email into email struct with text/plain, text/html and attachments  
based on https://github.com/kirabou/parseMIMEemail.go
### Usage

See `emailparser_test.go` for full example

```
import (
    "github.com/adambg/emailparser"
)

func main() {
    in := bufio.NewReader(os.Stdin)
	rawEmail, _ := io.ReadAll(in)

    eml := emailparser.Parse([]byte(demoEmail))
    
	fmt.Println("From: ", eml.From)
	fmt.Println("To: ", eml.To)
	fmt.Println("Subject: ", eml.Subject)
	fmt.Println("Date: ", eml.Date)
	fmt.Println("BodyHtml: ", eml.BodyHtml)
	fmt.Println("BodyText: ", eml.BodyText)
	fmt.Println("ContentType: ", eml.ContentType)
	fmt.Println("Attchments: ", len(eml.Attachments))
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
