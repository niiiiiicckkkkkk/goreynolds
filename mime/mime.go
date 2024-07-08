package mime

import (
	"encoding/base64"
	"fmt"
	"os"
	"slices"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// takes in message id, from name, from addr, to name, to addr, base64 image
var skeleton string = "MIME-Version: 1.0\n" +
	"Message-ID: <superduperimportantreynoldsbotmessage>\n" +
	"Date: Fri, 5 Jul 2024 09:45:36 -0400\n" +
	"Subject: %s\n" +
	"From: reynoldsbot <reynoldsbot70@gmail.com>\n" +
	"To: %s\n" +
	"CC: %s\n" +
	"BCC: %s\n" +
	"In-Reply-To: %s\n" +
	"References: %s\n" +
	"Content-Type: multipart/mixed; boundary=\"mrmacncheesesayshi\"\n\n" +
	"--mrmacncheesesayshi\n" +
	"Content-Type: multipart/alternative; boundary=\"flushythetoiletpee\"\n\n" +
	"--flushythetoiletpee\n" +
	"Content-Type: text/plain; charset=\"UTF-8\"\n\n\n\n" +
	"--flushythetoiletpee\n" +
	"Content-Type: text/html; charset=\"UTF-8\"\n\n" +
	"<h1>Reynolds</h1>\n<img src=\"cid:happyryan\"/>\n" +
	"--flushythetoiletpee--\n" +
	"--mrmacncheesesayshi\n" +
	"Content-Type: image/jpeg; name=\"reynolds.jpg\"\n" +
	"Content-Disposition: attachment; filename=\"reynolds.jpg\"\n" +
	"Content-Transfer-Encoding: base64\n" +
	"X-Attachment-Id: happyryan\n" +
	"Content-ID: <happyryan>\n\n%s\n\n" +
	"--mrmacncheesesayshi--"

func readFile() []byte {
	file, err := os.Open("resources/reynolds1.jpg")
	if err != nil {
		fmt.Println(err.Error())
	}

	buff := make([]byte, 100)
	data := make([]byte, 0)
	var offset int64 = 0
	for n, _ := file.ReadAt(buff, offset); n > 0; n, _ = file.ReadAt(buff, offset) {
		offset += int64(n)
		data = slices.Concat(data, buff[0:n])
	}
	return data
}

// walk the mime and apply a function
func walk(mime *gmail.MessagePart, f func(*gmail.MessagePart)) {

	f(mime)

	for _, part := range mime.Parts {
		walk(part, f)
	}
}

func Body(mime *gmail.MessagePart) string {
	text := make([]byte, 200)

	walk(mime, func(m *gmail.MessagePart) {
		if strings.HasPrefix(m.MimeType, "text/plain") {
			// TODO : handle errors here
			base64.StdEncoding.Decode(text, []byte(m.Body.Data))
		}
	})

	return string(text)
}

func Header(key string, mime *gmail.MessagePart) string {
	headers := mime.Headers

	for _, h := range headers {
		if strings.ToLower(key) == strings.ToLower(h.Name) {
			return h.Value
		}
	}
	return ""
}

func Reynolds(subject, to, cc, bcc, replyto, references string) string {
	img := readFile()
	img64 := make([]byte, base64.StdEncoding.EncodedLen(len(img)))
	base64.StdEncoding.Encode(img64, img)
	mail := fmt.Sprintf(skeleton, subject, to, cc, bcc, replyto, references, string(img64))

	out := make([]byte, base64.StdEncoding.EncodedLen(len(mail)))
	base64.StdEncoding.Encode(out, []byte(mail))
	return string(out)
}
