package main

import (
	"fmt"
	"ldap-passwd-webui/app"
	"net/http"

	"github.com/dchest/captcha"
)

func main() {
	reHandler := new(app.RegexpHandler)

	reHandler.HandleFunc(".*.[js|css|png|eof|svg|ttf|woff]", "GET", app.ServeAssets)
	reHandler.HandleFunc("/", "GET", app.ServeIndex)
	reHandler.HandleFunc("/", "POST", app.ChangePassword)
	http.Handle("/captcha/", captcha.Server(captcha.StdWidth, captcha.StdHeight))
	http.Handle("/", reHandler)
	fmt.Println("Starting server on port 8443")
	http.ListenAndServe(":8443", nil)
}
