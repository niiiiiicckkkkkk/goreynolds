package main

import (
	"fmt"
	"io"
	"reynolds/mime"
	"slices"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

type logger struct {
	buff []byte
}

type mail struct {
	messageId string
	threadId  string
	thread    []*gmail.MessagePart
	logger    *logger
}

func (l *logger) Write(p []byte) (int, error) {
	l.buff = slices.Concat(l.buff, p)
	return len(p), nil
}

func (l *logger) Read(p []byte) (int, error) {
	var n int
	var e error
	for i, _ := range p {
		if i >= len(l.buff) {
			e = io.EOF
			break
		} else {
			p[i] = l.buff[i]
			n += 1
		}
	}
	l.buff = l.buff[n:]
	return n, e
}

func newmail(mid, tid string) mail {
	messages := make([]*gmail.MessagePart, 0)
	l := logger{make([]byte, 0)}
	return mail{mid, tid, messages, &l}
}

func mapThread[A any](m mail, f func(*gmail.MessagePart) A) []A {
	out := make([]A, len(m.thread))

	for _, t := range m.thread {
		out = append(out, f(t))
	}
	return out
}

func (mail *mail) log(s string) {
	mid := mail.messageId
	tid := mail.threadId
	time := time.Now().String()

	l := fmt.Sprintf("{message : %s, thread : %s, time : %s, log : %s\n}", mid, tid, time, s)
	mail.logger.Write([]byte(l))
}

func (mail *mail) lastEntry() string {
	n := len(mail.thread)
	before := mime.Body(mail.thread[n-1])
	for i := n - 2; i >= 0; i-- {
		after := mime.Body(mail.thread[i])

		diff, b := strings.CutSuffix(after, before)

		if b {
			return diff
		}
	}
	// assume start of a new thread
	return before
}

func (mail *mail) pullMail(gmailService *gmail.Service) {
	threadService := gmail.NewUsersThreadsService(gmailService)
	msgService := gmail.NewUsersMessagesService(gmailService)
	thread, err := threadService.Get("me", mail.threadId).Do()

	if err != nil {
		mail.log(err.Error())
		return
	}
	for i := 0; i < len(thread.Messages); i++ {
		id := thread.Messages[i].Id
		rsp, err := msgService.Get("me", id).Do()
		if err != nil {
			mail.log(err.Error())
			return
		} else {
			mail.thread = append(mail.thread, rsp.Payload)
		}
		if id == mail.messageId {
			break
		}
	}

}
