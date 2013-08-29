package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	_ "github.com/mattn/go-sqlite3"
	"github.com/hoisie/web"
	"log"
	"os"
	"regexp"
	"strings"
)

type Status struct {
	Events []Event `json:"events"`
}

type Event struct {
	Id      int      `json:"event_id"`
	Message *Message `json:"message"`
}

type Message struct {
	Id              string `json:"id"`
	Room            string `json:"room"`
	PublicSessionId string `json:"public_session_id"`
	IconUrl         string `json:"icon_url"`
	Type            string `json:"type"`
	SpeakerId       string `json:"speaker_id"`
	Nickname        string `json:"nickname"`
	Text            string `json:"text"`
}

type Karuta struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func defaultAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		return ":80"
	}
	return ":" + port
}

var addr = flag.String("addr", defaultAddr(), "server address")

var reCheck = regexp.MustCompile(`^[あ-ん]$`)
var reUpdate = regexp.MustCompile(`^!vim-karuta\s+(\S+)\s+(.+)$`)
var reQuery = regexp.MustCompile(`^?vim-karuta\s+(\S+)$`)

func main() {
	flag.Parse()

	db, err := sql.Open("sqlite3", "karuta.db")

	_, err = db.Exec("create table karuta (key varchar not null primary key, value varchar not null);")
	if err != nil {
		log.Println(err)
	}

	web.Get("/", func(ctx *web.Context) string {
		ctx.SetHeader("Content-Type", "text/plain; charset=utf-8", true)
		rows, err := db.Query("select key, value from karuta order by key")
		if err != nil {
			log.Println(err)
			return ""
		}
		ret := ""
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			ret += key + ": " + value + "\n"
		}
		rows.Close()
		return ret
	})
	web.Get("/json", func(ctx *web.Context) {
		ctx.SetHeader("Content-Type", "application/json; charset=utf-8", true)
		rows, err := db.Query("select key, value from karuta order by key")
		if err != nil {
			log.Println(err)
			return
		}
		ret := make([]Karuta, 0)
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			ret = append(ret, Karuta{key, value})
		}
		rows.Close()
		json.NewEncoder(ctx).Encode(ret)
	})
	web.Post("/lingr", func(ctx *web.Context) string {
		var status Status
		err := json.NewDecoder(ctx.Request.Body).Decode(&status)
		if err != nil {
			return ""
		}
		ret := ""
		for _, event := range status.Events {
			text := event.Message.Text
			tokens := reUpdate.FindStringSubmatch(text)
			if len(tokens) == 3 {
				if !reCheck.MatchString(tokens[1]) {
					ret += "お前いい加減にしろよ\n"
				} else {
					rows, err := db.Query("select key, value from karuta where key = ?", tokens[1])
					if err != nil {
						log.Println(err)
					} else {
						exists := rows.Next()
						rows.Close()
						if exists {
							_, err = db.Exec("update karuta set value = ? where key = ?", tokens[2], tokens[1])
							if err != nil {
								log.Println(err)
							} else {
								ret += "更新しました\n"
							}
						} else {
							_, err = db.Exec("insert into karuta(key, value) values (?, ?)", tokens[1], tokens[2])
							if err != nil {
								log.Println(err)
							} else {
								ret += "登録しました\n"
							}
						}
						rows.Close()
					}
				}
			}
			tokens = reQuery.FindStringSubmatch(text)
			if len(tokens) == 2 {
				if !reCheck.MatchString(tokens[1]) {
					ret += "お前いい加減にしろよ\n"
				} else {
					rows, err := db.Query("select key, value from karuta where key = ?", tokens[1])
					if err != nil {
						log.Println(err)
					} else {
						if rows.Next() {
							var key, value string
							rows.Scan(&key, &value)
							ret += key + ": " + value + "\n"
						}
						rows.Close()
					}

				}
			}
		}
		if len(ret) > 0 {
			ret = strings.TrimRight(ret, "\n")
			if runes := []rune(ret); len(runes) > 1000 {
				ret = string(runes[0:999])
			}
		}
		return ret
	})
	web.Run(*addr)
}
