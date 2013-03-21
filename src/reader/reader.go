package reader

import (
	"fmt"
	"time"
	"net/http"
	"text/template"
	"encoding/xml"

	"appengine"
	"appengine/user"
	"appengine/datastore"
	"appengine/urlfetch"

	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
	rss "github.com/ungerik/go-rss"
)

var indexTemplate = template.Must(template.New("").ParseFiles("template/index.html"))

func init() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/add", handleAdd)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprint(w, "shit");
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	c := appengine.NewContext(r)
	u := user.Current(c)

	if u == nil {
		url, _ := user.LoginURL(c, "/")
		http.Redirect(w, r, url, http.StatusOK)
	} else {
		url, _ := user.LogoutURL(c, "/")
		input := map[string]string {
			"LogoutURL": url,
			"Email": u.Email,
		}
		err := checkOrNewUser(c, u.Email)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		}
		err = indexTemplate.ExecuteTemplate(w, "index.html", input)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		}
	}
}

type UserInfo struct {
	LastLogin time.Time

}

func checkOrNewUser(c appengine.Context, name string) error {
	key := datastore.NewKey(c, "User", name, 0, nil)

	info := UserInfo {}
	if err := datastore.Get(c, key, &info); err != nil {
		if err != datastore.ErrNoSuchEntity {
			return err
		}
	}
	info.LastLogin = time.Now()
	datastore.Put(c, key, &info)
	return nil
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	// if rss, err := r.FormValue("xmlUrl"); err != nil {
	// 	http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
	// 	return
	// }
	url := r.FormValue("xmlUrl")
	// fmt.Print(rss)

	channel, err := rssGet(c, url)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, `{ title: "%s", xmlUrl: "%s" }`, channel.Title, url)
}

func rssGet(c appengine.Context, url string) (*rss.Channel, error) {
	client := urlfetch.Client(c);
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	xmlDecoder := xml.NewDecoder(response.Body)
	xmlDecoder.CharsetReader = charset.NewReader

	var rssInput struct {
		Channel rss.Channel `xml:"channel"`
	}
	err = xmlDecoder.Decode(&rssInput)
	if err != nil {
		return nil, err
	}
	return &rssInput.Channel, nil
}