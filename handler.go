package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"time"

	"git.maharshi.ninja/root/rss2email/structures"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/storyicon/sigverify"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"nhooyr.io/websocket"
)

func (a *app) handler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	defer c.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		log.Printf("Error: %s\n", err)
	}
	conn := connection{
		a:    a,
		conn: c,
	}
	conn.loop()
}

type connection struct {
	a         *app
	conn      *websocket.Conn
	addr      [20]byte
	userID    primitive.ObjectID
	localizer *i18n.Localizer
}

func (c *connection) loop() {
	{
		var ir structures.InitializationRequest
		mi, ok := c.readMessage(&ir)
		if !ok {
			return
		}

		c.addr = ir.Address
		c.localizer = i18n.NewLocalizer(c.a.i18nBundle, ir.Locale)

		randBuf := make([]byte, 32)
		_, err := io.ReadFull(rand.Reader, randBuf)
		if err != nil {
			c.writeError(mi, structures.ErrorInternal, err)
			return
		}

		siweMessage := []byte(c.localizer.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "General.ChallengeMessage",
			TemplateData: map[string]string{
				"Data": base64.StdEncoding.EncodeToString(randBuf),
			},
		}))

		var user structures.User
		err = c.a.users.FindOne(context.TODO(), map[string][20]byte{
			"addr": ir.Address,
		}).Decode(&user)

		if err == mongo.ErrNoDocuments {
			c.writeMessage(true, mi, structures.InitializationResponse{
				UserFound: false,
				Challenge: siweMessage,
			})

			var userCreationReq structures.NewUserInitialization
			mi, ok := c.readMessage(&userCreationReq)
			if !ok {
				return
			}

			ok, err = sigverify.VerifyEllipticCurveSignature(c.addr, siweMessage, userCreationReq.Signature[:])
			if !ok {
				if err != nil {
					c.writeError(mi, structures.ErrorInvalidSignature, err)
				} else {
					c.writeMessage(false, mi, structures.ErrorMessage{
						Code: structures.ErrorInvalidSignature,
						Message: c.localizer.MustLocalize(&i18n.LocalizeConfig{
							MessageID: "Errors.InvalidSignature",
						}),
					})
				}
				return
			}

			docsWithSameEmail, err := c.a.users.CountDocuments(context.TODO(), bson.M{
				"email": userCreationReq.Email,
			})
			if err != nil {
				c.writeError(mi, structures.ErrorInternal, err)
				return
			}
			if docsWithSameEmail != 0 {
				c.writeMessage(false, mi, structures.ErrorMessage{
					Code: structures.ErrorInvalidInputs,
					Message: c.localizer.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "Errors.AccountWithSameEmail",
					}),
				})
				return
			}

			user.ID = primitive.NewObjectID()
			user.CreatedAt = time.Now()
			user.UpdatedAt = time.Now()
			user.Email = userCreationReq.Email
			user.Address = c.addr

			verificationToken := make([]byte, 32)
			_, err = io.ReadFull(rand.Reader, verificationToken)
			if err != nil {
				c.writeError(mi, structures.ErrorInternal, err)
				return
			}
			copy(user.EmailVerificationToken[:], verificationToken)
			user.EmailVerified = false

			_, err = c.a.users.InsertOne(context.TODO(), user)
			if err != nil {
				c.writeError(mi, structures.ErrorInternal, err)
				return
			}
		} else {
			c.writeMessage(true, mi, structures.InitializationResponse{
				UserFound: true,
				Challenge: siweMessage,
			})

			var ordinaryResponse structures.OrdinaryInitialization
			mi, ok := c.readMessage(&ordinaryResponse)
			if !ok {
				return
			}

			ok, err = sigverify.VerifyEllipticCurveSignature(c.addr, []byte(siweMessage), ordinaryResponse.Signature[:])
			if !ok {
				c.writeError(mi, structures.ErrorInvalidSignature, err)
				return
			}
			c.userID = user.ID
		}

		c.writeMessage(true, mi, structures.Welcome{
			Message:  "Welcome!",
			LoggedIn: true,
			User: structures.User{
				CreatedAt:     user.CreatedAt,
				UpdatedAt:     user.UpdatedAt,
				ID:            user.ID,
				Address:       user.Address,
				Email:         user.Email,
				EmailVerified: user.EmailVerified,
			},
		})
	}

	for {
		mi, buf, ok := c.readMessageInfo()
		if !ok {
			return
		}

		switch mi.RequestID {
		case structures.RequestListFeeds:
			go c.handleListFeeds(mi, buf)
		case structures.RequestAddFeed:
			go c.handleAddFeed(mi, buf)
		case structures.RequestEditFeed:
			go c.handleEditFeed(mi, buf)
		case structures.RequestDeleteFeed:
			go c.handleDeleteFeed(mi, buf)
		default:
			c.writeMessage(false, mi, structures.ErrorMessage{
				Code:    structures.ErrorInvalidInputs,
				Message: "unimplemented request or invalid request",
			})
		}
	}
}
