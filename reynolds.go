package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"reynolds/mime"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

// watch the inbox
func watchInbox(gmailService *gmail.Service, updates chan mail) {
	// sync bot with inbox state
	userService := gmail.NewUsersService(gmailService)
	profile, _ := userService.GetProfile("me").Do()

	historyId := profile.HistoryId

	// poll for new emails
	historyService := gmail.NewUsersHistoryService(gmailService)
	base := historyService.List("me").HistoryTypes("messageAdded")

	for ; ; time.Sleep(time.Minute) {
		update, _ := base.StartHistoryId(historyId).Do()

		for _, history := range update.History {

			for _, record := range history.MessagesAdded {
				m := newmail(record.Message.Id, record.Message.ThreadId)
				m.log("received email to inbox")
				updates <- m
			}
		}
		// TODO : nextpage param??
		historyId = update.HistoryId
	}
}

func scanMsgs(gmailService *gmail.Service, updates, worklist, completed chan mail) {

	for update := range updates {

		update.pullMail(gmailService)
		request := update.thread[len(update.thread)-1]
		if strings.Contains(mime.Header("From", request), "reynoldsbot70@gmail.com") {
			update.log("skipped message from reynoldsbot70@gmail.com")
		}

		body := update.lastEntry()

		update.log(fmt.Sprintf("read email plaintext [%s]", body))

		// check if body includes "!reynolds"
		if strings.Contains(body, "!reynolds") {
			update.log("reynolds request received, added to worklist")
			worklist <- update
		} else {
			update.log("did not contain reynolds request")
			completed <- update
		}
	}

}

func sendReynolds(gmailService *gmail.Service, worklist, completed chan mail) {
	draftService := gmail.NewUsersDraftsService(gmailService)

	for m := range worklist {
		var msg gmail.Message
		m.log("prepping reynolds draft")
		var _draft gmail.Draft
		msg.ThreadId = m.threadId
		_references := mapThread(m, func(m *gmail.MessagePart) string {
			return mime.Header("Message-Id", m)
		})
		request := m.thread[len(m.thread)-1]
		replyTo := _references[len(_references)-1]
		references := strings.Join(_references, " ")
		subject := mime.Header("Subject", request)
		to := mime.Header("To", request)
		from := mime.Header("From", request)
		cc := mime.Header("CC", request)
		bcc := mime.Header("BCC", request)

		to = fmt.Sprintf("%s, %s", to, from)

		msg.Raw = mime.Reynolds(subject, to, cc, bcc, replyTo, references)
		_draft.Message = &msg
		draft, err := draftService.Create("me", &_draft).Do()
		if err != nil {
			m.log(fmt.Sprintf("failed to create draft %s", err.Error()))
		}
		_, err = draftService.Send("me", draft).Do()
		if err != nil {
			m.log(fmt.Sprintf("failed to send draft %s", err.Error()))
		} else {
			m.log("message delivered")
		}
		completed <- m
	}
}

func main() {
	// authenticate gcloud magic
	ctx := context.Background()
	gmailService, err := gmail.NewService(ctx)
	if err != nil {
		fmt.Println("error")
	}

	completed, updates, worklist := make(chan mail, 50), make(chan mail, 50), make(chan mail, 50)

	// spin off a thread to continuously check the mailbox and new messages ids to a channel
	go watchInbox(gmailService, updates)

	// spin off another thread to continuously pull message ids and evaluate for !reynolds
	go scanMsgs(gmailService, updates, worklist, completed)

	// another thread to send the ryan reynolds images
	go sendReynolds(gmailService, worklist, completed)

	for m := range completed {
		io.Copy(os.Stdout, m.logger)
	}

}
