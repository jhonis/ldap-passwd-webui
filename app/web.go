package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/thammuio/ldap-passwd-webui/app"
	"log"
	"os/exec"
	"path"
	"strings"

	"html/template"

	"github.com/dchest/captcha"

	"regexp"

	"net/http"
)

type route struct {
	pattern *regexp.Regexp
	verb    string
	handler http.Handler
}

// RegexpHandler is used for http handler to bind using regular expressions
type RegexpHandler struct {
	routes []*route
}

// Handler binds http handler on RegexpHandler
func (h *RegexpHandler) Handler(pattern *regexp.Regexp, verb string, handler http.Handler) {
	h.routes = append(h.routes, &route{pattern, verb, handler})
}

// HandleFunc binds http handler function on RegexpHandler
func (h *RegexpHandler) HandleFunc(r string, v string, handler func(http.ResponseWriter, *http.Request)) {
	re := regexp.MustCompile(r)
	h.routes = append(h.routes, &route{re, v, http.HandlerFunc(handler)})
}

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) && route.verb == r.Method {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

type pageData struct {
	Title       string
	Pattern     string
	PatternInfo string
	Username    string
	Alerts      map[string]string
	CaptchaId   string
}

type ChangePasswordRequest struct {
	Username        string `json:"username"`
	OldPassword     string `json:"oldPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

func Serve() {
	reHandler := new(app.RegexpHandler)

	reHandler.HandleFunc(".*.[js|css|png|eof|svg|ttf|woff]", "GET", app.ServeAssets)
	reHandler.HandleFunc("/", "GET", ServeIndex)
	reHandler.HandleFunc("/", "POST", ChangePassword)
	http.Handle("/captcha/", captcha.Server(captcha.StdWidth, captcha.StdHeight))
	http.Handle("/", reHandler)
	fmt.Println("Starting server on port 8443")
	http.ListenAndServe(":8443", nil)
}

// ServeAssets : Serves the static assets
func ServeAssets(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, path.Join("static", req.URL.Path[1:]))
}

// ServeIndex : Serves index page on GET request
func ServeIndex(w http.ResponseWriter, req *http.Request) {
	p := &pageData{Title: getTitle(), CaptchaId: captcha.New(), Pattern: getPattern(), PatternInfo: getPatternInfo()}
	index, err := template.ParseFiles(path.Join("templates", "index.html"))
	if err != nil {
		log.Printf("Error parsing file %v\n", err)
	} else {
		index.Execute(w, p)
	}
	main, err := template.ParseFiles(path.Join("templates", "main.html"))
	if err != nil {
		log.Printf("Error parsing file %v\n", err)
	} else {
		main.Execute(w, p)
	}
	afterMain, err := template.ParseFiles(path.Join("templates", "after-main.html"))
	if err != nil {
		log.Printf("Error parsing file %v\n", err)
	} else {
		afterMain.Execute(w, p)
	}
}

// ChangePassword : Serves index page on POST request - executes the change
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	var cp ChangePasswordRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&cp)
	if err != nil {
		fmt.Println(err)
		return
	}

	alerts := map[string]string{}

	if cp.Username == "" {
		alerts["error"] = "Username not specified."
	}
	if cp.OldPassword == "" {
		alerts["error"] = alerts["error"] + "Old password not specified."
	}
	if cp.NewPassword == "" {
		alerts["error"] = alerts["error"] + "New password not specified."
	}
	if cp.ConfirmPassword == "" {
		alerts["error"] = alerts["error"] + "Confirmation password not specified."
	}

	if len(cp.ConfirmPassword) >= 1 && len(cp.NewPassword) >= 1 && strings.Compare(cp.NewPassword, cp.ConfirmPassword) != 0 {
		alerts["error"] = alerts["error"] + "New and confirmation passwords does not match."
	}

	if m, _ := regexp.MatchString(getPattern(), cp.NewPassword); !m {
		alerts["error"] = alerts["error"] + fmt.Sprintf("%s", getPatternInfo())
	}

	if len(alerts) == 0 {
		args := fmt.Sprintf(`Set-ADAccountPassword -Identity %s -OldPassword (ConvertTo-SecureString -AsPlainText "%s" -Force) -NewPassword (ConvertTo-SecureString -AsPlainText "%s" -Force)`, cp.Username, cp.OldPassword, cp.NewPassword)
		cmd := exec.Command("powershell", args)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		fmt.Println(stdout.String())
		fmt.Println(stderr.String())
		if err != nil {
			fmt.Println(err)
			regex := regexp.MustCompile(`Set-ADAccountPassword : (.*)\n(?s:.*)`)
			fmt.Println(regex.ReplaceAllString(stderr.String(), "$1"))
			fmt.Println(stderr)
			alerts["error"] = alerts["error"] + regex.ReplaceAllString(stderr.String(), "$1")
		} else {
			msg := fmt.Sprintf("Password has been changed successfully for %s", cp.Username)
			alerts["success"] = msg
			fmt.Println(msg)
		}
	}

	p := &pageData{Title: getTitle(), Alerts: alerts, Username: cp.Username, CaptchaId: captcha.New(), Pattern: getPattern(), PatternInfo: getPatternInfo()}

	main, err := template.ParseFiles(path.Join("templates", "main.html"))
	if err != nil {
		log.Printf("Error parsing file %v\n", err)
	} else {
		main.Execute(w, p)
	}
}
