package reader

import (
	"fmt"
	"time"
	"bytes"
	"net/http"
	"text/template"
	"encoding/json"
	"encoding/xml"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"appengine/user"
	"appengine/memcache"

	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
	rss "github.com/ungerik/go-rss"
)

var indexTemplate = template.Must(template.New("").ParseFiles("template/index.html"))

func init() {
	http.HandleFunc("/",       handleRoot)
	http.HandleFunc("/add",    handleAdd)
	http.HandleFunc("/list",   handleList)
	http.HandleFunc("/get",    handleGet)
	http.HandleFunc("/delete", handleDelete)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	c := appengine.NewContext(r)
	u := user.Current(c)

	if u == nil {
		url, _ := user.LoginURL(c, "/")
		http.Redirect(w, r, url, http.StatusOK)
	} else {
		url, _ := user.LogoutURL(c, "/")
		input := map[string]string{
			"LogoutURL": url,
			"Email":     u.Email,
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

	info := UserInfo{}
	if err := datastore.Get(c, key, &info); err != nil {
		if err != datastore.ErrNoSuchEntity {
			return err
		}
	}
	info.LastLogin = time.Now()
	datastore.Put(c, key, &info)
	return nil
}

type subscription struct {
	Title      string    `json:"title"`
	XMLUrl     string    `json:"xmlUrl"`
	UpdateTime time.Time `json:"updateTime, string"`
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	url := r.FormValue("xmlUrl")
	channel, err := rssGet(c, url)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}

	sub := subscription{}
	u := user.Current(c)
	key := datastore.NewKey(c, u.Email+".Subscriptions", channel.Title, 0, nil)
	if err := datastore.Get(c, key, &sub); err != nil {
		// Not found, now put it in datastore.
		if err == datastore.ErrNoSuchEntity {
			sub.Title      = channel.Title
			sub.XMLUrl     = url
			sub.UpdateTime = time.Now()
			datastore.Put(c, key, &sub)
			json.NewEncoder(w).Encode(&sub)
		} else {
			http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		}
	} else {
		// You have this subscription.
		http.Error(w, "Duplicated subscription: " + channel.Title,
			http.StatusInternalServerError)
	}
}

func handleList(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)

	//var sub []subscription
	var sub []subscription
	_, err := datastore.NewQuery(u.Email+".Subscriptions").GetAll(c, &sub)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(&sub)
}

// Cached read rss
func rssCachedGet(c appengine.Context, url string) ([]byte, error) {
	item, err := memcache.Get(c, url);
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}

	// Cache hit
	if err == nil {
		return item.Value, err
	}

	// Cache miss
	channel, err := rssGet(c, url)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	json.NewEncoder(buf).Encode(channel)
	expriation, err := time.ParseDuration("5m")
	if err != nil {
		return nil, err
	}
	item = &memcache.Item {
		Key: url,
		Value: buf.Bytes(),
		Expiration: expriation,
	}
	return item.Value, memcache.Set(c, item)
}

// Get rss xml data use appengine.Urlfetch only.
func rssGet(c appengine.Context, url string) (*rss.Channel, error) {
	client := urlfetch.Client(c)
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

func handleGet(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	title := r.FormValue("title")
	sub := subscription{}
	u := user.Current(c)
	key := datastore.NewKey(c, u.Email+".Subscriptions", title, 0, nil)
	if err := datastore.Get(c, key, &sub); err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}

	if channel, err := rssCachedGet(c, sub.XMLUrl); err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
	} else {
		w.Write(channel)
	}
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	title := r.FormValue("title")
	u := user.Current(c)
	key := datastore.NewKey(c, u.Email+".Subscriptions", title, 0, nil)
	if err := datastore.Delete(c, key); err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return	
	}

	fmt.Fprintf(w, `{ Status: "ok", Title: "%s" }`, title)
}