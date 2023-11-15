package emailparser

import (
	"testing"
)

var demoEmail = `From me@company.com  Tue Nov 14 21:49:33 2023
Return-Path: <me@company.com>
Received: from mail.someserver.com (mail.someserver.com [1.2.3.4])
        by copmany (Postfix) with ESMTPS id 344115AAE
        for <someone@othercompany.com>; Tue, 14 Nov 2023 21:49:33 +0000 (UTC)
Received: by mail-yb1-f182.google.com with SMTP id 3f1490d57ef6-da0359751dbso221233276.1
        for <someone@othercompany.com>; Tue, 14 Nov 2023 13:49:33 -0800 (PST)
MIME-Version: 1.0
From: Adam Ben-Gur <me@company.com>
Date: Tue, 14 Nov 2023 23:49:20 +0200
Message-ID: <CADa7U0EgMuL_34CWh5AJ48mvPStqvUA1PQfiOD+NmUfVUW5z2w@mail.company.com>
Subject: Test email
To: someone@othercompany.com
Content-Type: multipart/alternative; boundary="000000000000f79b3c060a23c2fc"

--000000000000f79b3c060a23c2fc
Content-Type: text/plain; charset="UTF-8"

This is a test

--000000000000f79b3c060a23c2fc
Content-Type: text/html; charset="UTF-8"

<div dir="ltr">This is a test</div>

--000000000000f79b3c060a23c2fc--`

func TestParse(t *testing.T) {
	// var emlObject email
	emlObject := Parse([]byte(demoEmail))

	// fmt.Printf("From: %s", emlObject.From)
	// fmt.Printf("To: %s", emlObject.To)
	// fmt.Printf("Subject: %s", emlObject.Subject)
	// fmt.Printf("Date: %s", emlObject.Date)
	// fmt.Printf("BodyHtml: %s", emlObject.BodyHtml)
	// fmt.Printf("BodyText: %s", emlObject.BodyText)
	// fmt.Printf("ContentType: %s", emlObject.ContentType)
	// fmt.Printf("Attchments: %d", len(emlObject.Attachments))

	if emlObject.Error != nil {
		t.Errorf("TestParse got error %+v", emlObject.Error)
	}
}

func TestError(t *testing.T) {
	// var emlObject email
	emlObject := Parse([]byte("some string that does not represent raw email"))

	if emlObject.Error == nil {
		t.Errorf("TestError resulted with no error although it should have")
	}
}
