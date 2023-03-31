package main

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"git.maharshi.ninja/root/rss2email/structures"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/net/idna"

	smtp "github.com/xhit/go-simple-mail/v2"
)

func (c *connection) handleEmailVerification(mi *MessageInfo, buf []byte) {
	var req structures.VerifyEmailRequest
	ok := c.decodeToInterface(buf, &req)
	if !ok {
		return
	}

	u := c.getUser(mi)
	if u == nil {
		return
	}

	if u.EmailVerified {
		c.writeMessage(false, mi, structures.ErrorMessage{
			Code: structures.ErrorInvalidInputs,
			Message: c.localizer.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "Errors.AlreadyVerified",
			}),
		})
		return
	}

	if subtle.ConstantTimeCompare(u.EmailVerificationToken[:], req.Token[:]) != 1 {
		c.writeMessage(false, mi, structures.ErrorMessage{
			Code: structures.ErrorInvalidInputs,
			Message: c.localizer.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "Errors.InvalidVerificationToken",
			}),
		})
		return
	}

	c.a.users.UpdateByID(context.TODO(), c.userID, bson.M{
		"$set": bson.M{
			"email_verification_last": time.Now(),
			"email_verified":          true,
		},
	})
	c.writeMessage(true, mi, structures.GenericIDResponse{
		OK: true,
		ID: c.userID,
	})
}

func (c *connection) sendVerificationEmail(user *structures.User) {
	if time.Since(user.EmailVerificationLast) <= 6*time.Hour {
		log.Printf("Ignoring email verification request since it was sent within the past 6 hours. User ID: %#v\n", user.ID)
		return
	}

	emailContent := c.localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "Emails.Verification",
		TemplateData: map[string]string{
			"BaseURL": c.a.config.BaseURL,
			"Code":    hex.EncodeToString(user.EmailVerificationToken[:]),
		},
	})
	emailSubject := c.localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "Emails.VerificationSubject",
	})

	msg := smtp.NewMSG()
	msg.SetFrom(c.a.config.EmailConfig.FromAddr)
	msg.SetSubject(emailSubject)
	msg.SetBody(smtp.TextPlain, emailContent)

	{
		eParts := strings.Split(user.Email, "@")
		if (len(eParts) != 2) {
			log.Printf("Ignoring invalid e-mail address, User Id: %#v\n", user.ID)
			return
		}
		x, err := idna.Punycode.ToASCII(eParts[1])
		if err != nil {
			log.Printf("Caught error converting from Unicode to Punycode: %s\n", err.Error())
			return
		}
		eParts[1] = x
		msg.AddTo(strings.Join(eParts, "@"))
	}

	conn, err := c.a.emailClient.Connect()
	if err != nil {
		log.Printf("Error while connecting to SMTP: %s\n", err.Error())
		return
	}

	err = msg.Send(conn)
	if err != nil {
		log.Printf("Error while sending to SMTP: %s\n", err.Error())
		return
	}

	_, err = c.a.users.UpdateByID(context.TODO(), user.ID, bson.M{
		"$set": bson.M{
			"email_verification_last": time.Now(),
		},
	})
	if err != nil {
		log.Printf("Updating EVL didn't work, user ID: %#v\n", user.ID)
		return
	}
}
