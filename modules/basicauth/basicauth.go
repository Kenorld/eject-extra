package basicauth

import (
	"encoding/base64"
	"strconv"

	"time"

	"github.com/kenorld/eject-core"
)

type (
	encodedUser struct {
		HeaderValue string
		Username    string
		logged      bool
		expires     time.Time
	}
	encodedUsers []encodedUser

	basicAuthMiddleware struct {
		config Config
		// these are filled from the config.Users map at the startup
		auth             encodedUsers
		realmHeaderValue string
		expireEnabled    bool // if the config.Expires is a valid date, default disabled
	}
)

//

// New takes one parameter, the Config returns a HandlerFunc
// use: eject.UseFunc(New(...)), eject.Get(...,New(...),...)
func New(c Config) eject.HandlerFunc {
	b := &basicAuthMiddleware{config: DefaultConfig().MergeSingle(c)}
	b.init()
	return b.Serve
}

// Default takes one parameter, the users returns a HandlerFunc
// use: eject.UseFunc(Default(...)), eject.Get(...,Default(...),...)
func Default(users map[string]string) eject.HandlerFunc {
	c := DefaultConfig()
	c.Users = users
	return New(c)
}

//

// User returns the user from context key same as 'ctx.GetString("user")' but cannot be used by the developer, use the basicauth.Config.User func instead.
func (b *basicAuthMiddleware) User(ctx *eject.Context) string {
	return b.config.User(ctx)
}

func (b *basicAuthMiddleware) init() {
	// pass the encoded users from the user's config's Users value
	b.auth = make(encodedUsers, 0, len(b.config.Users))

	for k, v := range b.config.Users {
		fullUser := k + ":" + v
		header := "Basic " + base64.StdEncoding.EncodeToString([]byte(fullUser))
		b.auth = append(b.auth, encodedUser{HeaderValue: header, Username: k, logged: false, expires: DefaultExpireTime})
	}

	// set the auth realm header's value
	b.realmHeaderValue = "Basic realm=" + strconv.Quote(b.config.Realm)

	if b.config.Expires > 0 {
		b.expireEnabled = true
	}
}

func (b *basicAuthMiddleware) findAuth(headerValue string) (auth *encodedUser, found bool) {
	if len(headerValue) == 0 {
		return
	}

	for _, user := range b.auth {
		if user.HeaderValue == headerValue {
			auth = &user
			found = true
			break
		}
	}

	return
}

func (b *basicAuthMiddleware) askForCredentials(ctx *eject.Context) {
	ctx.SetHeader("WWW-Authenticate", b.realmHeaderValue)
	ctx.SetStatusCode(eject.StatusUnauthorized)
}

// Serve the actual middleware
func (b *basicAuthMiddleware) Serve(ctx *eject.Context) {

	if auth, found := b.findAuth(ctx.RequestHeader("Authorization")); !found {
		b.askForCredentials(ctx)
		// don't continue to the next handler
	} else {
		// all ok set the context's value in order to be getable from the next handler
		ctx.Set(b.config.ContextKey, auth.Username)
		if b.expireEnabled {

			if auth.logged == false {
				auth.expires = time.Now().Add(b.config.Expires)
				auth.logged = true
			}

			if time.Now().After(auth.expires) {
				b.askForCredentials(ctx) // ask for authentication again
				return
			}

		}
		ctx.Next() // continue
	}

}
