package app

import (
	"encoding/json"
	"fmt"
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

// ServeAssets : Serves the static assets
func ServeAssets(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, path.Join("static", req.URL.Path[1:]))
}

// ServeIndex : Serves index page on GET request
func ServeIndex(w http.ResponseWriter, req *http.Request) {
	p := &pageData{Title: getTitle(), CaptchaId: captcha.New(), Pattern: getPattern(), PatternInfo: getPatternInfo()}
	t, e := template.ParseFiles(path.Join("templates", "index.html"))
	if e != nil {
		log.Printf("Error parsing file %v\n", e)
	} else {
		t.Execute(w, p)
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
		args := fmt.Sprintf(`-nologo -noprofile Set-ADAccountPassword -Identity %s -OldPassword (ConvertTo-SecureString -AsPlainText "%s" -Force) -NewPassword (ConvertTo-SecureString -AsPlainText "%s" -Force)`, cp.Username, cp.OldPassword, cp.NewPassword)
		out, err := exec.Command("powershell", strings.Split(args, " ")...).Output()
		if err != nil {
			fmt.Println(err)
			alerts["error"] = alerts["error"] + err.Error()
			return
		}
		fmt.Println(string(out))
		fmt.Println(fmt.Sprintf("Password has been changed successfully for %s", cp.Username))
	}

	p := &pageData{Title: getTitle(), Alerts: alerts, Username: cp.Username, CaptchaId: captcha.New()}

	t, e := template.ParseFiles(path.Join("templates", "index.html"))
	if e != nil {
		log.Printf("Error parsing file %v\n", e)
	} else {
		t.Execute(w, p)
	}
}
