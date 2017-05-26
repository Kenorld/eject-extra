package jwt


import (
	"testing"
	"github.com/dgrijalva/jwt-go"
	jwtmiddleware "github.com/eject-contrib/middleware/jwt"
	"github.com/kenorld/eject-core"
)

type Response struct {
	Text string `json:"text"`
}

func TestBasicJwt(t *testing.T) {
	var (
		api             = eject.New()
		myJwtMiddleware = jwtmiddleware.New(jwtmiddleware.Config{
			ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
				return []byte("My Secret"), nil
			},
			SigningMethod: jwt.SigningMethodHS256,
		})
	)

	securedPingHandler := func(ctx *eject.Context) {
		userToken := myJwtMiddleware.Get(ctx)
		var claimTestedValue string
		if claims, ok := userToken.Claims.(jwt.MapClaims); ok && userToken.Valid {
			claimTestedValue = claims["foo"].(string)
		} else {
			claimTestedValue = "Claims Failed"
		}

		response := Response{"Iauthenticated" + claimTestedValue}
		// get the *jwt.Token which contains user information using:
		// user:= myJwtMiddleware.Get(ctx) or context.Get("jwt").(*jwt.Token)

		ctx.JSON(eject.StatusOK, response)
	}

	api.Get("/secured/ping", myJwtMiddleware.Serve, securedPingHandler)
	e := api.Tester(t)

	e.GET("/secured/ping").Expect().Status(eject.StatusUnauthorized)

	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"foo": "bar",
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString([]byte("My Secret"))

	e.GET("/secured/ping").WithHeader("Authorization", "Bearer "+tokenString).Expect().Status(eject.StatusOK).Body().Contains("Iauthenticated").Contains("bar")

}