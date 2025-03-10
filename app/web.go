package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"regexp"

	"github.com/gorilla/mux"
	"net/http"
)

type ChangePasswordResponse struct {
	Alerts Alert `json:"alerts"`
}

type Alert struct {
	Success []string `json:"success,omitempty"`
	Error   []string `json:"error,omitempty"`
}

type ChangePasswordRequest struct {
	Username        string `json:"username"`
	OldPassword     string `json:"oldPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

func Serve() {
	router := mux.NewRouter()
	router.HandleFunc("/ldap-passwd-api/", ChangePassword).Methods("POST")
	router.HandleFunc("/ldap-passwd-api/health", HealthCheck).Methods("GET")
	fmt.Println("Starting server on port 8044")

	http.ListenAndServe(":8044", router)
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("OK"))
	if err != nil {
		fmt.Println(err)
		return
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

	alerts := Alert{}

	if cp.Username == "" {
		alerts.Error = append(alerts.Error, "Username not specified.")
	}
	if cp.OldPassword == "" {
		alerts.Error = append(alerts.Error, "Old password not specified.")
	}
	if cp.NewPassword == "" {
		alerts.Error = append(alerts.Error, "New password not specified.")
	}
	if cp.ConfirmPassword == "" {
		alerts.Error = append(alerts.Error, "Confirmation password not specified.")
	}

	checkInjection(cp.Username, alerts, "username")
	checkInjection(cp.OldPassword, alerts, "old password")
	checkInjection(cp.NewPassword, alerts, "new password")
	checkInjection(cp.ConfirmPassword, alerts, "confirm password")
	if len(cp.ConfirmPassword) >= 1 && len(cp.NewPassword) >= 1 && strings.Compare(cp.NewPassword, cp.ConfirmPassword) != 0 {
		alerts.Error = append(alerts.Error, "New and confirmation passwords does not match.")
	}

	upperCase := regexp.MustCompile(`[A-Z]`)
	lowerCase := regexp.MustCompile(`[a-z]`)
	digit := regexp.MustCompile(`[0-9]`)
	specialChars := regexp.MustCompile(`[!@#$&*]`)
	if !upperCase.MatchString(cp.NewPassword) ||
		!lowerCase.MatchString(cp.NewPassword) ||
		!digit.MatchString(cp.NewPassword) ||
		!specialChars.MatchString(cp.NewPassword) ||
		len(cp.NewPassword) <= 12 {
		alerts.Error = append(alerts.Error, fmt.Sprintf("%s", getPatternInfo()))
	}
	regex := regexp.MustCompile(`[^A-Za-z0-9!@#$&*]`)
	if regex.MatchString(cp.NewPassword) {
		alerts.Error = append(alerts.Error, "You can't use special characters other than \"!@#$&*\"")
	}

	if len(alerts.Error) == 0 {
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
			alerts.Error = append(alerts.Error, regex.ReplaceAllString(stderr.String(), "$1"))
		} else {
			msg := fmt.Sprintf("Password has been changed successfully for %s", cp.Username)
			alerts.Success = append(alerts.Success, msg)
			fmt.Println(msg)
		}
	}
	p := &ChangePasswordResponse{Alerts: alerts}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(p)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func checkInjection(s string, alerts Alert, name string) {
	if m, _ := regexp.MatchString(`^[a-zA-Z0-9!@#$&*]{1,22}$`, s); !m {
		alerts.Error = append(alerts.Error, fmt.Sprintf("Wrong format for %s", name))
	}
}
