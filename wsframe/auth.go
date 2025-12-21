package wsframe

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

type Auth struct {
	Alias     string `json:"alias,omitempty"`
	Passhash  string `json:"passhash"`
	Salt      string `json:"salt"`
	Hash      string `json:"hash"`
	Privilege string `json:"privilege"`
}

func (wfa *App) ReadCookie(r *http.Request) map[string]string {
	var value map[string]string
	cookie, err := r.Cookie(wfa.Name)
	if err != nil || wfa.SecureCookie.Decode(wfa.Name, cookie.Value, &value) != nil {
		return map[string]string{}
	}
	return value
}

func (wfa *App) SetCookie(w http.ResponseWriter, r *http.Request, value map[string]string, logout bool) {
	encoded, err := wfa.SecureCookie.Encode(wfa.Name, value)
	if err != nil {
		return
	}
	cookie := &http.Cookie{
		Name:     wfa.Name,
		Value:    encoded,
		Domain:   "localhost",
		Path:     "/",
		HttpOnly: true,
	}
	if logout {
		cookie.Expires = time.Now().Add(-24 * time.Hour)
		cookie.MaxAge = -1
	} else {
		cookie.Expires = time.Now().Add(24 * time.Hour)
		cookie.MaxAge = 60 * 60 * 24
	}
	http.SetCookie(w, cookie)
}

func (wfa *App) register(w http.ResponseWriter, r *http.Request) {
    var existing []byte
	alias := r.FormValue("alias")
	passhash := r.FormValue("passhash")
	if err := wfa.Driver.QueryRow(`SELECT value FROM auth WHERE key = $1`, alias).Scan(&existing); err != sql.ErrNoRows {
		log.Println("Alias already exists or DB error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	random := make([]byte, 16)
    if _, err := rand.Read(random); err != nil {
		log.Println("Salt generation failed:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	salt := fmt.Sprintf("%x", sha1.Sum(random))
	auth := Auth{
		Passhash:  passhash,
		Salt:      salt,
		Hash:      fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s%s%s", alias, passhash, salt)))),
		Privilege: "user",
	}
    data, err := json.Marshal(&auth)
	if err != nil {
		log.Println("Marshal error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err = wfa.Driver.Exec(fmt.Sprintf(`INSERT INTO auth (key, value) VALUES ($1, $2)`), alias, data); err != nil {
		log.Println("Insert error:", err)
		w.WriteHeader(http.StatusInternalServerError)
        return
	}
    wfa.SetCookie(w, r, map[string]string{
        "alias":     alias,
        "privilege": auth.Privilege,
    }, false)
    w.WriteHeader(http.StatusOK)
}

func (wfa *App) login(w http.ResponseWriter, r *http.Request) {
    var data []byte
	var auth Auth
	alias := r.FormValue("alias")
	passhash := r.FormValue("passhash")
	if err := wfa.Driver.QueryRow(`SELECT value FROM auth WHERE key = $1`, alias).Scan(&data); err != nil {
		log.Println("User not found or DB error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := json.Unmarshal(data, &auth); err != nil {
		log.Println("Unmarshal error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(alias + passhash + auth.Salt)))
	if hash != auth.Hash {
		w.WriteHeader(http.StatusInternalServerError)
        return
    }
    wfa.SetCookie(w, r, map[string]string{
        "alias":     alias,
        "privilege": auth.Privilege,
    }, false)
    w.WriteHeader(http.StatusOK)
}

func (wfa *App) logout(w http.ResponseWriter, r *http.Request) {
	wfa.SetCookie(w, r, nil, true)
	w.WriteHeader(http.StatusOK)
}
